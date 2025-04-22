package format

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/goccy/go-yaml"
)

type Format string

const (
	JSON Format = "json"
	YAML Format = "yaml"
)

// Codecs return a list of default codecs.
func Codecs() map[Format]Codec {
	return map[Format]Codec{
		JSON: NewJSONCodec(),
		YAML: NewYAMLCodec(),
	}
}

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

	Format() Format
}

var _ Codec = (*YAMLCodec)(nil)

// YAMLCodec is a Codec that encodes and decodes resources to and from YAML.
type YAMLCodec struct {
	BytesAsBase64 bool
}

// NewYAMLCodec returns a new YAMLCodec.
func NewYAMLCodec() *YAMLCodec {
	return &YAMLCodec{}
}

func (c *YAMLCodec) Format() Format {
	return YAML
}

func (c *YAMLCodec) Encode(dst io.Writer, value any) error {
	opts := []yaml.EncodeOption{
		yaml.Indent(2),
		yaml.IndentSequence(true),
		yaml.UseJSONMarshaler(),
	}

	if c.BytesAsBase64 {
		opts = append(opts, yaml.CustomMarshaler(func(data []byte) ([]byte, error) {
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)

			return dst, nil
		}))
	}

	return yaml.NewEncoder(dst, opts...).Encode(value)
}

func (c *YAMLCodec) Decode(src io.Reader, value any) error {
	opts := []yaml.DecodeOption{
		yaml.Strict(),
		yaml.UseJSONUnmarshaler(),
	}

	if c.BytesAsBase64 {
		opts = append(opts, yaml.CustomUnmarshaler(func(dest *[]byte, raw []byte) error {
			dst := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
			_, err := base64.StdEncoding.Decode(dst, raw)
			if err != nil {
				return err
			}

			*dest = dst

			return nil
		}))
	}

	return yaml.NewDecoder(src, opts...).Decode(value)
}

var _ Codec = (*JSONCodec)(nil)

// JSONCodec is a Codec that encodes and decodes resources to and from JSON.
type JSONCodec struct{}

// NewJSONCodec returns a new JSONCodec.
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

func (c *JSONCodec) Format() Format {
	return JSON
}

func (c *JSONCodec) Encode(dst io.Writer, value any) error {
	encoder := json.NewEncoder(dst)
	encoder.SetIndent("", "  ")

	return encoder.Encode(value)
}

func (c *JSONCodec) Decode(src io.Reader, value any) error {
	return json.NewDecoder(src).Decode(value)
}
