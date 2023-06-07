package compliance

type Benchmark struct {
	ID          string
	Title       string
	DocumentURI string
	Description string
	Children    []string
	Tags        map[string][]string
	Managed     bool
	LogoURI     string
	Category    string
	Enabled     bool
	AutoAssign  bool
	Baseline    bool
	Policies    []string
}

type Policy struct {
	ID                 string
	Title              string
	Description        string
	QueryID            *string
	DocumentURI        string
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
	ListOfTables   []string
}
