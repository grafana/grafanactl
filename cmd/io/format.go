package io

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/pflag"
)

type formatter func(any) ([]byte, error)

type Options struct {
	OutputFormat string
}

func (opts *Options) BindFlags(flags *pflag.FlagSet) {
	allowedFormats := slices.Collect(maps.Keys(opts.formatters()))

	flags.StringVarP(&opts.OutputFormat, "output", "o", "yaml", "Output format. One of: "+strings.Join(allowedFormats, ", "))
}

func (opts *Options) Validate() error {
	_, ok := opts.formatters()[opts.OutputFormat]
	if !ok {
		allowedFormats := slices.Collect(maps.Keys(opts.formatters()))
		return fmt.Errorf("unknown output format '%s'. Valid formats are: %s", opts.OutputFormat, strings.Join(allowedFormats, ", "))
	}

	return nil
}

func (opts *Options) Format(input any, out io.Writer) error {
	formatterFunc, ok := opts.formatters()[opts.OutputFormat]
	if !ok {
		allowedFormats := slices.Collect(maps.Keys(opts.formatters()))
		return fmt.Errorf("unknown output format '%s'. Valid formats are: %s", opts.OutputFormat, strings.Join(allowedFormats, ", "))
	}

	formatted, err := formatterFunc(input)
	if err != nil {
		return err
	}

	_, err = out.Write(formatted)

	return err
}

func (opts *Options) formatters() map[string]formatter {
	return map[string]formatter{
		"yaml": formatYAML,
		"json": formatJSON,
	}
}

func formatYAML(input any) ([]byte, error) {
	return yaml.MarshalWithOptions(
		input,
		yaml.Indent(2),
		yaml.CustomMarshaler[[]byte](func(data []byte) ([]byte, error) {
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)

			return dst, nil
		}),
	)
}

func formatJSON(input any) ([]byte, error) {
	formatted, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(formatted, '\n'), nil
}
