package io

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/grafana/grafanactl/internal/format"
	"github.com/spf13/pflag"
)

type Options struct {
	OutputFormat string

	customFormats map[string]format.Formatter
	defaultFormat string
}

func (opts *Options) RegisterCustomFormat(name string, formatFunc format.Formatter) {
	if opts.customFormats == nil {
		opts.customFormats = make(map[string]format.Formatter)
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

func (opts *Options) Format(out io.Writer, input any) error {
	formatterFunc := opts.formatterFor(opts.OutputFormat)
	if formatterFunc == nil {
		return fmt.Errorf("unknown output format '%s'. Valid formats are: %s", opts.OutputFormat, strings.Join(opts.allowedFormats(), ", "))
	}

	return formatterFunc(out, input)
}

func (opts *Options) formatterFor(format string) format.Formatter {
	if opts.customFormats != nil && opts.customFormats[format] != nil {
		return opts.customFormats[format]
	}

	return opts.builtinFormatters()[format]
}

func (opts *Options) builtinFormatters() map[string]format.Formatter {
	return map[string]format.Formatter{
		"yaml": format.YAML,
		"json": format.JSON,
	}
}

func (opts *Options) allowedFormats() []string {
	allowedFormats := slices.Collect(maps.Keys(opts.builtinFormatters()))
	for name := range opts.customFormats {
		allowedFormats = append(allowedFormats, name)
	}

	// the allowed formats are stored in a map: let's sort them to make the
	// return value of this function deterministic
	sort.Strings(allowedFormats)

	return allowedFormats
}
