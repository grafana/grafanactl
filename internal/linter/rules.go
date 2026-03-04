package linter

type Rule struct {
	Resource         string           `json:"resource"`
	Category         string           `json:"category"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Builtin          bool             `json:"builtin"`
	Severity         string           `json:"severity"`
	RelatedResources RelatedResources `json:"related_resources,omitempty"`
}

func (rule Rule) DocumentationURL() string {
	return rule.RelatedResources.DocumentationURL()
}
