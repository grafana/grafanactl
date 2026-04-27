package secrets

import (
	"reflect"
	"strings"
	"sync"
)

//nolint:gochecknoglobals
var denylistCache sync.Map

// RedactYAMLSecrets returns a copy of contents where every YAML scalar value
// associated with a key derived from a datapolicy:"secret"-tagged field in root
// (or any struct reachable from root via pointer, struct, slice, array, or map
// fields) is replaced by a length-preserving sentinel.
//
// The replacement preserves the exact byte length of the input, so that
// yaml-library-reported line and column offsets remain accurate after
// redaction. Sensitive scalar values are replaced with "**REDACTED**" padded
// with ASCII spaces (when the value is longer) or truncated to the first N
// bytes of "**REDACTED**" (when shorter). Because the sentinel is pure ASCII,
// no UTF-8 rune is ever split at a redaction boundary.
//
// Coverage:
//   - Inline scalars: key: value
//   - Block scalars:  key: |  (followed by indented lines)
//   - Folded scalars: key: >  (followed by indented lines)
//
// Non-sensitive keys, comments, newlines, and indentation are never modified.
// Key matching is exact: "tokenizer" does not match a "token" denylist entry.
//
// Note: a user-defined map key that coincidentally shares the name of a
// datapolicy-tagged field will also be redacted. This is an accepted,
// documented limitation with no known collision in the current config schema.
func RedactYAMLSecrets(contents []byte, root reflect.Type) []byte {
	if len(contents) == 0 {
		return contents
	}

	denylist := buildDenylist(root)

	out := make([]byte, len(contents))
	copy(out, contents)

	if len(denylist) > 0 {
		redactInPlace(out, denylist)
	}

	return out
}

// buildDenylist returns the set of sensitive YAML key names reachable from
// root, computing it once and caching per root type.
func buildDenylist(root reflect.Type) map[string]struct{} {
	if cached, ok := denylistCache.Load(root); ok {
		if m, ok := cached.(map[string]struct{}); ok {
			return m
		}
	}

	m := make(map[string]struct{})
	visited := make(map[reflect.Type]bool)
	walkType(root, m, visited)

	canonical, _ := denylistCache.LoadOrStore(root, m)
	if result, ok := canonical.(map[string]struct{}); ok {
		return result
	}

	return m
}

// walkType recursively collects the YAML key names of datapolicy:"secret"-tagged
// fields reachable from t, writing them into keys. The visited set prevents
// infinite recursion on self-referential types.
func walkType(t reflect.Type, keys map[string]struct{}, visited map[reflect.Type]bool) {
	// Unwrap pointer indirections.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		if visited[t] {
			return
		}
		visited[t] = true

		for _, f := range reflect.VisibleFields(t) {
			// Skip promoted fields (embedded structs) — we recurse into their
			// concrete types explicitly below to avoid double-counting.
			if len(f.Index) != 1 {
				continue
			}
			policy := strings.SplitN(f.Tag.Get(dataPolicyTag), ",", 2)[0]
			if policy == "secret" {
				name := yamlKeyName(f.Name, f.Tag.Get("yaml"))
				if name != "" && name != "-" {
					keys[name] = struct{}{}
				}
			} else {
				walkType(f.Type, keys, visited)
			}
		}

	case reflect.Slice, reflect.Array:
		walkType(t.Elem(), keys, visited)

	case reflect.Map:
		walkType(t.Elem(), keys, visited)
	}
}

// yamlKeyName returns the YAML mapping key for a struct field. It uses the
// first token of the "yaml" struct tag when present, or the lowercased field
// name otherwise.
func yamlKeyName(fieldName, yamlTag string) string {
	if yamlTag == "" {
		return strings.ToLower(fieldName)
	}
	name := strings.SplitN(yamlTag, ",", 2)[0]
	if name == "" {
		return strings.ToLower(fieldName)
	}
	return name
}

// redactInPlace scans out line by line and overwrites sensitive scalar values
// with a length-preserving sentinel. It maintains a two-state machine:
//
//	normal     – scanning for key lines
//	inBlock    – consuming block/folded scalar continuation lines
func redactInPlace(out []byte, denylist map[string]struct{}) {
	pos := 0
	inBlock := false
	blockKeyIndent := 0 // column of the key that opened the block scalar

	for pos < len(out) {
		lineStart := pos

		// Locate the end of the current line.
		lineEnd := lineStart
		for lineEnd < len(out) && out[lineEnd] != '\n' {
			lineEnd++
		}

		// visibleEnd: position just before a trailing '\r' (CRLF handling).
		visibleEnd := lineEnd
		if visibleEnd > lineStart && out[visibleEnd-1] == '\r' {
			visibleEnd--
		}

		if inBlock {
			// Count leading whitespace bytes on this line.
			li := lineStart
			for li < visibleEnd && (out[li] == ' ' || out[li] == '\t') {
				li++
			}
			lineIndent := li - lineStart
			isBlank := li == visibleEnd // all-whitespace or empty line

			switch {
			case isBlank:
				// Blank lines are part of the block scalar; leave them unchanged.
			case lineIndent > blockKeyIndent:
				// Continuation line: redact from first non-whitespace to end-of-line.
				redactFill(out, li, visibleEnd)
			default:
				// Indent dropped to key level or less: block scalar is over.
				inBlock = false
				tryRedactKey(out, lineStart, visibleEnd, denylist, &inBlock, &blockKeyIndent)
			}
		} else {
			tryRedactKey(out, lineStart, visibleEnd, denylist, &inBlock, &blockKeyIndent)
		}

		// Advance past the '\n', or to end of buffer.
		if lineEnd < len(out) {
			pos = lineEnd + 1
		} else {
			pos = lineEnd
		}
	}
}

// tryRedactKey inspects a single line for a denylist key and either redacts
// its inline value or enters block-scalar mode.
func tryRedactKey(out []byte, lineStart, visibleEnd int, denylist map[string]struct{}, inBlock *bool, blockKeyIndent *int) {
	p := lineStart

	// Skip leading whitespace.
	for p < visibleEnd && (out[p] == ' ' || out[p] == '\t') {
		p++
	}

	// Skip comment-only and blank lines.
	if p >= visibleEnd || out[p] == '#' {
		return
	}

	// Consume an optional list-item prefix "- " and track the column of the
	// actual key name (needed for block-scalar indentation comparison).
	if p+1 < visibleEnd && out[p] == '-' && out[p+1] == ' ' {
		p += 2
		// Skip any extra whitespace between '-' and the key name.
		for p < visibleEnd && (out[p] == ' ' || out[p] == '\t') {
			p++
		}
	}

	// keyIndent: byte column of the first character of the key name.
	keyStartPos := p
	keyIndent := p - lineStart

	// Extract the key name: scan forward while characters are valid YAML bare-key
	// bytes ([a-zA-Z0-9_-]). Stopping at the first non-key byte lets us detect
	// both the normal "key: value" form AND invalid-separator typos like "key; value".
	for p < visibleEnd && isYAMLKeyByte(out[p]) {
		p++
	}

	if p >= visibleEnd {
		return // line is a bare word with no separator and no value
	}

	// Key not in denylist: nothing to redact on this line.
	if _, ok := denylist[string(out[keyStartPos:p])]; !ok {
		return
	}

	sep := out[p]

	if sep != ':' {
		// Non-standard separator (e.g. ';', '=') — the line is invalid YAML, but
		// the key matched a sensitive name so redact everything from the separator
		// to end-of-line (inclusive), preserving length.
		redactFill(out, p, visibleEnd)
		return
	}

	p++ // advance past ':'

	// After ':' the next byte must be whitespace, '#', or end-of-line.
	// This guard prevents "tokenizer:" from matching "token".
	if p < visibleEnd && out[p] != ' ' && out[p] != '\t' && out[p] != '#' {
		return
	}

	// Skip optional whitespace between ':' and value.
	for p < visibleEnd && (out[p] == ' ' || out[p] == '\t') {
		p++
	}

	// Empty value or trailing comment: nothing to redact inline.
	if p >= visibleEnd || out[p] == '#' {
		return
	}

	// Block or folded scalar indicator ('|' / '>') — enter block mode.
	if isBlockMarker(out[p:visibleEnd]) {
		*inBlock = true
		*blockKeyIndent = keyIndent
		return
	}

	// Inline scalar: redact everything from the start of the value to end-of-line.
	// This includes any trailing comment, which is the safest policy.
	redactFill(out, p, visibleEnd)
}

// isYAMLKeyByte reports whether b is a valid bare YAML key character: ASCII
// alphanumeric, underscore, or hyphen.
func isYAMLKeyByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_' || b == '-'
}

// isBlockMarker reports whether b starts with a YAML block scalar indicator
// ('|' or '>'), optionally followed by chomping indicators ('+' / '-') and/or
// an explicit indentation digit, then optional whitespace and/or a comment.
func isBlockMarker(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if b[0] != '|' && b[0] != '>' {
		return false
	}
	i := 1
	for i < len(b) && (b[i] == '-' || b[i] == '+' || (b[i] >= '0' && b[i] <= '9')) {
		i++
	}
	for i < len(b) {
		switch b[i] {
		case ' ', '\t':
			i++
		case '#':
			return true
		default:
			return false
		}
	}
	return true
}

// redactFill overwrites out[start:end] with a length-preserving sentinel.
//
//   - len >= 12: "**REDACTED**" followed by (len-12) ASCII spaces.
//   - len <  12: first len bytes of "**REDACTED**" (pure ASCII, no rune split).
//   - len == 0:  no-op.
func redactFill(out []byte, start, end int) {
	n := end - start
	if n <= 0 {
		return
	}
	slen := len(redacted) // 12
	if n >= slen {
		copy(out[start:], redacted)
		for i := start + slen; i < end; i++ {
			out[i] = ' '
		}
	} else {
		copy(out[start:start+n], redacted[:n])
	}
}
