package fx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	mu         sync.RWMutex
	cachedRate float64
	cachedAt   time.Time
	cacheTTL   = 30 * time.Minute
)

// GetUSDToIRR returns the current USD/IRR exchange rate.
// Falls back to manualRate if live fetch fails or provider is "manual".
func GetUSDToIRR(provider string, manualRate float64) (float64, string, error) {
	if provider == "manual" {
		if manualRate > 0 {
			return manualRate, "manual", nil
		}
		return 0, "", fmt.Errorf("fx: manual rate is zero")
	}

	mu.RLock()
	if cachedRate > 0 && time.Since(cachedAt) < cacheTTL {
		r := cachedRate
		mu.RUnlock()
		return r, "cache", nil
	}
	mu.RUnlock()

	rate, err := fetchLive(provider)
	if err != nil {
		if manualRate > 0 {
			return manualRate, "manual_fallback", nil
		}
		return 0, "", fmt.Errorf("fx: %w", err)
	}

	mu.Lock()
	cachedRate = rate
	cachedAt = time.Now()
	mu.Unlock()
	return rate, provider, nil
}

func fetchLive(provider string) (float64, error) {
	switch provider {
	case "exchangerate-api", "":
		return fetchExchangeRateAPI()
	default:
		return 0, fmt.Errorf("unknown provider: %s", provider)
	}
}

func fetchExchangeRateAPI() (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://open.er-api.com/v6/latest/USD")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var r struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}
	if rate, ok := r.Rates["IRR"]; ok && rate > 0 {
		return rate, nil
	}
	return 0, fmt.Errorf("IRR rate not found in response")
}

// IRRtoUSD converts an amount in IRR to USD using the given rate.
func IRRtoUSD(amountIRR int64, rate float64) float64 {
	if rate == 0 {
		return 0
	}
	return float64(amountIRR) / rate
}

// FormatRate formats a rate as a string for storage.
func FormatRate(r float64) string {
	return strconv.FormatFloat(r, 'f', 2, 64)
}

// InvalidateCache clears the cached exchange rate.
func InvalidateCache() {
	mu.Lock()
	cachedRate = 0
	cachedAt = time.Time{}
	mu.Unlock()
}
