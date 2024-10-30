package server

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/data"
	"github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"io"
	"sync"
	"time"
)

// clientSubscription holds the subscription details for a client
type clientSubscription struct {
	rateRequests []*protos.RateRequest
	lastActivity time.Time
}

// Currency is a gRPC server that implements the methods defined by the CurrencyServer interface
type Currency struct {
	log           hclog.Logger
	rates         *data.ExchangeRates
	subscriptions map[protos.Currency_SubscribeRatesServer]*clientSubscription
	subsMutex     sync.RWMutex
	protos.UnimplementedCurrencyServer
}

// NewCurrency creates a new Currency server
func NewCurrency(l hclog.Logger, r *data.ExchangeRates) *Currency {
	c := &Currency{
		log:           l,
		rates:         r,
		subscriptions: make(map[protos.Currency_SubscribeRatesServer]*clientSubscription)}
	go c.handleUpdates()

	return c
}

// handleUpdates sends updated rates to subscribed clients and removes stale subscriptions
func (c *Currency) handleUpdates() {
	rateUpdates := c.rates.MonitorRates(5 * time.Second)
	cleanupTicker := time.NewTicker(1 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-rateUpdates:
			c.log.Info("Got updated rates")
			subsCopy := c.getSubscriptionsCopy()

			// loop over subscribed clients
			for clientStream, sub := range subsCopy {
				// send updates to client
				for _, rateRequest := range sub.rateRequests {
					rate, err := c.rates.GetRate(rateRequest.GetBase().String(), rateRequest.GetDestination().String())
					if err != nil {
						c.log.Error(
							"Unable to get updated rate",
							"base", rateRequest.GetBase().String(),
							"destination", rateRequest.GetDestination().String(),
							"error", err,
						)
						continue
					}

					// send the updated rate to client
					err = clientStream.Send(&protos.StreamingRateResponse{
						Message: &protos.StreamingRateResponse_RateResponse{
							RateResponse: &protos.RateResponse{
								Base:        rateRequest.Base,
								Destination: rateRequest.Destination,
								Rate:        rate,
							},
						},
					})

					if err != nil {
						c.log.Error(
							"Unable to send updated rate to client, removing subscription",
							"base", rateRequest.GetBase().String(),
							"destination", rateRequest.GetDestination().String(),
							"error", err,
						)
						c.removeSubscription(clientStream)
						break // exit inner loop since client is removed
					}
				}
			}
		case <-cleanupTicker.C:
			c.removeStaleSubscriptions()
		}
	}
}

// removeStaleSubscriptions removes subscriptions that have been inactive for more than 5 minutes
func (c *Currency) removeStaleSubscriptions() {
	c.subsMutex.Lock()
	defer c.subsMutex.Unlock()

	for clientStream, sub := range c.subscriptions {
		if time.Since(sub.lastActivity) > 5*time.Minute {
			c.log.Info("Removing stale client subscription")
			delete(c.subscriptions, clientStream)
		}
	}
}

// addSubscription adds a rate request to a client's subscription and updates last activity time
func (c *Currency) addSubscription(clientStream protos.Currency_SubscribeRatesServer, rateRequest *protos.RateRequest) {
	c.subsMutex.Lock()
	defer c.subsMutex.Unlock()

	sub, exists := c.subscriptions[clientStream]
	if !exists {
		sub = &clientSubscription{
			rateRequests: []*protos.RateRequest{},
		}
		c.subscriptions[clientStream] = sub
	}
	sub.rateRequests = append(sub.rateRequests, rateRequest)
	sub.lastActivity = time.Now()
}

func (c *Currency) removeSubscription(clientStream protos.Currency_SubscribeRatesServer) {
	c.subsMutex.Lock()
	defer c.subsMutex.Unlock()
	delete(c.subscriptions, clientStream)

	// extract peer information
	p, ok := peer.FromContext(clientStream.Context())
	if ok {
		c.log.Info("Removed client subscription", "client", p.Addr.String())
	} else {
		c.log.Info("Removed client subscription", "client", "unknown")
	}
}

// getSubscriptionsCopy returns a copy of the subscriptions map
func (c *Currency) getSubscriptionsCopy() map[protos.Currency_SubscribeRatesServer]*clientSubscription {
	c.subsMutex.RLock()
	defer c.subsMutex.RUnlock()

	subsCopy := make(map[protos.Currency_SubscribeRatesServer]*clientSubscription)
	for clientStream, sub := range c.subscriptions {
		subsCopy[clientStream] = sub
	}
	return subsCopy
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle request response for GetRate", "base", rr.GetBase(), "dest", rr.GetDestination())

	// validate parameters: base currency cannot be the same as destination
	if rr.Base == rr.Destination {
		err := status.Errorf(
			codes.InvalidArgument,
			"Base currency %s cannot be equal to destination currency %s",
			rr.Base.String(),
			rr.Base.String(),
		)

		return nil, err
	}
	// validate Base currency
	if rr.GetBase() == protos.Currencies_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "Base currency is not specified")
	}

	// validate Destination currency
	if rr.GetDestination() == protos.Currencies_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "Destination currency is not specified")
	}

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Exchange rate not found")
	}

	return &protos.RateResponse{
		Base:        rr.GetBase(),
		Destination: rr.GetDestination(),
		Rate:        rate,
	}, nil
}

// SubscribeRates implements the rpc function specified in the .proto file
func (c *Currency) SubscribeRates(clientStream protos.Currency_SubscribeRatesServer) error {
	for {
		rateRequest, err := clientStream.Recv()
		if err == io.EOF {
			c.log.Info("Client has closed the connection")
			c.removeSubscription(clientStream)
			break
		}
		if err != nil {
			c.log.Error("Unable to read from client", "error", err)
			c.removeSubscription(clientStream)
			return status.Errorf(codes.Internal, "Error receiving from client %v", err)
		}

		c.log.Info("Handle client request", "request_base", rateRequest.GetBase(), "request_dest", rateRequest.GetDestination())

		if isHeartbeat(rateRequest) {
			c.updateClientActivity(clientStream)
		} else {
			// validate the RateRequest
			errMsg := c.validateRateRequest(rateRequest)
			if errMsg != "" {
				c.log.Error("Invalid RateRequest", "error", errMsg)

				// create a google.rpc.Status error message
				grpcError := status.New(codes.InvalidArgument, errMsg)
				grpcErrorWithDetails, err := grpcError.WithDetails(rateRequest)
				if err != nil {
					c.log.Error("Failed to add details to error", "error", err)
					// fallback to sending error without details
					grpcErrorWithDetails = grpcError
				}

				// send error response within the stream
				err = clientStream.Send(&protos.StreamingRateResponse{
					Message: &protos.StreamingRateResponse_Error{
						Error: grpcErrorWithDetails.Proto(),
					},
				})

				if err != nil {
					c.log.Error("Failed to send error response", "error", err)
					c.removeSubscription(clientStream)
					return status.Errorf(codes.Internal, "Failed to send error response: %v", err)
				}
				continue // skip adding the subscription
			}

			// check for duplicate subscription
			if c.subscriptionExists(clientStream, rateRequest) {
				errMsg := "Subscription already exists for this currency pair!"
				c.log.Error(errMsg)

				// create a google.rpc.Status error message
				grpcError := status.New(codes.InvalidArgument, errMsg)
				grpcErrorWithDetails, err := grpcError.WithDetails(rateRequest)
				if err != nil {
					c.log.Error("Failed to add details to error", "error", err)
					// fallback to sending error without details
					grpcErrorWithDetails = grpcError
				}

				// send error response within the stream
				err = clientStream.Send(&protos.StreamingRateResponse{
					Message: &protos.StreamingRateResponse_Error{
						Error: grpcErrorWithDetails.Proto(),
					},
				})

				if err != nil {
					c.log.Error("Failed to send error response", "error", err)
					c.removeSubscription(clientStream)
					return status.Errorf(codes.Internal, "Failed to send error response %v", err)
				}
				continue
			}

			c.addSubscription(clientStream, rateRequest)
		}
	}
	return nil
}

func (c *Currency) ListCurrencies(ctx context.Context, req *protos.Empty) (*protos.ListCurrenciesResponse, error) {
	c.log.Info("Handling ListCurrencies request")

	// Utilize the thread-safe method from ExchangeRates
	allRates := c.rates.GetAllRates()

	// Extract currency codes from the rates map
	var currencies []string
	for currency := range allRates {
		currencies = append(currencies, currency)
	}

	return &protos.ListCurrenciesResponse{
		Currencies: currencies,
	}, nil
}

// updateClientActivity updates the last activity timestamp for a client
func (c *Currency) updateClientActivity(clientStream protos.Currency_SubscribeRatesServer) {
	c.subsMutex.Lock()
	defer c.subsMutex.Unlock()

	if sub, exists := c.subscriptions[clientStream]; exists {
		sub.lastActivity = time.Now()
	}
}

// isHeartbeat checks if the RateRequest is a heartbeat message
func isHeartbeat(rr *protos.RateRequest) bool {
	// Assuming that a RateRequest with UNKNOWN currencies is a heartbeat
	return rr.GetBase() == protos.Currencies_UNKNOWN && rr.GetDestination() == protos.Currencies_UNKNOWN
}

func (c *Currency) validateRateRequest(rr *protos.RateRequest) string {
	if rr.GetBase() == protos.Currencies_UNKNOWN {
		return "Base currency is not specified"
	}
	if rr.GetDestination() == protos.Currencies_UNKNOWN {
		return "Destination currency is not specified"
	}
	if rr.GetBase() == rr.GetDestination() {
		return "Base currency cannot be the same as destination currency"
	}
	return ""
}

// subscriptionExists checks if the client has already subscribed to a particular rate request
func (c *Currency) subscriptionExists(
	clientStream protos.Currency_SubscribeRatesServer,
	rateRequest *protos.RateRequest) bool {
	c.subsMutex.RLock()
	defer c.subsMutex.RUnlock()

	if sub, exists := c.subscriptions[clientStream]; exists {
		for _, existingRequest := range sub.rateRequests {
			if existingRequest.GetBase() == rateRequest.GetBase() &&
				existingRequest.GetDestination() == rateRequest.GetDestination() {
				return true
			}
		}
	}

	return false
}
