package price

import "github.com/shopspring/decimal"

type Price struct {
	SKU       string
	PriceUnit string
	Price     decimal.Decimal
}
