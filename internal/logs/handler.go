package logs

import (
	"bytes"
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
)

const (
	ansiEsc = '\u001b'
	errKey  = "err"
)

//nolint:gochecknoglobals
var (
	debugColor = color.New(color.FgMagenta).Add(color.Faint).SprintfFunc()
	infoColor  = color.New(color.FgGreen).SprintfFunc()
	warnColor  = color.New(color.FgYellow).SprintfFunc()
	errColor   = color.New(color.FgRed).SprintfFunc()

	attrKeyColor = color.New(color.Faint).SprintfFunc()
	errMsgColor  = color.New(color.FgRed).Add(color.Faint).SprintfFunc()

	defaultLevel = slog.LevelInfo
)

// Options for a slog.Handler that writes colored logs. A zero Options consists
// entirely of default values.
type Options struct {
	// Minimum level to log (Default: slog.LevelInfo)
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See https://pkg.go.dev/log/slog#HandlerOptions for details.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr
}

// NewHandler creates a [slog.Handler] that writes tinted logs to Writer w,
// using the default options. If opts is nil, the default options are used.
func NewHandler(w io.Writer, opts *Options) slog.Handler {
	h := &handler{
		w:     w,
		level: defaultLevel,
	}
	if opts == nil {
		return h
	}

	if opts.Level != nil {
		h.level = opts.Level
	}
	h.replaceAttr = opts.ReplaceAttr

	return h
}

// handler implements a [slog.Handler].
type handler struct {
	attrsPrefix string
	groupPrefix string
	groups      []string

	mu sync.Mutex
	w  io.Writer

	level       slog.Leveler
	replaceAttr func([]string, slog.Attr) slog.Attr
}

func (h *handler) clone() *handler {
	return &handler{
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		w:           h.w,
		level:       h.level,
		replaceAttr: h.replaceAttr,
	}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	buf := &bytes.Buffer{}

	rep := h.replaceAttr

	// Level
	if rep == nil {
		h.appendLevel(buf, r.Level)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.Any(slog.LevelKey, r.Level)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// Message
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.String(slog.MessageKey, r.Message)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// Handler attributes
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}

	// Record attributes
	r.Attrs(func(attr slog.Attr) bool {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
		return true
	})

	if buf.Len() == 0 {
		return nil
	}

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	clone := h.clone()
	buf := &bytes.Buffer{}

	for _, attr := range attrs {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
	}
	clone.attrsPrefix = h.attrsPrefix + buf.String()

	return clone
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.groupPrefix += name + "."
	clone.groups = append(clone.groups, name)

	return clone
}

func (h *handler) appendLevel(buf *bytes.Buffer, level slog.Level) {
	switch {
	case level < slog.LevelInfo:
		buf.WriteString(debugColor(level.String()))
	case level < slog.LevelWarn:
		buf.WriteString(infoColor(level.String()))
	case level < slog.LevelError:
		buf.WriteString(warnColor(level.String()))
	default:
		buf.WriteString(errColor(level.String()))
	}
}

func (h *handler) appendAttr(buf *bytes.Buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	attr.Value = attr.Value.Resolve()
	if rep := h.replaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
	} else if err, ok := attr.Value.Any().(error); ok {
		buf.WriteString(errMsgColor("%s=%s", prepareString(groupsPrefix+attr.Key, true), prepareString(err.Error(), true)))
		buf.WriteByte(' ')
	} else {
		h.appendKey(buf, attr.Key, groupsPrefix)
		h.appendValue(buf, attr.Value, true)
		buf.WriteByte(' ')
	}
}

func (h *handler) appendKey(buf *bytes.Buffer, key, groups string) {
	buf.WriteString(attrKeyColor(prepareString(groups+key, true)))
	buf.WriteString(attrKeyColor("="))
}

func (h *handler) appendValue(buf *bytes.Buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		buf.WriteString(strconv.FormatInt(v.Int64(), 10))
	case slog.KindUint64:
		buf.WriteString(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindFloat64:
		buf.WriteString(strconv.FormatFloat(v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.WriteString(strconv.FormatBool(v.Bool()))
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		appendString(buf, v.Time().String(), quote)
	case slog.KindLogValuer:
		h.appendValue(buf, v.LogValuer().LogValue(), quote)
	case slog.KindAny:
		defer func() {
			// Copied from log/slog/handler.go.
			if r := recover(); r != nil {
				// If it panics with a nil pointer, the most likely cases are
				// an encoding.TextMarshaler or error fails to guard against nil,
				// in which case "<nil>" seems to be the feasible choice.
				//
				// Adapted from the code in fmt/print.go.
				if v := reflect.ValueOf(v.Any()); v.Kind() == reflect.Pointer && v.IsNil() {
					buf.WriteString("<nil>")
					return
				}

				// Otherwise just print the original panic message.
				appendString(buf, fmt.Sprintf("!PANIC: %v", r), true)
			}
		}()

		switch cv := v.Any().(type) {
		case slog.Level:
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendString(buf, string(data), quote)
		default:
			appendString(buf, fmt.Sprintf("%+v", cv), quote)
		}
	}
}

func prepareString(s string, quote bool) string {
	if quote {
		// trim ANSI escape sequences
		var inEscape bool
		s = cut(s, func(r rune) bool {
			if r == ansiEsc {
				inEscape = true
			} else if inEscape && unicode.IsLetter(r) {
				inEscape = false
				return true
			}

			return inEscape
		})
	}

	switch {
	case quote && needsQuoting(s):
		s = strconv.Quote(s)
		s = strings.ReplaceAll(s, `\x1b`, string(ansiEsc))
		return s
	default:
		return s
	}
}

func appendString(buf *bytes.Buffer, s string, quote bool) {
	buf.WriteString(prepareString(s, quote))
}

func cut(s string, f func(r rune) bool) string {
	var res []rune
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			break
		}
		if !f(r) {
			res = append(res, r)
		}
		i += size
	}
	return string(res)
}

// Copied from log/slog/text_handler.go.
func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			// Quote anything except a backslash that would need quoting in a
			// JSON string, as well as space and '='
			if b != '\\' && (b == ' ' || b == '=' || !safeSet[b]) {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

// Copied from log/slog/json_handler.go.
//
// safeSet is extended by the ANSI escape code "\u001b".
//
//nolint:gochecknoglobals
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
	'\u001b': true,
}

// Err is just a shortcut to `slog.Any("err", err)`.
func Err(err error) slog.Attr {
	return slog.Any(errKey, err)
}
