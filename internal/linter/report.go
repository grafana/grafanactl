package linter

type Report struct {
	Violations []Violation `json:"violations"`
	Summary    Summary     `json:"summary"`
}

// ViolationsFileCount returns the number of violations per file (for files containing violations).
func (report Report) ViolationsFileCount() map[string]int {
	violationsMap := map[string]int{}
	for _, violation := range report.Violations {
		violationsMap[violation.Location.File]++
	}

	return violationsMap
}

// Violation describes any violation found by the linter.
type Violation struct {
	Rule             string            `json:"rule"`
	Description      string            `json:"description"`
	ResourceType     string            `json:"resource_type"`
	Severity         string            `json:"severity"`
	Location         Location          `json:"location,omitempty"`
	Details          string            `json:"details,omitempty"`
	RelatedResources []RelatedResource `json:"related_resources"`
}

func (violation Violation) DocumentationURL() string {
	for _, resource := range violation.RelatedResources {
		if resource.Description == "documentation" {
			return resource.Reference
		}
	}

	return ""
}

type Location struct {
	File string `json:"file"`
}

func (l Location) String() string {
	return l.File
}

type Summary struct {
	FilesScanned  int `json:"files_scanned"`
	FilesFailed   int `json:"files_failed"`
	RulesSkipped  int `json:"rules_skipped"`
	NumViolations int `json:"num_violations"`
}

type RelatedResource struct {
	Description string `json:"description"`
	Reference   string `json:"ref"`
}
