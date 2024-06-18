package model

import (
	"google.golang.org/api/cloudbilling/v1"
	"gorm.io/gorm"
	"regexp"
	"strings"
)

const (
	ram = "RAM"
	cpu = "CPU"
	gpu = "GPU"
)

type GCPComputeSKU struct {
	gorm.Model

	// Basic fields
	SKU                string `gorm:"index"`
	ResourceFamily     string `gorm:"index"`
	ResourceGroup      string `gorm:"index"`
	ServiceDisplayName string `gorm:"index"`
	UsageType          string `gorm:"index"`
	Location           string `gorm:"index"`
	Type               string `gorm:"index"`

	Description   string
	MachineFamily string

	UnitPrice    float64
	CurrencyCode string
}

func (p *GCPComputeSKU) PopulateFromObject(sku *cloudbilling.Sku, region string) {
	p.Location = region
	p.SKU = sku.SkuId
	if sku.Category != nil {
		p.ResourceFamily = sku.Category.ResourceFamily
		p.ResourceGroup = sku.Category.ResourceGroup
		p.ServiceDisplayName = sku.Category.ServiceDisplayName
		p.UsageType = sku.Category.UsageType
	}
	p.Description = sku.Description
	p.MachineFamily, p.ResourceGroup, p.Type = GetSkuDetails(sku)
	pe := sku.PricingInfo[len(sku.PricingInfo)-1].PricingExpression
	p.UnitPrice = float64(pe.TieredRates[0].UnitPrice.Units) +
		(float64(pe.TieredRates[0].UnitPrice.Nanos) / float64(1_000_000_000))
	p.CurrencyCode = pe.TieredRates[0].UnitPrice.CurrencyCode
}

// GetSkuDetails returns 'Machine Family', 'Resource Group', 'Type'
func GetSkuDetails(sku *cloudbilling.Sku) (string, string, string) {
	if sku.Category == nil {
		return "", "", ""
	}
	if sku.Category.ResourceGroup == cpu || sku.Category.ResourceGroup == ram || sku.Category.ResourceGroup == gpu {
		mf := strings.ToLower(strings.Split(sku.Description, " ")[0])
		if mf == "n4" || mf == "e2" || mf == "n2" || mf == "c3" || mf == "c3d" || mf == "n2d" ||
			mf == "t2d" || mf == "t2a" || mf == "h3" || mf == "c2" || mf == "c2d" || mf == "m3" || mf == "m2" ||
			mf == "m1" || mf == "z3" || mf == "a3" || mf == "a3plus" || mf == "a2" || mf == "g2" {
			reST := regexp.MustCompile(`^.* Sole Tenancy Instance (Core|Ram) running in .*$`)
			if reST.MatchString(sku.Description) {
				return mf, sku.Category.ResourceGroup, ""
			}
			reCustomExt := regexp.MustCompile(`^.* Custom Extended Instance (Core|Ram) running in .*$`)
			if reCustomExt.MatchString(sku.Description) {
				return mf, sku.Category.ResourceGroup, "Custom Extended"
			}
			reCustomExt = regexp.MustCompile(`^.* Custom Extended (Core|Ram) running in .*$`)
			if reCustomExt.MatchString(sku.Description) {
				return mf, sku.Category.ResourceGroup, "Custom Extended"
			}
			reCustom := regexp.MustCompile(`^.* Custom Instance (Core|Ram) running in .*$`)
			if reCustom.MatchString(sku.Description) {
				return mf, sku.Category.ResourceGroup, "Custom"
			}
			re := regexp.MustCompile(`^.* Instance (Core|Ram) running in .*$`)
			if re.MatchString(sku.Description) {
				return mf, sku.Category.ResourceGroup, "Predefined"
			}
			return mf, sku.Category.ResourceGroup, ""
		}
	}

	if sku.Category.ResourceGroup == "N1Standard" {
		reCore := regexp.MustCompile(`^N1 Predefined Instance Core running in .*$`)
		if reCore.MatchString(sku.Description) {
			return "n1", cpu, "Predefined"
		}
		reRam := regexp.MustCompile(`^N1 Predefined Instance Ram running in .*$`)
		if reRam.MatchString(sku.Description) {
			return "n1", ram, "Predefined"
		}
	}

	return "", sku.Category.ResourceGroup, ""
}
