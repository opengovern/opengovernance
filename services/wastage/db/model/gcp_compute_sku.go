package model

import (
	"google.golang.org/api/cloudbilling/v1"
	"gorm.io/gorm"
	"strings"
)

const (
	ram = "RAM"
	cpu = "CPU"
)

type GCPComputeSKU struct {
	gorm.Model

	// Basic fields
	SKU                string `gorm:"index"`
	ResourceFamily     string `gorm:"index"`
	ResourceGroup      string `gorm:"index"`
	ServiceDisplayName string `gorm:"index"`
	UsageType          string `gorm:"index"`
	location           string `gorm:"index"`

	Description   string
	MachineFamily string

	UnitPrice    float64
	CurrencyCode string
}

func (p *GCPComputeSKU) PopulateFromObject(sku *cloudbilling.Sku, region string) {
	p.location = region
	p.SKU = sku.SkuId
	if sku.Category != nil {
		p.ResourceFamily = sku.Category.ResourceFamily
		p.ResourceGroup = sku.Category.ResourceGroup
		p.ServiceDisplayName = sku.Category.ServiceDisplayName
		p.UsageType = sku.Category.UsageType
	}
	p.Description = sku.Description
	if p.ResourceGroup == cpu || p.ResourceGroup == ram {
		mf := strings.ToLower(strings.Split(sku.Description, " ")[0])
		p.MachineFamily = mf
	}
	p.UnitPrice = float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Units) +
		(float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos) / float64(1_000_000_000))
	p.CurrencyCode = sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.CurrencyCode
}
