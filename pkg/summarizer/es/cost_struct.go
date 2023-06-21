package es

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	awsModel "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	azureModel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/helpers"
)

func getTimeFromTimestring(timestring string) time.Time {
	t, _ := time.Parse("2006-01-02", timestring)
	return t
}

func getTimeFromTimeInt(timeint int) time.Time {
	timestring := fmt.Sprintf("%d", timeint)
	t, _ := time.Parse("20060102", timestring)
	return t
}

type CostResourceType string

const (
	CostResourceTypeNull CostResourceType = ""

	CostResourceTypeAWSCostExplorerServiceCostMonthly CostResourceType = "aws::costexplorer::byservicemonthly"
	CostResourceTypeAWSCostExplorerAccountCostMonthly CostResourceType = "aws::costexplorer::byaccountmonthly"
	CostResourceTypeAWSCostExplorerServiceCostDaily   CostResourceType = "aws::costexplorer::byservicedaily"
	CostResourceTypeAWSCostExplorerAccountCostDaily   CostResourceType = "aws::costexplorer::byaccountdaily"
	CostResourceTypeAWSEBSVolume                      CostResourceType = "aws::ec2::volume"

	CostResourceTypeAzureCostManagementCostByResourceType CostResourceType = "microsoft.costmanagement/costbyresourcetype"
	CostResourceTypeAzureCostManagementCostBySubscription CostResourceType = "microsoft.costmanagement/costbysubscription"
)

func (c CostResourceType) String() string {
	return string(c)
}

var CostResourceTypeList = []CostResourceType{
	CostResourceTypeAWSCostExplorerServiceCostMonthly,
	CostResourceTypeAWSCostExplorerAccountCostMonthly,
	CostResourceTypeAWSCostExplorerServiceCostDaily,
	CostResourceTypeAWSCostExplorerAccountCostDaily,
	CostResourceTypeAWSEBSVolume,
	CostResourceTypeAzureCostManagementCostByResourceType,
	CostResourceTypeAzureCostManagementCostBySubscription,
}

func GetCostResourceTypeFromString(resourceType string) CostResourceType {
	lowerResourceType := strings.ToLower(resourceType)
	switch lowerResourceType {
	case "aws::costexplorer::byservicemonthly":
		return CostResourceTypeAWSCostExplorerServiceCostMonthly
	case "aws::costexplorer::byaccountmonthly":
		return CostResourceTypeAWSCostExplorerAccountCostMonthly
	case "aws::costexplorer::byservicedaily":
		return CostResourceTypeAWSCostExplorerServiceCostDaily
	case "aws::costexplorer::byaccountdaily":
		return CostResourceTypeAWSCostExplorerAccountCostDaily
	case "aws::ec2::volume":
		return CostResourceTypeAWSEBSVolume
	case "microsoft.costmanagement/costbyresourcetype":
		return CostResourceTypeAzureCostManagementCostByResourceType
	case "microsoft.costmanagement/costbysubscription":
		return CostResourceTypeAzureCostManagementCostBySubscription
	}
	return CostResourceTypeNull
}

func (c CostResourceType) GetProviderReportType() ProviderReportType {
	switch c {
	case CostResourceTypeAWSCostExplorerServiceCostMonthly, CostResourceTypeAWSCostExplorerAccountCostMonthly:
		return CostServiceSummaryMonthly
	case CostResourceTypeAWSCostExplorerServiceCostDaily, CostResourceTypeAWSCostExplorerAccountCostDaily, CostResourceTypeAWSEBSVolume:
		return CostServiceSummaryDaily
	case CostResourceTypeAzureCostManagementCostByResourceType, CostResourceTypeAzureCostManagementCostBySubscription:
		return CostServiceSummaryDaily
	}
	return ""
}

func (c CostResourceType) GetCostAndUnitFromResource(costDescriptionObj any) (float64, string) {
	costDescriptionJson, _ := json.Marshal(costDescriptionObj)
	var costDescription map[string]any
	_ = json.Unmarshal(costDescriptionJson, &costDescription)

	var err error
	switch c {
	case CostResourceTypeAWSCostExplorerServiceCostMonthly, CostResourceTypeAWSCostExplorerAccountCostMonthly, CostResourceTypeAWSCostExplorerServiceCostDaily, CostResourceTypeAWSCostExplorerAccountCostDaily:
		var costFloat float64
		switch costDescription["AmortizedCostAmount"].(type) {
		case string:
			costFloat, err = strconv.ParseFloat(costDescription["AmortizedCostAmount"].(string), 64)
			if err != nil {
				return 0, ""
			}
		case float64:
			costFloat = costDescription["AmortizedCostAmount"].(float64)
		}
		costUnit, ok := costDescription["AmortizedCostUnit"]
		if !ok {
			return costFloat, "USD"
		}
		return costFloat, costUnit.(string)
	case CostResourceTypeAzureCostManagementCostByResourceType, CostResourceTypeAzureCostManagementCostBySubscription:
		var costFloat float64
		switch costDescription["Cost"].(type) {
		case string:
			costFloat, err = strconv.ParseFloat(costDescription["Cost"].(string), 64)
			if err != nil {
				return 0, ""
			}
		case float64:
			costFloat = costDescription["Cost"].(float64)
		}
		costUnit, ok := costDescription["Currency"]
		if !ok {
			return costFloat, "USD"
		}
		return costFloat, costUnit.(string)
	case CostResourceTypeAWSEBSVolume:
		var desc helpers.EBSCostDescription
		jsonDesc, err := json.Marshal(costDescription)
		if err != nil {
			return 0, ""
		}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return 0, ""
		}
		return desc.GetCost(), "USD"
	}
	return 0, ""
}

func (c CostResourceType) GetCostSummaryAndKey(resource es.Resource, lookupResource es.LookupResource) (CostSummary, string, error) {
	switch c {
	case CostResourceTypeAWSCostExplorerServiceCostMonthly:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := awsModel.CostExplorerByServiceMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%s|%s", resource.SourceID, *desc.Dimension1, *desc.PeriodStart, *desc.PeriodEnd)
		serviceCostSummary := &ServiceCostSummary{
			ServiceName: *desc.Dimension1,
			Cost:        desc,
			PeriodStart: getTimeFromTimestring(*desc.PeriodStart).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimestring(*desc.PeriodEnd).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostServiceSummaryMonthly,
		}
		if serviceCostSummary.ServiceName == "" {
			serviceCostSummary.ServiceName = "AWS Uncategorized"
		}
		return serviceCostSummary, key, nil
	case CostResourceTypeAWSCostExplorerServiceCostDaily:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := awsModel.CostExplorerByServiceDailyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%s|%s", resource.SourceID, *desc.Dimension1, *desc.PeriodStart, *desc.PeriodEnd)
		serviceCostSummary := &ServiceCostSummary{
			ServiceName: *desc.Dimension1,
			Cost:        desc,
			PeriodStart: getTimeFromTimestring(*desc.PeriodStart).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimestring(*desc.PeriodEnd).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostServiceSummaryDaily,
		}
		if serviceCostSummary.ServiceName == "" {
			serviceCostSummary.ServiceName = "AWS Uncategorized"
		}
		return serviceCostSummary, key, nil
	case CostResourceTypeAWSCostExplorerAccountCostMonthly:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := awsModel.CostExplorerByAccountMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		connectionCostSummary := &ConnectionCostSummary{
			AccountID:   *desc.Dimension1,
			Cost:        desc,
			PeriodStart: getTimeFromTimestring(*desc.PeriodStart).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimestring(*desc.PeriodEnd).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostConnectionSummaryMonthly,
		}
		return connectionCostSummary, key, nil
	case CostResourceTypeAWSCostExplorerAccountCostDaily:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := awsModel.CostExplorerByAccountDailyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		connectionCostSummary := &ConnectionCostSummary{
			AccountID:   *desc.Dimension1,
			Cost:        desc,
			PeriodStart: getTimeFromTimestring(*desc.PeriodStart).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimestring(*desc.PeriodEnd).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostConnectionSummaryDaily,
		}
		return connectionCostSummary, key, nil
	case CostResourceTypeAWSEBSVolume:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := awsModel.EC2VolumeDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		region, ok := resource.Metadata["region"]
		if !ok {
			re := regexp.MustCompile(`[a-z]$`)
			region = re.ReplaceAllString(*desc.Volume.AvailabilityZone, "")
			resource.Metadata["region"] = region
		}
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, region, *desc.Volume.VolumeId)
		serviceCostSummary := &ServiceCostSummary{
			ServiceName: string(CostResourceTypeAWSEBSVolume),
			Cost:        desc,
			PeriodStart: time.UnixMilli(lookupResource.CreatedAt).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   time.UnixMilli(lookupResource.CreatedAt).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostServiceSummaryDaily,
			Region:      &region,
		}
		return serviceCostSummary, key, nil
	case CostResourceTypeAzureCostManagementCostByResourceType:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := azureModel.CostManagementCostByResourceTypeDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%s|%d", resource.SourceID, *desc.CostManagementCostByResourceType.ResourceType, desc.CostManagementCostByResourceType.Currency, desc.CostManagementCostByResourceType.UsageDate)
		serviceCostSummary := &ServiceCostSummary{
			ServiceName: *desc.CostManagementCostByResourceType.ResourceType,
			Cost:        desc.CostManagementCostByResourceType,
			PeriodStart: getTimeFromTimeInt(desc.CostManagementCostByResourceType.UsageDate).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimeInt(desc.CostManagementCostByResourceType.UsageDate).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostServiceSummaryDaily,
		}
		if serviceCostSummary.ServiceName == "" {
			serviceCostSummary.ServiceName = "Azure Uncategorized"
		}
		return serviceCostSummary, key, nil
	case CostResourceTypeAzureCostManagementCostBySubscription:
		jsonDesc, err := json.Marshal(resource.Description)
		if err != nil {
			return nil, "", err
		}
		desc := azureModel.CostManagementCostBySubscriptionDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return nil, "", err
		}
		key := fmt.Sprintf("%s|%s|%d", resource.SourceID, desc.CostManagementCostBySubscription.Currency, desc.CostManagementCostBySubscription.UsageDate)
		connectionCostSummary := &ConnectionCostSummary{
			AccountID:   *desc.CostManagementCostBySubscription.SubscriptionID,
			Cost:        desc.CostManagementCostBySubscription,
			PeriodStart: getTimeFromTimeInt(desc.CostManagementCostBySubscription.UsageDate).Truncate(24 * time.Hour).Unix(),
			PeriodEnd:   getTimeFromTimeInt(desc.CostManagementCostBySubscription.UsageDate).Truncate(24 * time.Hour).Unix(),
			ReportType:  CostConnectionSummaryDaily,
		}
		return connectionCostSummary, key, nil
	}
	return nil, "", fmt.Errorf("unknown resource type %s", resource.ResourceType)
}

type CostSummary interface {
	GetCostAndUnit() (float64, string)
	KeysAndIndex() ([]string, string)
}

type ServiceCostSummary struct {
	SummarizeJobTime int64              `json:"summarize_job_time"`
	SummarizeJobID   uint               `json:"summarize_job_id"`
	ServiceName      string             `json:"service_name"`
	SourceID         string             `json:"source_id"`
	Connector        source.Type        `json:"source_type"`
	SourceJobID      uint               `json:"source_job_id"`
	ResourceType     string             `json:"resource_type"`
	Cost             any                `json:"cost"`
	CostValue        float64            `json:"cost_value"`
	PeriodStart      int64              `json:"period_start"`
	PeriodEnd        int64              `json:"period_end"`
	ReportType       ProviderReportType `json:"report_type"`
	Region           *string            `json:"region,omitempty"`
}

func (c ServiceCostSummary) GetCostAndUnit() (float64, string) {
	costResourceType := GetCostResourceTypeFromString(c.ResourceType)
	if costResourceType != CostResourceTypeNull {
		return costResourceType.GetCostAndUnitFromResource(c.Cost)
	}
	return 0, ""
}

func (c ServiceCostSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		c.ServiceName,
		c.SourceID,
		c.ResourceType,
		fmt.Sprint(c.PeriodStart),
		fmt.Sprint(c.PeriodEnd),
	}

	costResourceType := GetCostResourceTypeFromString(c.ResourceType)
	keys = append(keys, string(costResourceType.GetProviderReportType()))
	if c.Region != nil {
		keys = append(keys, *c.Region)
	}

	return keys, CostSummeryIndex
}

type ConnectionCostSummary struct {
	SummarizeJobTime int64                `json:"summarize_job_time"`
	SummarizeJobID   uint                 `json:"summarize_job_id"`
	AccountID        string               `json:"account_id"`
	SourceID         string               `json:"source_id"`
	SourceType       source.Type          `json:"source_type"`
	SourceJobID      uint                 `json:"source_job_id"`
	ResourceType     string               `json:"resource_type"`
	Cost             any                  `json:"cost"`
	CostValue        float64              `json:"cost_value"`
	PeriodStart      int64                `json:"period_start"`
	PeriodEnd        int64                `json:"period_end"`
	ReportType       ConnectionReportType `json:"report_type"`
}

func (c ConnectionCostSummary) GetCostAndUnit() (float64, string) {
	costResourceType := GetCostResourceTypeFromString(c.ResourceType)
	if costResourceType != CostResourceTypeNull {
		return costResourceType.GetCostAndUnitFromResource(c.Cost)
	}
	return 0, ""
}

func (c ConnectionCostSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		c.AccountID,
		c.SourceID,
		c.ResourceType,
		fmt.Sprint(c.PeriodStart),
		fmt.Sprint(c.PeriodEnd),
	}

	costResourceType := GetCostResourceTypeFromString(c.ResourceType)
	keys = append(keys, string(costResourceType.GetProviderReportType()))

	return keys, CostSummeryIndex
}
