package data

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"github.com/kahvecikaan/buildingMicroservices/product-api/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"sync"
	"time"
)

// ErrProductNotFound is an error raised when a product can't be found in the database
var ErrProductNotFound = fmt.Errorf("product not found")

// Product defines the structure for an API product
// swagger:model
type Product struct {
	// the id for this user
	//
	// required: true
	// min: 1
	ID int `json:"id"`

	// the name for this product
	//
	// required: true
	// max length: 255
	Name string `json:"name" validate:"required"`

	// the description for this product
	//
	// required: false
	// max length: 10000
	Description string `json:"description"`

	// the price for the product
	//
	// required: true
	// min: 0.01
	Price float64 `json:"price" validate:"gt=0"`

	// the sku for this product
	//
	// required: true
	// pattern: [a-z]+-[a-z]+-[a-z]+
	SKU string `json:"sku" validate:"required,sku"`
}

// Products is a collection of Product
type Products []*Product

type ProductsDB struct {
	log            hclog.Logger
	currencyClient protos.CurrencyClient
	rates          map[string]float64
	stream         protos.Currency_SubscribeRatesClient
	closeCh        chan struct{}
	ratesMutex     sync.RWMutex
	eventBus       *events.EventBus[any]
}

func NewProductsDB(log hclog.Logger, c protos.CurrencyClient, eventBus *events.EventBus[any]) *ProductsDB {
	db := &ProductsDB{
		log:            log,
		currencyClient: c,
		rates:          make(map[string]float64),
		closeCh:        make(chan struct{}),
		eventBus:       eventBus,
	}

	go db.handleUpdates()

	return db
}

func (db *ProductsDB) handleUpdates() {
	// establish the bidirectional stream
	stream, err := db.currencyClient.SubscribeRates(context.Background())
	if err != nil {
		db.log.Error("Unable to subscribe for rates", "error", err)
		return
	}

	db.stream = stream

	// start a ticker to send heartbeat messages every minute
	heartbeatTicker := time.NewTicker(1 * time.Minute)
	defer heartbeatTicker.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-heartbeatTicker.C:
				err := db.sendHeartbeat()
				if err != nil {
					db.log.Info("Error sending heartbeat", "error", err)
				}
			case <-db.closeCh:
				return
			}
		}
	}()

	for {
		select {
		case <-db.closeCh:
			// exit if close signal is received
			return
		default:
			response, err := stream.Recv()
			if err == io.EOF {
				db.log.Info("Server closed the stream")
				return
			}
			if err != nil {
				db.log.Error("Error receiving message from the stream", "error", err)
				return
			}

			switch msg := response.Message.(type) {
			case *protos.StreamingRateResponse_RateResponse:
				rateResponse := msg.RateResponse
				db.log.Info(
					"Received rate update",
					"destination", rateResponse.GetDestination().String(),
					"rate", rateResponse.GetRate())

				currency := rateResponse.GetDestination().String()
				newRate := rateResponse.GetRate()

				// update the rates in a thread-safe manner
				db.ratesMutex.Lock()
				db.rates[currency] = newRate
				db.ratesMutex.Unlock()

				// publish price updates for products priced in this currency
				db.publishPriceUpdates(currency)

			case *protos.StreamingRateResponse_Error:
				errorStatus := msg.Error
				db.log.Error("Received error from server", "error", errorStatus.GetMessage())

				// handle specific error codes if needed
				grpcErr := status.FromProto(errorStatus)
				switch grpcErr.Code() {
				case codes.InvalidArgument:
					db.log.Error("Invalid argument error received", "details", grpcErr.Message())
				default:
					db.log.Error("Error received", "code", grpcErr.Code(), "message", grpcErr.Message())
				}
			}
		}
	}

	wg.Wait() // wait for heartbeat routine to finish
}

func (db *ProductsDB) publishPriceUpdates(currency string) {
	db.log.Info("Publish price updates", "currency", currency)

	db.ratesMutex.RLock()
	rate, ok := db.rates[currency]
	db.ratesMutex.RUnlock()

	if !ok {
		db.log.Error("Rate not found for currency", "currency", currency)
		return
	}

	// iterate over products and publish updates
	for _, product := range productList {
		// assume products are priced in EUR and convert to destination
		newPrice := product.Price * rate

		// create a PriceUpdate event
		update := events.PriceUpdate{
			ProductID: product.ID,
			NewPrice:  newPrice,
			Currency:  currency,
		}

		// publish the event
		db.eventBus.Publish(update)
	}
}

func (db *ProductsDB) sendHeartbeat() error {
	heartbeat := &protos.RateRequest{
		Base:        protos.Currencies_UNKNOWN,
		Destination: protos.Currencies_UNKNOWN,
	}

	err := db.stream.Send(heartbeat)
	if err != nil {
		return err
	}

	db.log.Info("Sent heartbeat to server")
	return nil
}

// GetProducts returns all products from the database
func (db *ProductsDB) GetProducts(currency string) (Products, error) {
	if currency == "" {
		return productList, nil
	}

	rate, err := db.getRate(currency)
	if err != nil {
		db.log.Error("Unable to get the rate", "currency", currency, "error", err)
		return nil, err
	}

	prodList := Products{}
	for _, p := range productList {
		// a copy of p
		np := *p
		np.Price = np.Price * rate
		prodList = append(prodList, &np)
	}

	return prodList, nil
}

// GetProductByID returns a single product which matches the id from the
// database.
// If a product is not found this function returns a ProductNotFound error
func (db *ProductsDB) GetProductByID(id int, currency string) (*Product, error) {
	i := findIndexByProductID(id)
	if i == -1 {
		return nil, ErrProductNotFound
	}

	if currency == "" {
		return productList[i], nil
	}

	rate, err := db.getRate(currency)
	if err != nil {
		db.log.Error("Unable to get the rate", "currency", currency, "error", err)
		return nil, err
	}

	np := *productList[i]
	np.Price = np.Price * rate

	return &np, nil
}

// UpdateProduct replaces a product in the database with the given
// item.
// If a product with the given id does not exist in the database
// this function returns a ProductNotFound error
func (db *ProductsDB) UpdateProduct(pr *Product) error {
	i := findIndexByProductID(pr.ID)
	if i == -1 {
		return ErrProductNotFound
	}

	// update the product in the DB
	productList[i] = pr

	return nil
}

// AddProduct adds a product to the database
func (db *ProductsDB) AddProduct(pr *Product) {
	pr.ID = getNextID()
	productList = append(productList, pr)
}

func (db *ProductsDB) DeleteProduct(id int) error {
	i := findIndexByProductID(id)
	if i == -1 {
		return ErrProductNotFound
	}

	productList = append(productList[:i], productList[i+1:]...)

	return nil
}

func (db *ProductsDB) Close() {
	close(db.closeCh)
	if db.stream != nil {
		if err := db.stream.CloseSend(); err != nil {
			db.log.Error("Error closing stream", "error", err)
		}
	}
}

// findIndex finds the index of a product in the database
// returns -1 when no product can be found
func findIndexByProductID(id int) int {
	for i, p := range productList {
		if p.ID == id {
			return i
		}
	}

	return -1
}

func getNextID() int {
	lp := productList[len(productList)-1]
	return lp.ID + 1
}

func (db *ProductsDB) getRate(destination string) (float64, error) {
	// validate the destination currency
	currencyValue, ok := protos.Currencies_value[destination]
	if !ok {
		errMsg := fmt.Sprintf("Invalid destination currency: %s", destination)
		db.log.Error(errMsg)
		return -1, fmt.Errorf(errMsg)
	}
	destinationCurrency := protos.Currencies(currencyValue)

	// check if rate is cached
	db.ratesMutex.RLock()
	rate, ok := db.rates[destination]
	db.ratesMutex.RUnlock()
	if ok {
		return rate, nil
	}

	rateRequest := &protos.RateRequest{
		// base rate is always EUR (since products are priced in EUR by default)
		Base:        protos.Currencies(protos.Currencies_value["EUR"]),
		Destination: destinationCurrency,
	}

	// get initial rate via unary RPC
	resp, err := db.currencyClient.GetRate(context.Background(), rateRequest)
	if err != nil {
		// convert the gRPC error message
		grpcErr, ok := status.FromError(err)
		if !ok {
			// not a gRPC error
			db.log.Error("Non-gRPC error calling GetRate", "error", err)
			return -1, err
		}

		// handle specific gRPC error codes
		switch grpcErr.Code() {
		case codes.InvalidArgument:
			db.log.Error("Invalid arguments when calling GetRate", "error", grpcErr.Message())
			return -1, fmt.Errorf("invalid argument: %s", grpcErr.Message())
		case codes.NotFound:
			db.log.Error("Rate not found", "error", grpcErr.Message())
			return -1, fmt.Errorf("rate not found: %s", grpcErr.Message())
		default:
			db.log.Error("Error calling GetRate", grpcErr.Message())
			return -1, fmt.Errorf("error getting rate: %s", grpcErr.Message())
		}
	}

	// update the rate in a thread-safe manner
	db.ratesMutex.Lock()
	db.rates[destination] = resp.Rate
	db.ratesMutex.Unlock()

	// subscribe for updates
	err = db.stream.Send(rateRequest)
	if err != nil {
		// handle gRPC errors when sending subscription
		db.log.Error("Error subscribing for updates", "error", err)
		grpcErr, ok := status.FromError(err)
		if ok {
			switch grpcErr.Code() {
			case codes.InvalidArgument:
				db.log.Error("Invalid arguments when subscribing", "error", grpcErr.Message())
				return -1, fmt.Errorf("invalid argument: %s", grpcErr.Message())
			default:
				db.log.Error("Error subscribing for updates", "code", grpcErr.Code(), "error", grpcErr.Message())
				return -1, fmt.Errorf("error subscribing for updates: %s", grpcErr.Message())
			}
		}

		return -1, err
	}
	return resp.Rate, nil
}

var productList = []*Product{
	{
		ID:          1,
		Name:        "Latte",
		Description: "Frothy milky coffee",
		Price:       2.45,
		SKU:         "abc323",
	},
	{
		ID:          2,
		Name:        "Espresso",
		Description: "Short and strong coffee without milk",
		Price:       1.99,
		SKU:         "fjd34",
	},
}
