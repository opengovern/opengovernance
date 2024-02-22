package shared

type Query struct {
	ID             string           `json:"ID" yaml:"ID"`
	Engine         string           `json:"Engine" yaml:"Engine"`
	QueryToExecute string           `json:"QueryToExecute" yaml:"QueryToExecute"`
	Connector      string           `json:"Connector" yaml:"Connector"`
	PrimaryTable   *string          `json:"PrimaryTable" yaml:"PrimaryTable"`
	ListOfTables   []string         `json:"ListOfTables" yaml:"ListOfTables"`
	Parameters     []QueryParameter `json:"Parameters" yaml:"Parameters"`
}

type QueryParameter struct {
	Key      string `json:"Key" yaml:"Key"`
	Required bool   `json:"Required" yaml:"Required"`
}
