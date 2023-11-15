package postgresql

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"gorm.io/gorm"
)

func parseProductFilter(q *gorm.DB, filter *product.Filter) *gorm.DB {
	if *filter.Provider == "AWS" {
		q = q.Where("region_code = ?", filter.Location)
	} else if *filter.Provider == "Azure" {
		q = q.Where("arm_region_name = ?", filter.Location)
	}

	for _, f := range filter.AttributeFilters {
		if f.Value != nil {
			q = q.Where(fmt.Sprintf("%s = ?", f.Key), f.Value)
		} else if f.ValueRegex != nil {
			q = q.Where(fmt.Sprintf("%s LIKE '%%?%%'", f.Key), f.Value)
		}
	}

	return q
}
