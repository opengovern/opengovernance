package client

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/httprequest"
	inventory_api "gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func ListComplianceReportJobs(baseUrl string, sourceID uuid.UUID, filter *inventory_api.TimeRangeFilter) ([]api.ComplianceReport, error) {
	url := fmt.Sprintf("%s/api/v1/sources/%s/jobs/compliance", baseUrl, sourceID.String())
	if filter != nil {
		url = fmt.Sprintf("%s?from=%d&to=%d", url, filter.From, filter.To)
	}

	reports := []api.ComplianceReport{}
	if err := httprequest.DoRequest(http.MethodGet, url, nil, &reports); err != nil {
		return nil, err
	}
	return reports, nil
}

func GetLastComplianceReportID(baseUrl string) (uint, error) {
	url := fmt.Sprintf("%s/api/v1/compliance/report/last/completed", baseUrl)
	var res uint
	if err := httprequest.DoRequest(http.MethodGet, url, nil, &res); err != nil {
		return 0, err
	}
	return res, nil
}
