package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PaymentService handles Culqi payment gateway operations.
type PaymentService struct {
	PrivateKey string
}

// CulqiChargeRequest represents the payload sent to Culqi to create a charge.
type CulqiChargeRequest struct {
	Amount       int    `json:"amount"`        // amount in cents (8500 = 85.00 PEN)
	CurrencyCode string `json:"currency_code"` // "PEN"
	SourceID     string `json:"source_id"`     // token from Culqi Checkout
	Email        string `json:"email"`
	Description  string `json:"description"`
}

// CulqiChargeResponse represents Culqi's response after creating a charge.
type CulqiChargeResponse struct {
	ID           string `json:"id"`
	Amount       int    `json:"amount"`
	CurrencyCode string `json:"currency_code"`
	Email        string `json:"email"`
	Source       struct {
		ID string `json:"id"`
	} `json:"source"`
	Outcome struct {
		Type string `json:"type"` // "venta_exitosa"
	} `json:"outcome"`
	ReferenceCode string `json:"reference_code"`
}

// CulqiErrorResponse represents an error returned by the Culqi API.
type CulqiErrorResponse struct {
	Object      string `json:"object"`
	Type        string `json:"type"`
	MerchantMsg string `json:"merchant_message"`
	UserMsg     string `json:"user_message"`
}

// CreateCharge creates a charge via the Culqi API.
func (s *PaymentService) CreateCharge(ctx context.Context, req *CulqiChargeRequest) (*CulqiChargeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal charge request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.culqi.com/v2/charges", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.PrivateKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("culqi request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read culqi response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var culqiErr CulqiErrorResponse
		if json.Unmarshal(respBody, &culqiErr) == nil && culqiErr.UserMsg != "" {
			return nil, fmt.Errorf("culqi: %s", culqiErr.UserMsg)
		}
		return nil, fmt.Errorf("culqi error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var chargeResp CulqiChargeResponse
	if err := json.Unmarshal(respBody, &chargeResp); err != nil {
		return nil, fmt.Errorf("parse culqi response: %w", err)
	}

	return &chargeResp, nil
}
