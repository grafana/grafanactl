package linter

type Rule struct {
	Category         string            `json:"category"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Builtin          bool              `json:"builtin"`
	Severity         string            `json:"severity"`
	RelatedResources []RelatedResource `json:"related_resources,omitempty"`
}
