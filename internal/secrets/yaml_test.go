package secrets_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/grafana/grafanactl/internal/secrets"
	"github.com/stretchr/testify/require"
)

// The local types below mirror internal/config's secret-carrying structs.
// A local definition is used to avoid an import cycle: internal/config imports
// internal/secrets via secrets.Redact[V], so internal/secrets must not import
// internal/config.

type yamlTestTLS struct {
	KeyData string `datapolicy:"secret" yaml:"key-data"`
	Other   string `yaml:"other"`
}

type yamlTestGrafana struct {
	Server   string       `yaml:"server"`
	User     string       `yaml:"user"`
	Password string       `datapolicy:"secret" yaml:"password"`
	Token    string       `datapolicy:"secret" yaml:"token"`
	TLS      *yamlTestTLS `yaml:"tls"`
}

type yamlTestContext struct {
	Grafana *yamlTestGrafana `yaml:"grafana"`
}

//nolint:gochecknoglobals
var testRootType = reflect.TypeFor[yamlTestContext]()

// utf8MultiByteValue is a string whose byte length equals the sentinel length
// (12 bytes) and whose runes require multiple bytes each to encode. It is
// composed of 4 euro-sign runes (U+20AC, 3 bytes each) = 12 bytes total.
// Using a non-CJK character avoids the gosmopolitan linter warning.
const utf8MultiByteValue = "\xe2\x82\xac\xe2\x82\xac\xe2\x82\xac\xe2\x82\xac" // "€€€€" 12 bytes

// utf8ShortValue is a 3-byte UTF-8 rune (U+20AC euro sign).
const utf8ShortValue = "\xe2\x82\xac" // "€" 3 bytes

func TestRedactYAMLSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		root            reflect.Type // defaults to testRootType
		notContains     []string     // substrings that MUST be absent in output
		contains        []string     // substrings that MUST be present in output
		expectUnchanged bool         // output must byte-equal input
		wantOutput      string       // exact expected output (when set)
	}{
		// ── inline scalars ───────────────────────────────────────────────
		{
			name:        "inline token",
			input:       "token: abc123\n",
			notContains: []string{"abc123"},
			contains:    []string{"token"},
		},
		{
			// AC-1 / AC-13 unit basis: semicolon instead of colon (colon-typo).
			// goccy/go-yaml reports a parse error whose source window includes
			// this line; we must redact the value even though there is no ':'.
			name:        "invalid separator (semicolon) redacts rest of line",
			input:       "token; glc_fixture_secret_value\n",
			notContains: []string{"glc_fixture_secret_value"},
			contains:    []string{"token"},
		},
		{
			name:        "inline password",
			input:       "password: hunter2\n",
			notContains: []string{"hunter2"},
			contains:    []string{"password"},
		},
		{
			name:        "nested tls key-data inline",
			input:       "tls:\n  key-data: supersecretvalue\n",
			notContains: []string{"supersecretvalue"},
			contains:    []string{"key-data"},
		},
		// ── block / folded scalars ────────────────────────────────────────
		{
			name:  "block scalar token",
			input: "token: |\n  secretline1\n  secretline2\nother: value\n",
			// Both continuation lines must be redacted, the terminating key must survive.
			notContains: []string{"secretline1", "secretline2"},
			contains:    []string{"token", "other: value"},
		},
		{
			name:        "folded scalar token",
			input:       "token: >\n  secret folded\n  more secret\nother: value\n",
			notContains: []string{"secret folded", "more secret"},
			contains:    []string{"token", "other: value"},
		},
		{
			name: "block scalar PEM key-data",
			input: "tls:\n" +
				"  key-data: |\n" +
				"    -----BEGIN PRIVATE KEY-----\n" +
				"    MIIEvQIBADANBgk=\n" +
				"    -----END PRIVATE KEY-----\n" +
				"other: value\n",
			notContains: []string{"BEGIN PRIVATE KEY", "MIIEvQIBADANBgk=", "END PRIVATE KEY"},
			contains:    []string{"key-data", "other: value"},
		},
		// ── no-over-redaction ─────────────────────────────────────────────
		{
			name:            "similar-prefix key is not redacted",
			input:           "tokenizer: somevalue\n",
			expectUnchanged: true,
		},
		{
			name:            "comment line with secret-looking content is unchanged",
			input:           "# token: fake\n",
			expectUnchanged: true,
		},
		{
			name:            "no sensitive keys leaves input unchanged",
			input:           "server: https://grafana.example.com\nuser: alice\n",
			expectUnchanged: true,
		},
		// ── edge cases ────────────────────────────────────────────────────
		{
			name:  "empty input returns empty output",
			input: "",
		},
		{
			name:        "windows CRLF line endings are preserved",
			input:       "token: secret\r\n",
			notContains: []string{"secret"},
			contains:    []string{"\r\n"},
		},
		{
			name:        "duplicate keys are both redacted",
			input:       "token: firstsecret\ntoken: secondsecret\n",
			notContains: []string{"firstsecret", "secondsecret"},
			contains:    []string{"token"},
		},
		{
			name:        "value indented under list item is redacted",
			input:       "- token: xyz\n",
			notContains: []string{"xyz"},
			contains:    []string{"token"},
		},
		// ── UTF-8 / length preservation ───────────────────────────────────
		{
			// 4 x euro-sign (U+20AC, 3 bytes each) = 12 bytes = exactly the sentinel length.
			name:        "UTF-8 value of exactly sentinel length",
			input:       "token: " + utf8MultiByteValue + "\n",
			notContains: []string{utf8MultiByteValue},
			wantOutput:  "token: **REDACTED**\n",
		},
		{
			// Short secret: value is 1 ASCII byte; sentinel truncates to "*".
			name:       "short secret under 12 bytes produces truncated sentinel",
			input:      "token: a\n",
			wantOutput: "token: *\n",
		},
		{
			// 3-byte UTF-8 rune: sentinel truncates to first 3 bytes "**R".
			name:       "UTF-8 short secret produces rune-safe truncated sentinel",
			input:      "token: " + utf8ShortValue + "\n",
			wantOutput: "token: **R\n",
		},
		// ── AC-8: denylist derives entirely from struct reflection ─────────
		// The synthetic struct below introduces "super-secret" as a new key
		// that yaml.go has never been told about explicitly.  Redaction must
		// work purely because of the datapolicy:"secret" tag.
		{
			name:        "AC-8 synthetic struct with novel secret field",
			input:       "super-secret: hiddenvalue\npublic-field: visible\n",
			root:        reflect.TypeFor[ac8Root](),
			notContains: []string{"hiddenvalue"},
			contains:    []string{"public-field: visible"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			in := []byte(tt.input)
			root := testRootType
			if tt.root != nil {
				root = tt.root
			}

			out := secrets.RedactYAMLSecrets(in, root)

			// AC-9: output must have the same byte length as input.
			req.Len(out, len(in),
				"output byte length must equal input byte length")

			// AC-9: output must be valid UTF-8 with no split rune.
			req.True(utf8.Valid(out), "output must be valid UTF-8")

			// Exact output check.
			if tt.wantOutput != "" {
				req.Equal(tt.wantOutput, string(out))
			}

			// Byte-equality for no-op cases.
			if tt.expectUnchanged {
				req.Equal(string(in), string(out),
					"input should be returned unchanged")
			}

			// Absent substrings.
			for _, sub := range tt.notContains {
				req.NotContains(string(out), sub,
					"sensitive substring %q must not appear in output", sub)
			}

			// Required substrings.
			for _, sub := range tt.contains {
				req.Contains(string(out), sub,
					"expected substring %q not found in output", sub)
			}
		})
	}
}

// TestRedactYAMLSecrets_LengthInvariantAllCases re-runs every case and asserts
// len(out) == len(in) as a belt-and-suspenders check separated from the main
// table so failures show a dedicated message.
func TestRedactYAMLSecrets_LengthInvariantAllCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		root  reflect.Type
	}{
		{"empty", "", testRootType},
		{"inline token", "token: glc_secret\n", testRootType},
		{"inline password", "password: hunter2\n", testRootType},
		{"no sensitive keys", "server: https://x.com\nuser: bob\n", testRootType},
		{"crlf", "token: secret\r\n", testRootType},
		{"block scalar", "token: |\n  line1\n  line2\n", testRootType},
		{"utf8 short", "token: " + utf8ShortValue + "\n", testRootType},
		{"utf8 exact", "token: " + utf8MultiByteValue + "\n", testRootType},
		{"ac8 synthetic", "super-secret: x\n", reflect.TypeFor[ac8Root]()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := []byte(c.input)
			out := secrets.RedactYAMLSecrets(in, c.root)
			require.Len(t, out, len(in),
				"len(out) must equal len(in) for %q", c.name)
			require.True(t, utf8.Valid(out),
				"output must be valid UTF-8 for %q", c.name)
		})
	}
}

// ── AC-8 supporting type ──────────────────────────────────────────────────────

// ac8Root is a synthetic struct that introduces a secret field ("super-secret")
// that is unknown to yaml.go's code.  It verifies that denylist derivation is
// purely reflection-driven and requires no hardcoding.
type ac8Root struct {
	SuperSecret string `datapolicy:"secret" yaml:"super-secret"`
	PublicField string `yaml:"public-field"`
}
