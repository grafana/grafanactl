package format

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/goccy/go-yaml"
)

// Encoder encodes values to an io.Writer in a specific format.
type Encoder interface {
	Encode(dst io.Writer, value any) error
}

// Decoder decodes values from an io.Reader in a specific format.
type Decoder interface {
	Decode(src io.Reader, value any) error
}

// Codec takes care of encoding and decoding resources to and from a given format.
// It writes and reads data to provided io.Writer and io.Reader.
type Codec interface {
	Encoder
	Decoder
}

var _ Codec = (*YAMLCodec)(nil)

// YAMLCodec is a Codec that encodes and decodes resources to and from YAML.
type YAMLCodec struct{}

// NewYAMLCodec returns a new YAMLCodec.
func NewYAMLCodec() *YAMLCodec {
	return &YAMLCodec{}
}

func (c *YAMLCodec) Encode(dst io.Writer, value any) error {
	encoder := yaml.NewEncoder(
		dst,
		yaml.Indent(2),
		yaml.IndentSequence(true),
		yaml.UseJSONMarshaler(),
		yaml.CustomMarshaler(func(data []byte) ([]byte, error) {
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)

			return dst, nil
		}),
	)

	return encoder.Encode(value)
}

func (c *YAMLCodec) Decode(src io.Reader, value any) error {
	return yaml.NewDecoder(src).Decode(value)
}

var _ Codec = (*JSONCodec)(nil)

// JSONCodec is a Codec that encodes and decodes resources to and from JSON.
type JSONCodec struct{}

// NewJSONCodec returns a new JSONCodec.
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

func (c *JSONCodec) Encode(dst io.Writer, value any) error {
	encoder := json.NewEncoder(dst)
	encoder.SetIndent("", "  ")

	return encoder.Encode(value)
}

func (c *JSONCodec) Decode(src io.Reader, value any) error {
	return json.NewDecoder(src).Decode(value)
}
