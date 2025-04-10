package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type formatParser func(resources *resources.Resources, input io.Reader) error

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
			"json": func(results *resources.Resources, input io.Reader) error {
				resource := &unstructured.Unstructured{}
				if err := json.NewDecoder(input).Decode(resource); err != nil {
					return err
				}

				metaAccessor, err := utils.MetaAccessor(resource)
				if err != nil {
					return err
				}

				results.Add(&resources.Resource{Raw: metaAccessor})

				return nil
			},
			"yaml": func(results *resources.Resources, input io.Reader) error {
				decoder := yaml.NewDecoder(input, yaml.Strict(), yaml.UseJSONUnmarshaler())
				resource := &unstructured.Unstructured{}
				if err := decoder.Decode(resource); err != nil {
					return err
				}

				metaAccessor, err := utils.MetaAccessor(resource)
				if err != nil {
					return err
				}

				results.Add(&resources.Resource{Raw: metaAccessor})

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

func (parser *Parser) ParseInto(resources *resources.Resources, resourcePath string) error {
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

func (parser *Parser) ParseBytesInto(resources *resources.Resources, raw []byte, format string) error {
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

func (parser *Parser) parseFileInto(results *resources.Resources, file string) error {
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

	tmpResources := resources.NewResources()

	if err := parserFunc(tmpResources, f); err != nil {
		return ParseError{File: file, Err: err}
	}

	_ = tmpResources.ForEach(func(resource *resources.Resource) error {
		properties, _ := resource.Raw.GetSourceProperties()
		properties.Path = fmt.Sprintf("%s://%s", format, file)

		resource.Raw.SetSourceProperties(properties)

		return nil
	})

	results.Merge(tmpResources)

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
