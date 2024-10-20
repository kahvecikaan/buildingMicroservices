package data

import (
	"encoding/xml"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ExchangeRates struct {
	log   hclog.Logger
	rates map[string]float64
	mutex sync.RWMutex
}

func NewRates(logger hclog.Logger) (*ExchangeRates, error) {
	er := &ExchangeRates{
		log:   logger,
		rates: map[string]float64{}}

	err := er.getRates()
	if err != nil {
		return nil, err
	}

	return er, nil
}

func (e *ExchangeRates) GetRate(base, dest string) (float64, error) {
	br, ok := e.rates[base]
	if !ok {
		return 0, fmt.Errorf("rate not found for currency %s", base)
	}

	dr, ok := e.rates[dest]
	if !ok {
		return 0, fmt.Errorf("rate not found for currency %s", dest)
	}

	return dr / br, nil
}

// MonitorRates checks the rates in the ECB API every interval and sends a message to the
// returned channel when there are changes
//
// Note: the ECB API only returns data once a day, this function only simulates the changes
// in rates for demonstration purposes
func (e *ExchangeRates) MonitorRates(interval time.Duration) chan struct{} {
	ret := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				// just add a random difference to the rate and return it
				// this simulates the fluctuations in currency rates
				e.mutex.Lock()
				for k, v := range e.rates {
					// change can be 10% of original value
					change := rand.Float64() / 10
					// is this a pos. or neg. change
					direction := rand.Intn(1)

					if direction == 0 {
						// new value will be min 90% of old
						change = 1 - change
					} else {
						// new value will be 110% of old
						change = 1 + change
					}

					// modify the rate
					e.rates[k] = v * change
				}
				e.mutex.Unlock()

				// notify updates, this will block unless there is a listener on the other end
				ret <- struct{}{}
			}
		}
	}()

	return ret
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

	e.mutex.Lock()
	defer e.mutex.Unlock()
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

// GetAllRates returns a copy of all exchange rates in a thread-safe manner
func (e *ExchangeRates) GetAllRates() map[string]float64 {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	ratesCopy := make(map[string]float64)
	for currencyCode, rate := range e.rates {
		ratesCopy[currencyCode] = rate
	}
	return ratesCopy
}

type Cubes struct {
	CubeData []Cube `xml:"Cube>Cube>Cube"`
}

type Cube struct {
	Currency string `xml:"currency,attr"`
	Rate     string `xml:"rate,attr"`
}
