package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"github.com/google/uuid"
)

func ListComplianceReportJobs(baseUrl string, sourceID uuid.UUID, filter *TimeRangeFilter) ([]api2.ComplianceReport, error) {
	url := baseUrl + "/api/v1/sources/" + sourceID.String() + "/jobs/compliance"
	if filter != nil {
		url = fmt.Sprintf("%s?from=%d&to=%d", url, filter.From, filter.To)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response []api2.ComplianceReport
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
