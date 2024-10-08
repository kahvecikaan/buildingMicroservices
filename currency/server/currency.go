package server

import (
	"context"
	"github.com/hashicorp/go-hclog"
	protos "github.com/kahvecikaan/buildingMicroservices/currency/protos/currency"
)

// Currency is a gRPC server that implements the methods defined by the CurrencyServer interface
type Currency struct {
	log hclog.Logger
	protos.UnimplementedCurrencyServer
}

// NewCurrency creates a new Currency server
func NewCurrency(l hclog.Logger) *Currency {
	return &Currency{log: l}
}

func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle request response for GetRate", "base", rr.GetBase(), "dest", rr.GetDestination())
	return &protos.RateResponse{Rate: 0.5}, nil
}
