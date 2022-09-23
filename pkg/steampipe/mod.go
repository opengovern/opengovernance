package steampipe

import "strings"

type Mod struct {
	ID    string
	Title string
}

func (c Mod) String() string {
	return `mod "` + c.ID + `" {
    title = "` + c.Title + `"
}
`
}

type Benchmark struct {
	ID          string
	Title       string
	Description string
	Children    []Control
}

func (c Benchmark) String() string {
	var childrenString string
	var children []string
	for _, child := range c.Children {
		children = append(children, "control."+child.ID)
		childrenString += "\n\n" + child.String()
	}
	return `benchmark "` + c.ID + `" {
  title = "` + c.Title + `"
  description = "` + c.Description + `"
  children = [
    ` + strings.Join(children, ",") + `
  ]
}` + childrenString
}

type Control struct {
	ID          string
	Title       string
	Description string
	Severity    Severity
	Tags        map[string]string
	SQL         string
}

type Severity = string

const (
	SeverityNone     = "none"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

func (c Control) String() string {
	var tagsLines []string
	for k, v := range c.Tags {
		tagsLines = append(tagsLines, k+" = \""+v+"\"")
	}
	return `control "` + c.ID + `" {
  title         = "` + c.Title + `"
  description   = "` + c.Description + `"
  severity      = "` + c.Severity + `"
  tags = {
    ` + strings.Join(tagsLines, "\n") + `
  }
    sql = <<EOT
        ` + c.SQL + `
    EOT
}`
}
