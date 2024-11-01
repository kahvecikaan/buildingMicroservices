package service

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/events"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

type CurrencyService interface {
	GetRate(ctx context.Context, base, destination string) (float64, error)
	SubscribeToRates(ctx context.Context, currencies []string) error
	ListAvailableCurrencies(ctx context.Context) ([]string, error)
	Close() error
}

type currencyService struct {
	log           hclog.Logger
	client        protos.CurrencyClient
	rates         map[string]float64
	ratesMutex    sync.RWMutex
	stream        protos.Currency_SubscribeRatesClient
	subscriptions map[string]struct{}
	subMutex      sync.RWMutex
	closeCh       chan struct{}
	eventBus      *events.EventBus[any]
	once          sync.Once
	wg            sync.WaitGroup
}

func NewCurrencyService(
	logger hclog.Logger,
	client protos.CurrencyClient,
	eventBus *events.EventBus[any]) CurrencyService {
	svc := &currencyService{
		log:           logger,
		client:        client,
		rates:         make(map[string]float64),
		subscriptions: make(map[string]struct{}),
		closeCh:       make(chan struct{}),
		eventBus:      eventBus,
	}

	// Initialize the stream
	if err := svc.initializeStream(context.Background()); err != nil {
		logger.Error("Failed to initialize the stream", "error", err)
	}

	// Start handling rate updates and heartbeat
	svc.wg.Add(2)
	go svc.handleRateUpdates()
	go svc.handleHeartbeat()

	return svc
}

func (s *currencyService) initializeStream(ctx context.Context) error {
	stream, err := s.client.SubscribeRates(ctx)
	if err != nil {
		s.log.Error("Error establishing subscription stream", "error", err)
		return err
	}
	s.stream = stream
	return nil
}

func (s *currencyService) handleHeartbeat() {
	defer s.wg.Done()
	ticker := time.NewTimer(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.stream == nil {
				s.log.Error("Bi-directional stream is nil, attempting to reinitialize")
				if err := s.initializeStream(context.Background()); err != nil {
					continue
				}
			}
			heartbeat := &protos.RateRequest{
				Base:        protos.Currencies_UNKNOWN,
				Destination: protos.Currencies_UNKNOWN,
			}

			if err := s.stream.Send(heartbeat); err != nil {
				s.log.Error("Failed to send heartbeat", "error", err)
				// Attempt to reinitialize the stream
				_ = s.initializeStream(context.Background())
			} else {
				s.log.Debug("Heartbeat send successfully")
			}
		case <-s.closeCh:
			s.log.Info("handleHeartbeat received shutdown signal")
			return
		}
	}
}

func (s *currencyService) handleRateUpdates() {
	defer s.wg.Done()
	for {
		select {
		case <-s.closeCh:
			s.log.Info("handleRateUpdates received shutdown signal")
			return
		default:
			if s.stream == nil {
				s.log.Error("Bidirectional stream is nil, waiting before retry")
				time.Sleep(5 * time.Second)
				continue
			}

			response, err := s.stream.Recv()
			if err != nil {
				s.log.Error("Error receiving rate updates", "error", err)
				// Attempting to reinitialize the stream
				_ = s.initializeStream(context.Background())
				time.Sleep(5 * time.Second)
				continue
			}

			switch msg := response.Message.(type) {
			case *protos.StreamingRateResponse_RateResponse:
				currency := msg.RateResponse.GetDestination().String()
				newRate := msg.RateResponse.GetRate()

				s.ratesMutex.Lock()
				oldRate, exists := s.rates[currency]
				s.rates[currency] = newRate
				s.ratesMutex.Unlock()

				s.log.Debug(
					"Updated rate",
					"destination", msg.RateResponse.GetDestination().String(),
					"rate", msg.RateResponse.GetRate())

				// Only publish the event if rate actually changed
				if !exists || oldRate != newRate {
					// Publish rate changed event
					s.eventBus.Publish(events.RateChanged{
						Currency: currency,
						NewRate:  newRate,
					})
				}
			case *protos.StreamingRateResponse_Error:
				s.log.Error("Received error from server", "error", msg.Error.GetMessage())
			}
		}
	}
}

func (s *currencyService) GetRate(ctx context.Context, base, destination string) (float64, error) {
	s.log.Debug("Getting exchange rate", "base", base, "destination", destination)

	// check if rate is already available
	s.ratesMutex.RLock()
	rate, ok := s.rates[destination]
	s.ratesMutex.RUnlock()
	if ok {
		return rate, nil
	}

	// Request new rate via gRPC call
	rateRequest := &protos.RateRequest{
		Base:        protos.Currencies(protos.Currencies_value[base]),
		Destination: protos.Currencies(protos.Currencies_value[destination]),
	}

	resp, err := s.client.GetRate(ctx, rateRequest)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		s.log.Error(
			"Error getting exchange rate",
			"base", base,
			"destination", destination,
			"error", grpcErr.Message())
		return 0, err
	}

	// Store the rate
	s.ratesMutex.Lock()
	s.rates[destination] = resp.Rate
	s.ratesMutex.Unlock()

	// Subscribe to rates for this pair
	if err := s.SubscribeToRates(ctx, []string{destination}); err != nil {
		s.log.Error("Failed to subscribe to rate updates", "error", err)
	}
	return resp.Rate, nil
}

func (s *currencyService) SubscribeToRates(ctx context.Context, currencies []string) error {
	s.log.Debug("Subscribing to currency rate updates", "currencies", currencies)

	if s.stream == nil {
		if err := s.initializeStream(ctx); err != nil {
			return err
		}
	}

	s.subMutex.Lock()
	for _, currency := range currencies {
		// Check if already subscribed
		if _, exists := s.subscriptions[currency]; exists {
			continue
		}
		s.subscriptions[currency] = struct{}{}

		rateRequest := &protos.RateRequest{
			Base:        protos.Currencies(protos.Currencies_value["EUR"]),
			Destination: protos.Currencies(protos.Currencies_value[currency]),
		}

		if err := s.stream.Send(rateRequest); err != nil {
			s.log.Error("Error sending rate subscription request", "currency", currency, "error", err)
			s.subMutex.Unlock()
			return err
		}
	}
	s.subMutex.Unlock()

	return nil
}

func (s *currencyService) ListAvailableCurrencies(ctx context.Context) ([]string, error) {
	s.log.Debug("Listing available currencies")

	resp, err := s.client.ListCurrencies(ctx, &protos.Empty{})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		s.log.Error("Error listing currencies", "error", grpcErr.Message())
		return nil, err
	}

	return resp.Currencies, nil
}

// Close gracefully shuts down the CurrencyService
func (s *currencyService) Close() error {
	var err error
	s.once.Do(func() {
		s.log.Info("Shutting down CurrencyService...")
		close(s.closeCh) // Signal goroutines to stop

		// Close the gRPC stream
		if s.stream != nil {
			err = s.stream.CloseSend()
		}

		// Wait for all goroutines to finish
		s.wg.Wait()

		s.log.Info("CurrencyService shutdown complete.")
	})
	return err
}
