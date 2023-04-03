package api

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
	"time"
)

type Benchmark struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	LogoURI     string            `json:"logoURI"`
	Category    string            `json:"category"`
	DocumentURI string            `json:"documentURI"`
	Enabled     bool              `json:"enabled"`
	Managed     bool              `json:"managed"`
	AutoAssign  bool              `json:"autoAssign"`
	Baseline    bool              `json:"baseline"`
	Tags        map[string]string `json:"tags"`
	Connectors  []source.Type     `json:"connectors"`
	Children    []string          `json:"children"`
	Policies    []string          `json:"policies"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func (b *Benchmark) PopulateConnectors(db db.Database) error {
	if len(b.Connectors) > 0 {
		return nil
	}

	for _, childId := range b.Children {
		child, err := db.GetBenchmark(childId)
		if err != nil {
			return err
		}
		if child == nil {
			return fmt.Errorf("child %s not found", childId)
		}

		ca := child.ToApi()
		err = ca.PopulateConnectors(db)
		if err != nil {
			return err
		}

		b.Connectors = append(b.Connectors, ca.Connectors...)
	}

	for _, policyId := range b.Policies {
		policy, err := db.GetPolicy(policyId)
		if err != nil {
			return err
		}
		if policy == nil {
			return fmt.Errorf("policy %s not found", policyId)
		}

		query, err := db.GetQuery(*policy.QueryID)
		if err != nil {
			return err
		}
		if query == nil {
			return fmt.Errorf("query %s not found", *policy.QueryID)
		}

		ty, err := source.ParseType(query.Connector)
		if err != nil {
			return err
		}

		b.Connectors = append(b.Connectors, ty)
	}

	return nil
}

type Policy struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Description        string            `json:"description"`
	Tags               map[string]string `json:"tags"`
	DocumentURI        string            `json:"documentURI"`
	QueryID            *string           `json:"queryID"`
	Severity           types.Severity    `json:"severity"`
	ManualVerification bool              `json:"manualVerification"`
	Managed            bool              `json:"managed"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type Query struct {
	ID             string    `json:"id"`
	QueryToExecute string    `json:"queryToExecute"`
	Connector      string    `json:"connector"`
	ListOfTables   string    `json:"listOfTables"`
	Engine         string    `json:"engine"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
