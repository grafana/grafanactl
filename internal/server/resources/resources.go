package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/go-multierror"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourceRef string

type Resource struct {
	Raw utils.GrafanaMetaAccessor
}

func (r Resource) Ref() ResourceRef {
	return ResourceRef(fmt.Sprintf("%s/%s:%s-%s", r.APIVersion(), r.Kind(), r.Namespace(), r.Name()))
}

func (r Resource) Namespace() string {
	return r.Raw.GetNamespace()
}

func (r Resource) Name() string {
	return r.Raw.GetName()
}

func (r Resource) Group() string {
	return r.Raw.GetGroupVersionKind().Group
}

func (r Resource) Kind() string {
	return r.Raw.GetGroupVersionKind().Kind
}

func (r Resource) Version() string {
	return r.Raw.GetGroupVersionKind().Version
}

func (r Resource) APIVersion() string {
	return r.Group() + "/" + r.Version()
}

func (r Resource) SourcePath() string {
	properties, _ := r.Raw.GetSourceProperties()
	if properties.Path == "" {
		return ""
	}

	u, err := url.Parse(properties.Path)
	if err != nil {
		return ""
	}

	return filepath.Join(u.Host, u.Path)
}

func (r Resource) SourceFormat() string {
	properties, _ := r.Raw.GetSourceProperties()
	if properties.Path == "" {
		return ""
	}

	u, err := url.Parse(properties.Path)
	if err != nil {
		return ""
	}

	return u.Scheme
}

type Resources struct {
	collection    *orderedmap.OrderedMap[ResourceRef, *Resource]
	onChangeFuncs []func(resource *Resource)
}

func NewResources(resources ...*Resource) *Resources {
	r := &Resources{
		collection: orderedmap.New[ResourceRef, *Resource](),
	}

	r.Add(resources...)

	return r
}

func (r *Resources) Add(resources ...*Resource) {
	for _, resource := range resources {
		r.collection.Set(resource.Ref(), resource)

		for _, cb := range r.onChangeFuncs {
			cb(resource)
		}
	}
}

func (r *Resources) OnChange(callback func(resource *Resource)) {
	r.onChangeFuncs = append(r.onChangeFuncs, callback)
}

// TODO: kind + name isn't enough to unambiguously identify a resource
func (r *Resources) Find(kind string, name string) (*Resource, bool) {
	for pair := r.collection.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.Kind() == kind && pair.Value.Name() == name {
			return pair.Value, true
		}
	}

	return nil, false
}

func (r *Resources) Merge(resources *Resources) {
	_ = resources.ForEach(func(resource *Resource) error {
		r.Add(resource)
		return nil
	})
}

func (r *Resources) ForEach(callback func(*Resource) error) error {
	for pair := r.collection.Oldest(); pair != nil; pair = pair.Next() {
		if err := callback(pair.Value); err != nil {
			return err
		}
	}

	return nil
}

func (r *Resources) Len() int {
	if r.collection == nil {
		return 0
	}

	return r.collection.Len()
}

func (r *Resources) AsList() []*Resource {
	if r.collection == nil {
		return nil
	}

	list := make([]*Resource, 0, r.Len())

	_ = r.ForEach(func(resource *Resource) error {
		list = append(list, resource)
		return nil
	})

	return list
}

func (r *Resources) GroupByKind() map[string]*Resources {
	resourceByKind := map[string]*Resources{}
	_ = r.ForEach(func(resource *Resource) error {
		if _, ok := resourceByKind[resource.Kind()]; !ok {
			resourceByKind[resource.Kind()] = NewResources()
		}

		resourceByKind[resource.Kind()].Add(resource)
		return nil
	})

	return resourceByKind
}

type formatParser func(resources *Resources, input io.Reader) error

type ParseError struct {
	File string
	Err  error
}

func (err ParseError) Error() string {
	return fmt.Sprintf("parse error in '%s': %s", err.File, err.Err)
}

type UnrecognisedFormatError struct {
	File   string
	Format string
}

func (e UnrecognisedFormatError) Error() string {
	if e.Format != "" {
		return "unrecognized format " + e.Format
	}

	return "unrecognized format for " + e.File
}

type ParserConfig struct {
	ContinueOnError bool
}

func DefaultParser(ctx context.Context, config ParserConfig) *Parser {
	return &Parser{
		logger: logging.FromContext(ctx).With(slog.String("component", "parser")),
		parsers: map[string]formatParser{
			"json": func(resources *Resources, input io.Reader) error {
				resource := &unstructured.Unstructured{}
				if err := json.NewDecoder(input).Decode(resource); err != nil {
					return err
				}

				metaAccessor, err := utils.MetaAccessor(resource)
				if err != nil {
					return err
				}

				resources.Add(&Resource{Raw: metaAccessor})

				return nil
			},
			"yaml": func(resources *Resources, input io.Reader) error {
				decoder := yaml.NewDecoder(input, yaml.Strict(), yaml.UseJSONUnmarshaler())
				resource := &unstructured.Unstructured{}
				if err := decoder.Decode(resource); err != nil {
					return err
				}

				metaAccessor, err := utils.MetaAccessor(resource)
				if err != nil {
					return err
				}

				resources.Add(&Resource{Raw: metaAccessor})

				return nil
			},
		},
		continueOnError: config.ContinueOnError,
	}
}

type Parser struct {
	logger          logging.Logger
	parsers         map[string]formatParser
	continueOnError bool
}

func (parser *Parser) ParseInto(resources *Resources, resourcePath string) error {
	stat, err := os.Stat(resourcePath)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return parser.parseFileInto(resources, resourcePath)
	}

	var finalErr error
	_ = filepath.WalkDir(resourcePath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if err := parser.parseFileInto(resources, path); err != nil {
			finalErr = multierror.Append(finalErr, err)

			if !parser.continueOnError {
				return err
			}
		}

		return nil
	})

	return finalErr
}

func (parser *Parser) ParseBytesInto(resources *Resources, raw []byte, format string) error {
	parser.logger.Debug("Parsing bytes", slog.String("format", format))

	parserFunc, err := parser.parserForFormat(format)
	if err != nil {
		return err
	}

	if err := parserFunc(resources, bytes.NewBuffer(raw)); err != nil {
		return ParseError{Err: err}
	}

	return nil
}

func (parser *Parser) parseFileInto(resources *Resources, file string) error {
	format := strings.TrimPrefix(path.Ext(file), ".")

	parser.logger.Debug("Parsing file", slog.String("file", file), slog.String("format", format))

	parserFunc, err := parser.parserForFormat(format)
	if err != nil {
		return err
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpResources := NewResources()

	if err := parserFunc(tmpResources, f); err != nil {
		return ParseError{File: file, Err: err}
	}

	_ = tmpResources.ForEach(func(resource *Resource) error {
		properties, _ := resource.Raw.GetSourceProperties()
		properties.Path = fmt.Sprintf("%s://%s", format, file)

		resource.Raw.SetSourceProperties(properties)

		return nil
	})

	resources.Merge(tmpResources)

	return nil
}

func (parser *Parser) parserForFormat(format string) (formatParser, error) {
	switch format {
	case "json":
		return parser.parsers["json"], nil
	case "yaml", "yml":
		return parser.parsers["yaml"], nil
	default:
		return nil, UnrecognisedFormatError{Format: format}
	}
}
