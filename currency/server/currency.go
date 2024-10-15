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
	c := &Currency{log: l, rates: r,
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
					err = clientStream.Send(&protos.RateResponse{
						Base:        rateRequest.Base,
						Destination: rateRequest.Destination,
						Rate:        rate,
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

// SubscribeRates implement the rpc function specified in the .proto file
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
			c.addSubscription(clientStream, rateRequest)
		}
	}
	return nil
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
