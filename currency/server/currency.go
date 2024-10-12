package server

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/currency/data"
	protos "github.com/kahvecikaan/buildingMicroservices/currency/protos"
	"io"
	"time"
)

// Currency is a gRPC server that implements the methods defined by the CurrencyServer interface
type Currency struct {
	log           hclog.Logger
	rates         *data.ExchangeRates
	subscriptions map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest
	protos.UnimplementedCurrencyServer
}

// NewCurrency creates a new Currency server
func NewCurrency(l hclog.Logger, r *data.ExchangeRates) *Currency {
	c := &Currency{log: l, rates: r,
		subscriptions: make(map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest)}
	go c.handleUpdates()

	return c
}

func (c *Currency) handleUpdates() {
	updatedRates := c.rates.MonitorRates(5 * time.Second)
	for range updatedRates {
		c.log.Info("Got updated rates")

		// loop over subscribed clients
		for k, v := range c.subscriptions {

			// loop over subscribed rates
			for _, rr := range v {
				r, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
				if err != nil {
					c.log.Error("Unable to get update rate", "base", rr.GetBase().String(),
						"destination", rr.GetDestination().String(), "error", err)
				}

				err = k.Send(&protos.RateResponse{
					Base:        rr.Base,
					Destination: rr.Destination,
					Rate:        r})
				if err != nil {
					c.log.Error("Unable to send updated rate", "base", rr.GetBase().String(),
						"destination", rr.GetDestination().String())
				}
			}
		}
	}
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle request response for GetRate", "base", rr.GetBase(), "dest", rr.GetDestination())

	// Validate Base currency
	if rr.GetBase() == protos.Currencies_UNKNOWN {
		return nil, fmt.Errorf("base currency is not specified")
	}

	// Validate Destination currency
	if rr.GetDestination() == protos.Currencies_UNKNOWN {
		return nil, fmt.Errorf("destination currency is not specified")
	}

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &protos.RateResponse{
		Base:        rr.GetBase(),
		Destination: rr.GetDestination(),
		Rate:        rate,
	}, nil
}

func (c *Currency) SubscribeRates(stream protos.Currency_SubscribeRatesServer) error {
	// handle client messages
	for {
		in, err := stream.Recv() // Recv is a blocking method which returns on client data
		// io.EOF signals that the client has closed the connection
		if err == io.EOF {
			c.log.Info("Client has closed the connection")
			break
		}

		// any other error means the transport between the server and client is unavailable
		if err != nil {
			c.log.Error("Unable to read from the client", "error", err)
			return err
		}

		c.log.Info("Handle client request", "request_base", in.GetBase(),
			"request_dest", in.GetDestination())

		rrs, ok := c.subscriptions[stream]
		if !ok {
			rrs = []*protos.RateRequest{}
		}

		rrs = append(rrs, in)
		c.subscriptions[stream] = rrs
	}

	return nil
}
