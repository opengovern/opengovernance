package price

import "github.com/shopspring/decimal"

type Price struct {
	SKU       string
	Currency  string
	PriceUnit string
	Price     decimal.Decimal
}
