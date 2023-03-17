package compliance

type Benchmark struct {
	ID          string
	Title       string
	DocumentURI string
	Description string
	Children    []string
	Tags        []uint
	Managed     bool
	LogoURI     string
	Category    string
	Enabled     bool
	AutoAssign  bool
	Baseline    bool
	Policies    []string
}

type BenchmarkTag struct {
	ID    uint
	Key   string
	Value string
}

type Policy struct {
	ID                 string
	Title              string
	Description        string
	QueryID            *string
	DocumentURI        string
	ManualVerification bool
	Severity           string
	Tags               []uint
	Managed            bool
}

type PolicyTag struct {
	ID    uint
	Key   string
	Value string
}

type Query struct {
	ID             string
	Engine         string
	QueryToExecute string
	Connector      string
	ListOfTables   string
	ResourceName   string
}
