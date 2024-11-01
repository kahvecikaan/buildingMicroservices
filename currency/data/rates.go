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
	log     hclog.Logger
	rates   map[string]float64
	mutex   sync.RWMutex
	closeCh chan struct{}  // Channel to signal shutdown
	wg      sync.WaitGroup // WaitGroup to manage goroutines
}

func NewRates(logger hclog.Logger) (*ExchangeRates, error) {
	er := &ExchangeRates{
		log:     logger,
		rates:   map[string]float64{},
		closeCh: make(chan struct{}),
	}

	err := er.getRates()
	if err != nil {
		return nil, err
	}

	return er, nil
}

func (e *ExchangeRates) GetRate(base, dest string) (float64, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

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

// MonitorRates periodically simulates rate changes and notifies via the returned channel
func (e *ExchangeRates) MonitorRates(interval time.Duration) chan struct{} {
	ret := make(chan struct{})

	e.wg.Add(1) // Increment WaitGroup counter

	go func() {
		defer e.wg.Done() // Decrement WaitGroup counter when goroutine completes
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.mutex.Lock()
				for k, v := range e.rates {
					if k == "EUR" {
						// Skip modifying EUR's rate
						continue
					}
					// Simulate rate fluctuation
					change := rand.Float64() / 10 // Up to 10%
					direction := rand.Intn(2)     // 0 or 1

					if direction == 0 {
						// Decrease rate by up to 10%
						change = 1 - change
					} else {
						// Increase rate by up to 10%
						change = 1 + change
					}

					// Modify the rate
					e.rates[k] = v * change
				}
				e.mutex.Unlock()

				// Notify updates
				ret <- struct{}{}
			case <-e.closeCh:
				e.log.Info("MonitorRates received shutdown signal")
				return
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

	e.rates["EUR"] = 1.0 // Ensure EUR is always present with rate 1.0
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

// Close gracefully shuts down the ExchangeRates service
func (e *ExchangeRates) Close() {
	close(e.closeCh) // Signal goroutines to stop
	e.wg.Wait()      // Wait for all goroutines to finish
	e.log.Info("ExchangeRates service closed gracefully")
}
