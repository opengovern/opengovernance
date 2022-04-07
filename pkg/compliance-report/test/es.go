package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/google/uuid"
	report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
)

func PopulateElastic(address string) error {
	cfg := elasticsearchv7.Config{
		Addresses: []string{address},
		Username:  "",
		Password:  "",
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	es, err := elasticsearchv7.NewClient(cfg)
	if err != nil {
		return err
	}

	reports := GenerateReports()
	for _, rep := range reports {
		err := IndexReport(es, rep)
		if err != nil {
			return err
		}
	}

	return nil
}

func GenerateReports() []report.Report {
	uuid1, _ := uuid.Parse("c29c0dae-823f-4726-ade0-5fa94a941e88")
	uuid2, _ := uuid.Parse("efa82471-fd00-4d17-9009-e9dad0fe986b")

	jobIDs := []uint{1, 2}
	sourceIDs := []uuid.UUID{uuid1, uuid2}
	benchmarks := []string{"cis.001", "cis.002", "cis.003"}
	results := []string{"cis.001.control1", "cis.001.control2", "cis.001.control3"}

	var reports []report.Report
	for _, jobID := range jobIDs {
		for _, sourceID := range sourceIDs {
			for _, benchmark := range benchmarks {
				r := report.Report{
					Result: nil,
					Group: &report.ReportGroupObj{
						ID:             benchmark,
						Title:          benchmark,
						Description:    benchmark,
						Tags:           map[string]string{},
						Summary:        report.Summary{},
						ChildGroupIds:  nil,
						ControlIds:     nil,
						ParentGroupIds: nil,
					},
					Type:        report.ReportTypeBenchmark,
					ReportJobId: jobID,
					SourceID:    sourceID,
				}
				reports = append(reports, r)
			}

			for _, result := range results {
				r := report.Report{
					Result: &report.ReportResultObj{
						Result: report.Result{
							Reason:     "",
							Resource:   "",
							Status:     "",
							Dimensions: nil,
						},
						ControlId:      result,
						ParentGroupIds: nil,
					},
					Group:       nil,
					Type:        report.ReportTypeResult,
					ReportJobId: jobID,
					SourceID:    sourceID,
				}
				reports = append(reports, r)
			}
		}
	}

	return reports
}

func IndexReport(es *elasticsearchv7.Client, rep report.Report) error {
	js, err := json.Marshal(rep)
	if err != nil {
		return err
	}

	// Set up the request object.
	req := esapi.IndexRequest{
		Index:      report.ComplianceReportIndex,
		DocumentID: uuid.New().String(),
		Body:       bytes.NewReader(js),
		Refresh:    "true",
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("[%s] Error indexing document ID=%s", res.Status(), req.DocumentID)
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}
	}
	return nil
}
