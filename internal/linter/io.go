package linter

import (
	"fmt"
	"io/fs"
	"log"
	"strings"

	gbundle "github.com/grafana/grafanactl/internal/linter/bundle"
	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/loader/filter"
)

// builtinBundle contains a bundle with built-in rules and utilities as well as
// the linter's main logic.
//
//nolint:gochecknoglobals
var builtinBundle = MustLoadBundleFS(gbundle.BundleFS)

// LoadBundleFS loads a bundle from the given filesystem.
// Note: tests are excluded.
func LoadBundleFS(fs fs.FS) (opabundle.Bundle, error) {
	embedLoader, err := opabundle.NewFSLoader(fs)
	if err != nil {
		return opabundle.Bundle{}, fmt.Errorf("failed to load bundle from filesystem: %w", err)
	}

	return opabundle.NewCustomReader(embedLoader.WithFilter(excludeTestFilter())).
		WithCapabilities(ast.CapabilitiesForThisVersion()).
		WithSkipBundleVerification(true).
		WithProcessAnnotations(true).
		Read()
}

// MustLoadBundleFS implements the same functionality as LoadBundleFS, but logs
// an error on failure and exits.
func MustLoadBundleFS(fs fs.FS) opabundle.Bundle {
	bundle, err := LoadBundleFS(fs)
	if err != nil {
		log.Fatal(err)
	}

	return bundle
}

// excludeTestFilter implements a filter.LoaderFilter that excludes test files.
// ie: files with a `_test.rego` suffix
func excludeTestFilter() filter.LoaderFilter {
	return func(_ string, info fs.FileInfo, _ int) bool {
		return strings.HasSuffix(info.Name(), "_test.rego")
	}
}
