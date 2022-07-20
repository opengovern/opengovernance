package cloudservice

import (
	"encoding/csv"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

var cloudServices []CloudService = nil
var resourceList []ResourceList = nil

func initCloudService() {
	if cloudServices == nil {
		parseCSV()
	}
}

func parseCSV() error {
	reader := csv.NewReader(strings.NewReader(awsCloudServicesCSV))
	cells, err := reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		if strings.TrimSpace(strings.ToLower(row[3])) != "yes" {
			continue
		}

		cloudServices = append(cloudServices, CloudService{
			Provider:         source.CloudAWS,
			Category:         row[0],
			FullServiceName:  row[1],
			ServiceNamespace: row[2],
		})
	}

	reader = csv.NewReader(strings.NewReader(azureCloudServicesCSV))
	cells, err = reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		cloudServices = append(cloudServices, CloudService{
			Provider:         source.CloudAzure,
			Category:         row[0],
			FullServiceName:  row[2],
			ServiceNamespace: row[1],
		})
	}

	reader = csv.NewReader(strings.NewReader(awsResourceListCSV))
	cells, err = reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		resourceList = append(resourceList, ResourceList{
			Provider:         source.CloudAWS,
			ResourceTypeName: row[1],
			ServiceNamespace: row[0],
		})
	}

	reader = csv.NewReader(strings.NewReader(azureResourceListCSV))
	cells, err = reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		resourceList = append(resourceList, ResourceList{
			Provider:         source.CloudAzure,
			ResourceTypeName: row[0],
			ServiceNamespace: row[1],
		})
	}

	return nil
}

func findProvider(resourceType string) source.Type {
	resourceType = strings.ToLower(resourceType)
	var provider source.Type
	if strings.HasPrefix(resourceType, "aws") {
		provider = source.CloudAWS
	} else if strings.HasPrefix(resourceType, "microsoft") {
		provider = source.CloudAzure
	}
	return provider
}

func findResourceListRecord(resourceType string) *ResourceList {
	initCloudService()
	resourceType = strings.ToLower(resourceType)
	provider := findProvider(resourceType)
	for _, v := range resourceList {
		if v.Provider != provider {
			continue
		}

		var recordResourceType string
		if provider == source.CloudAWS {
			recordResourceType = ParseARN(v.ServiceNamespace).Type()
		} else {
			recordResourceType = strings.ToLower(v.ServiceNamespace)
		}
		if strings.HasPrefix(resourceType, recordResourceType) {
			return &v
		}
	}
	return nil
}

func findCloudServiceRecord(resourceType string) *CloudService {
	initCloudService()
	resourceType = strings.ToLower(resourceType)
	provider := findProvider(resourceType)
	for _, v := range cloudServices {
		if v.Provider != provider {
			continue
		}

		var recordResourceType string
		if provider == source.CloudAWS {
			recordResourceType = ParseARN(v.ServiceNamespace).Type()
		} else {
			recordResourceType = strings.ToLower(v.ServiceNamespace)
		}
		if strings.HasPrefix(resourceType, recordResourceType) {
			return &v
		}
	}
	return nil
}

func CategoryByResourceType(resourceType string) string {
	if r := findCloudServiceRecord(resourceType); r != nil {
		return r.Category
	}
	return ""
}

func ServiceNameByResourceType(resourceType string) string {
	if record := findResourceListRecord(resourceType); record != nil {
		return record.ResourceTypeName
	}
	if record := findCloudServiceRecord(resourceType); record != nil {
		return record.FullServiceName
	}
	return ""
}

func IsCommonByResourceType(resourceType string) bool {
	if record := findResourceListRecord(resourceType); record != nil {
		return true
	}
	return false
}
