package data

import (
	"encoding/xml"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"net/http"
	"strconv"
)

type ExchangeRates struct {
	log   hclog.Logger
	rates map[string]float64
}

func NewRates(logger hclog.Logger) (*ExchangeRates, error) {
	er := &ExchangeRates{logger, map[string]float64{}}

	err := er.getRates()

	return er, err
}

func (e *ExchangeRates) GetRates(base, dest string) (float64, error) {
	br, ok := e.rates[base]
	if !ok {
		return 0, fmt.Errorf("Rate not found for currency %s", base)
	}

	dr, ok := e.rates[dest]
	if !ok {
		return 0, fmt.Errorf("Rate not found for currency %s", dest)
	}

	return dr / br, nil
}

func (e *ExchangeRates) getRates() error {
	resp, err := http.DefaultClient.Get("https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml")
	if err != nil {
		e.log.Error("Failed to fetch exchange rates", "error", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected success code 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	parsedCubes := &Cubes{}
	err = xml.NewDecoder(resp.Body).Decode(&parsedCubes)
	if err != nil {
		e.log.Error("Failed to decode XML response", "error", err)
		return err
	}

	for _, cube := range parsedCubes.CubeData {
		rate, err := strconv.ParseFloat(cube.Rate, 64)
		if err != nil {
			return err
		}

		e.rates[cube.Currency] = rate
	}

	e.rates["EUR"] = 1
	return nil
}

type Cubes struct {
	CubeData []Cube `xml:"Cube>Cube>Cube"`
}

type Cube struct {
	Currency string `xml:"currency,attr"`
	Rate     string `xml:"rate,attr"`
}
