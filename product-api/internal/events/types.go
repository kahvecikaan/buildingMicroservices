package events

type ProductAdded struct {
	ProductID int `json:"product_id"`
}

type ProductUpdated struct {
	ProductID int `json:"product_id"`
}

type ProductDeleted struct {
	ProductID int `json:"product_id"`
}

type PriceUpdate struct {
	ProductID int     `json:"product_id"`
	NewPrice  float64 `json:"new_price"`
	Currency  string  `json:"currency"`
}

type RateChanged struct {
	Currency string  `json:"currency"`
	NewRate  float64 `json:"new_rate"`
}
