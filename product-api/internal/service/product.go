package service

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/events"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/repository"
	"sync"
)

type ProductService interface {
	GetProducts(ctx context.Context, currency string) (Products, error)
	GetProductByID(ctx context.Context, id int, currency string) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) error
	AddProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id int) error
	ListCurrencies(ctx context.Context) ([]string, error)
	Close() error
}

type productService struct {
	repo            repository.ProductRepository
	currencyService CurrencyService
	eventBus        *events.EventBus[any]
	logger          hclog.Logger
	rateSubscriber  events.Subscriber[any]
	wg              sync.WaitGroup
	once            sync.Once
}

type Products []*domain.Product

func NewProductService(
	repo repository.ProductRepository,
	currencyService CurrencyService,
	eventBus *events.EventBus[any],
	logger hclog.Logger) ProductService {
	ps := &productService{
		repo:            repo,
		currencyService: currencyService,
		eventBus:        eventBus,
		logger:          logger,
	}

	// Subscribe to events
	ps.rateSubscriber = eventBus.Subscribe()

	// Start handling rate change events
	ps.wg.Add(1)
	go ps.handleRateChanges()

	return ps
}

func (s *productService) handleRateChanges() {
	defer s.wg.Done()
	for event := range s.rateSubscriber {
		if rateEvent, ok := event.(events.RateChanged); ok {
			s.logger.Debug("Received rate changed event",
				"currency", rateEvent.Currency,
				"new_rate", rateEvent.NewRate)

			// Get all products
			ctx := context.Background()
			products, err := s.repo.GetAll(ctx)
			if err != nil {
				s.logger.Error("Failed to get products for price updates", "error", err)
				continue
			}

			// Update prices and publish events for each product
			for _, product := range products {
				newPrice := product.Price * rateEvent.NewRate

				// Publish price update event
				s.eventBus.Publish(events.PriceUpdate{
					ProductID: product.ID,
					NewPrice:  newPrice,
					Currency:  rateEvent.Currency,
				})
			}
		}

	}
}

func (s *productService) GetProducts(ctx context.Context, currency string) (Products, error) {
	s.logger.Debug("Getting all products", "currency", currency)

	products, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("Unable to get products", "error", err)
		return nil, err
	}

	if currency == "" {
		return products, nil
	}

	rate, err := s.currencyService.GetRate(ctx, "EUR", currency)
	if err != nil {
		s.logger.Error("Unable to get currency rate", "currency", currency, "error", err)
		return nil, err
	}

	// Create a copy of the products with updated prices
	productCopies := make(Products, len(products))
	for i, product := range products {
		productCopy := *product
		productCopy.Price = productCopy.Price * rate
		productCopies[i] = &productCopy
	}

	return productCopies, nil
}

func (s *productService) GetProductByID(ctx context.Context, id int, currency string) (*domain.Product, error) {
	s.logger.Debug("Getting product by ID", "id", id)

	product, err := s.repo.GetById(ctx, id)
	if err != nil {
		s.logger.Error("Unable to get the product by ID", "id", id, "error", err)
		return nil, err
	}

	if currency == "" {
		return product, nil
	}

	rate, err := s.currencyService.GetRate(ctx, "EUR", currency)
	if err != nil {
		s.logger.Error("Unable to get currency rate", "currency", currency, "error", err)
		return nil, err
	}

	// Create a copy of the product with updated price
	productCopy := *product
	productCopy.Price = productCopy.Price * rate
	return &productCopy, nil
}

func (s *productService) UpdateProduct(ctx context.Context, product *domain.Product) error {
	s.logger.Debug("Updating product", "id", product.ID)

	err := s.repo.Update(ctx, product)
	if err != nil {
		s.logger.Error("Unable to update product", "id", product.ID, "error", err)
		return err
	}

	// Publish an event for product update
	s.eventBus.Publish(events.ProductUpdated{ProductID: product.ID})
	return nil
}

func (s *productService) AddProduct(ctx context.Context, product *domain.Product) error {
	s.logger.Debug("Adding new product", "name", product.Name)

	err := s.repo.Add(ctx, product)
	if err != nil {
		s.logger.Error("Unable to add product", "name", product.Name, "error", err)
		return err
	}

	// Publish an event for product addition
	s.eventBus.Publish(events.ProductAdded{ProductID: product.ID})
	return nil
}

func (s *productService) DeleteProduct(ctx context.Context, id int) error {
	s.logger.Debug("Deleting product", "id", id)

	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("Unable to delete product", "id", id, "error", err)
		return err
	}

	// Publish an event for product deletion
	s.eventBus.Publish(events.ProductDeleted{ProductID: id})
	return nil
}

func (s *productService) ListCurrencies(ctx context.Context) ([]string, error) {
	s.logger.Debug("Listing available currencies")

	currencies, err := s.currencyService.ListAvailableCurrencies(ctx)
	if err != nil {
		s.logger.Error("Unable to list available currencies", "error", err)
		return nil, err
	}

	return currencies, err
}

func (s *productService) Close() error {
	s.once.Do(func() {
		s.logger.Info("Shutting down ProductService...")

		// Unsubscribe from the event bus to stop receiving events
		if s.rateSubscriber != nil {
			s.eventBus.Unsubscribe(s.rateSubscriber)
			s.rateSubscriber = nil
			s.logger.Info("Unsubscribed from rate change events")
		}

		// Wait for handleRateChanges goroutine to finish
		s.wg.Wait()

		s.logger.Info("ProductService shutdown complete.")
	})

	return nil
}
