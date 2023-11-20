package postgresql

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
)

func parseProductFilter(filter *product.Filter) string {
	query := ""
	if *filter.Provider == "AWS" {
		query = query + fmt.Sprintf("region_code = %s", *filter.Location)
	} else if *filter.Provider == "Azure" {
		query = query + fmt.Sprintf("arm_region_name = %s", *filter.Location)
	}

	for _, f := range filter.AttributeFilters {
		if f.Value != nil {
			query = query + fmt.Sprintf("%s = '%s'", f.Key, *f.Value)
		} else if f.ValueRegex != nil {
			query = query + fmt.Sprintf("%s LIKE '%s'", f.Key, *f.ValueRegex)
		}
	}

	return query
}
