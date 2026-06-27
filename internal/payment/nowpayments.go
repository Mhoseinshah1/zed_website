package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const nowpaymentsAPI = "https://api.nowpayments.io/v1"
const nowpaymentsAPISandbox = "https://api-sandbox.nowpayments.io/v1"

// NOWPayments is the NOWPayments gateway client.
type NOWPayments struct {
	APIKey    string
	IPNSecret string
	Sandbox   bool
	http      *http.Client
}

// NewNOWPayments creates a new NOWPayments client.
func NewNOWPayments(apiKey, ipnSecret string, sandbox bool) *NOWPayments {
	return &NOWPayments{
		APIKey:    apiKey,
		IPNSecret: ipnSecret,
		Sandbox:   sandbox,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// CreatePaymentReq is the request body for creating a payment.
type CreatePaymentReq struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayCurrency      string  `json:"pay_currency"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	IPNCallbackURL   string  `json:"ipn_callback_url"`
	SuccessURL       string  `json:"success_url"`
	CancelURL        string  `json:"cancel_url"`
}

// CreatePaymentResp is the response from creating a payment.
type CreatePaymentResp struct {
	ID          string  `json:"id"`
	Status      string  `json:"payment_status"`
	PayAddress  string  `json:"pay_address"`
	PayAmount   float64 `json:"pay_amount"`
	PayCurrency string  `json:"pay_currency"`
	PaymentURL  string  `json:"invoice_url"`
	ExpiresAt   string  `json:"expiration_estimate_date"`
}

func (n *NOWPayments) baseURL() string {
	if n.Sandbox {
		return nowpaymentsAPISandbox
	}
	return nowpaymentsAPI
}

func (n *NOWPayments) do(method, path string, body interface{}) ([]byte, error) {
	var br io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		br = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, n.baseURL()+path, br)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", n.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := n.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("NOWPayments API error %d: %s", resp.StatusCode, string(b))
	}
	return b, nil
}

// TestStatus verifies the API key is valid.
func (n *NOWPayments) TestStatus() error {
	b, err := n.do("GET", "/status", nil)
	if err != nil {
		return err
	}
	var r struct {
		Message string `json:"message"`
	}
	json.Unmarshal(b, &r) //nolint:errcheck
	if r.Message != "OK" {
		return fmt.Errorf("unexpected status: %s", r.Message)
	}
	return nil
}

// CreatePayment creates a new payment invoice.
func (n *NOWPayments) CreatePayment(req CreatePaymentReq) (*CreatePaymentResp, error) {
	b, err := n.do("POST", "/payment", req)
	if err != nil {
		return nil, err
	}
	var r CreatePaymentResp
	json.Unmarshal(b, &r) //nolint:errcheck
	return &r, nil
}

// VerifyIPNSignature verifies the NOWPayments IPN HMAC-SHA512 signature.
// sortedBody must be the raw JSON body sorted by keys as per NOWPayments docs.
func (n *NOWPayments) VerifyIPNSignature(sortedBody []byte, sig string) bool {
	mac := hmac.New(sha512.New, []byte(n.IPNSecret))
	mac.Write(sortedBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}
