package api

type Benchmark struct {
	ID          string
	Title       string
	Description string
	Provider    string
	Enabled     bool
	Tags        map[string]string
	Policies    []Policy
}

type Policy struct {
	ID                    string
	Title                 string
	Description           string
	Tags                  map[string]string
	Provider              string
	Category              string
	SubCategory           string
	Section               string
	Severity              string
	ManualVerification    string
	ManualRemedation      string
	CommandLineRemedation string
	QueryToRun            string
	KeibiManaged          bool
}
