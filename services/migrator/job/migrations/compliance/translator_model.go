package compliance

type Benchmark struct {
	ID          string
	Title       string
	DisplayCode string
	Connector   string
	Description string
	Children    []string
	Tags        map[string][]string
	Managed     bool
	Enabled     bool
	AutoAssign  bool
	Baseline    bool
	Controls    []string
}

type Control struct {
	ID                 string
	Title              string
	Description        string
	Query              *Query
	ManualVerification bool
	Severity           string
	Tags               map[string][]string
	Managed            bool
}

type Query struct {
	ID             string
	Engine         string
	QueryToExecute string
	Connector      string
	PrimaryTable   *string
	ListOfTables   []string
}
