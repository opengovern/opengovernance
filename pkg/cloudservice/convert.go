package cloudservice

import (
	"encoding/csv"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

var cloudServices []CloudService = nil
var resourceList []ResourceList = nil

var categories []Category = nil
var cloudResources []CloudResource = nil

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

	reader = csv.NewReader(strings.NewReader(categoriesCSV))
	cells, err = reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		categories = append(categories, Category{
			Category:     row[0],
			SubCategory:  row[1],
			Cloud:        row[2],
			CloudService: row[3],
		})
	}

	reader = csv.NewReader(strings.NewReader(cloudResourcesCSV))
	cells, err = reader.ReadAll()
	if err != nil {
		return err
	}
	// remove header
	cells = cells[1:]
	for _, row := range cells {
		t, err := source.ParseType(row[0])
		if err != nil {
			return err
		}

		cloudResources = append(cloudResources, CloudResource{
			Cloud:                     t,
			CloudService:              row[1],
			ResourceTypeName:          row[2],
			ResourceProviderNamespace: row[3],
		})
	}

	return nil
}

func findCloudResourceRecord(resourceType string) *CloudResource {
	initCloudService()
	resourceType = strings.ToLower(resourceType)
	provider := findProvider(resourceType)
	for _, v := range cloudResources {
		if v.Cloud != provider {
			continue
		}

		var recordResourceType string
		if provider == source.CloudAWS {
			recordResourceType = ParseARN(v.ResourceProviderNamespace).Type()
		} else {
			recordResourceType = strings.ToLower(v.ResourceProviderNamespace)
		}
		if strings.HasPrefix(resourceType, recordResourceType) {
			return &v
		}
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
	return "Others"
}

func ServiceNameByResourceType(resourceType string) string {
	if record := findCloudResourceRecord(resourceType); record != nil {
		return record.CloudService
	}
	return resourceType
}

func ResourceTypeName(resourceType string) string {
	if record := findCloudResourceRecord(resourceType); record != nil {
		return record.ResourceTypeName
	}

	if record := findResourceListRecord(resourceType); record != nil {
		return record.ResourceTypeName
	}
	if record := findCloudServiceRecord(resourceType); record != nil {
		return record.FullServiceName + " Resource"
	}
	return ""
}

func IsCommonByResourceType(resourceType string) bool {
	if record := findResourceListRecord(resourceType); record != nil {
		return true
	}
	return false
}

func ListCategories() []string {
	initCloudService()

	m := map[string]interface{}{}
	for _, v := range cloudServices {
		m[v.Category] = true
	}

	var cat []string
	for k := range m {
		cat = append(cat, k)
	}
	return cat
}

func ResourceListByCategory(category string) []string {
	initCloudService()

	var res []string
	for _, v := range cloudServices {
		if v.Category == category {
			var vtype string
			if v.Provider == source.CloudAWS {
				vtype = ParseARN(v.ServiceNamespace).Type()
			} else {
				vtype = strings.ToLower(v.ServiceNamespace)
			}
			for _, r := range resourceList {
				if r.Provider != v.Provider {
					continue
				}

				var rtype string
				if r.Provider == source.CloudAWS {
					rtype = ParseARN(r.ServiceNamespace).Type()
				} else {
					rtype = strings.ToLower(r.ServiceNamespace)
				}

				if strings.HasPrefix(rtype, vtype) {
					res = append(res, rtype)
				}
			}
		}
	}
	return res
}

func ResourceListByServiceName(serviceName string) []string {
	initCloudService()
	var response []string
	for _, v := range resourceList {
		srv := findCloudResourceRecord(v.ServiceNamespace)
		if srv != nil && srv.CloudService == serviceName {
			response = append(response, v.ServiceNamespace)
		}
	}
	return response
}
