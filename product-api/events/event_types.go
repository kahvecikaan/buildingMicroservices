package events

type PriceUpdate struct {
	ProductID int     `json:"product_id"`
	NewPrice  float64 `json:"new_price"`
	Currency  string  `json:"currency"`
}

// We can define other event types here
// type StockUpdate struct {
//     ProductID int `json:"product_id"`
//     NewStock  int `json:"new_stock"`
// }
