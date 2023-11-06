package es

type ItemsStr struct {
	CurrencyCode         string
	TierMinimumUnits     float64
	RetailPrice          float64
	UnitPrice            float64
	ArmRegionName        string
	Location             string
	EffectiveStartDate   string
	MeterId              string
	MeterName            string
	ProductId            string
	SkuId                string
	ProductName          string
	SkuName              string
	ServiceName          string
	ServiceId            string
	ServiceFamily        string
	UnitOfMeasure        string
	Type                 string
	IsPrimaryMeterRegion bool
	ArmSkuName           string
}

type AzureCostStr struct {
	BillingCurrency    string
	CustomerEntityId   string
	CustomerEntityType string
	Items              []ItemsStr
	NextPageLink       *string
	Count              int
}

func (i ItemsStr) KeysAndIndex() ([]string, string) {
	if i.ServiceName == "Virtual Machines" && i.Type == "Consumption" && i.ServiceFamily == "Compute" {
		return []string{
			i.ServiceName,
			i.Type,
			i.ServiceFamily,
			i.ArmSkuName,
			i.ArmRegionName,
		}, "azure_cost_table"
	} else {
		return []string{}, "azure_cost_table"
	}
}
