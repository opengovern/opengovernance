package shared

type Query struct {
	ID             string           `json:"ID,omitempty" yaml:"ID,omitempty"`
	Engine         string           `json:"Engine" yaml:"Engine"`
	QueryToExecute string           `json:"QueryToExecute" yaml:"QueryToExecute"`
	PrimaryTable   *string          `json:"PrimaryTable" yaml:"PrimaryTable"`
	ListOfTables   []string         `json:"ListOfTables" yaml:"ListOfTables"`
	Parameters     []QueryParameter `json:"Parameters" yaml:"Parameters"`
	Global         bool             `json:"Global,omitempty" yaml:"Global,omitempty"`
}

type QueryParameter struct {
	Key          string  `json:"Key" yaml:"Key"`
	Required     bool    `json:"Required" yaml:"Required"`
	DefaultValue *string `json:"DefaultValue" yaml:"DefaultValue"`
}
