package calculator

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
	NextPageLink       string
	Count              int
}
