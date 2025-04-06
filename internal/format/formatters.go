package format

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/goccy/go-yaml"
)

type Formatter func(output io.Writer, resource any) error

func YAML(output io.Writer, input any) error {
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

func JSON(output io.Writer, input any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	return encoder.Encode(input)
}
