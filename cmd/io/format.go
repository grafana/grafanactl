package io

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/pflag"
)

type formatter func(output io.Writer, data any) error

type Options struct {
	OutputFormat string

	customFormats map[string]formatter
	defaultFormat string
}

func (opts *Options) RegisterCustomFormat(name string, formatFunc formatter) {
	if opts.customFormats == nil {
		opts.customFormats = make(map[string]formatter)
	}

	opts.customFormats[name] = formatFunc
}

func (opts *Options) DefaultFormat(name string) {
	opts.defaultFormat = name
}

func (opts *Options) BindFlags(flags *pflag.FlagSet) {
	defaultFormat := "yaml"
	if opts.defaultFormat != "" {
		defaultFormat = opts.defaultFormat
	}

	flags.StringVarP(&opts.OutputFormat, "output", "o", defaultFormat, "Output format. One of: "+strings.Join(opts.allowedFormats(), ", "))
}

func (opts *Options) Validate() error {
	formatterFunc := opts.formatterFor(opts.OutputFormat)
	if formatterFunc == nil {
		return fmt.Errorf("unknown output format '%s'. Valid formats are: %s", opts.OutputFormat, strings.Join(opts.allowedFormats(), ", "))
	}

	return nil
}

func (opts *Options) Format(input any, out io.Writer) error {
	formatterFunc := opts.formatterFor(opts.OutputFormat)
	if formatterFunc == nil {
		return fmt.Errorf("unknown output format '%s'. Valid formats are: %s", opts.OutputFormat, strings.Join(opts.allowedFormats(), ", "))
	}

	return formatterFunc(out, input)
}

func (opts *Options) formatterFor(format string) formatter {
	if opts.customFormats != nil && opts.customFormats[format] != nil {
		return opts.customFormats[format]
	}

	return opts.builtinFormatters()[format]
}

func (opts *Options) builtinFormatters() map[string]formatter {
	return map[string]formatter{
		"yaml": formatYAML,
		"json": formatJSON,
	}
}

func (opts *Options) allowedFormats() []string {
	allowedFormats := slices.Collect(maps.Keys(opts.builtinFormatters()))
	for format := range opts.customFormats {
		allowedFormats = append(allowedFormats, format)
	}

	// the allowed formats are stored in a map: let's sort them to make the
	// return value of this function deterministic
	sort.Strings(allowedFormats)

	return allowedFormats
}

func formatYAML(output io.Writer, input any) error {
	encoder := yaml.NewEncoder(
		output,
		yaml.Indent(2),
		yaml.IndentSequence(true),
		yaml.UseJSONMarshaler(),
		yaml.CustomMarshaler[[]byte](func(data []byte) ([]byte, error) {
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)

			return dst, nil
		}),
	)

	return encoder.Encode(input)
}

func formatJSON(output io.Writer, input any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	return encoder.Encode(input)
}
