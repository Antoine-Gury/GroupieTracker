package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// PayPalAccessToken représente le token d'accès PayPal
type PayPalAccessToken struct {
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	AppID       string `json:"app_id"`
	ExpiresIn   int    `json:"expires_in"`
}

// PayPalOrderRequest représente une demande de création de commande
type PayPalOrderRequest struct {
	Intent string             `json:"intent"`
	PurchaseUnits []PurchaseUnit `json:"purchase_units"`
	ApplicationContext ApplicationContext `json:"application_context"`
}

type PurchaseUnit struct {
	Amount Amount `json:"amount"`
	Description string `json:"description,omitempty"`
}

type Amount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type ApplicationContext struct {
	ReturnURL string `json:"return_url"`
	CancelURL string `json:"cancel_url"`
}

// PayPalOrderResponse représente la réponse de création de commande
type PayPalOrderResponse struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Links  []Link   `json:"links"`
}

type Link struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// PayPalCaptureResponse représente la réponse de capture de paiement
type PayPalCaptureResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// GetPayPalAccessToken obtient un token d'accès PayPal
func GetPayPalAccessToken(client *http.Client) (string, error) {
	if PayPalClientID == "" || PayPalSecret == "" {
		return "", fmt.Errorf("PayPal credentials not configured")
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", PayPalBaseURL+"/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(PayPalClientID, PayPalSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("PayPal token error: %s", string(body))
	}

	var token PayPalAccessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

// CreatePayPalOrder crée une commande PayPal
func CreatePayPalOrder(client *http.Client, amount float64, description, returnURL, cancelURL string) (*PayPalOrderResponse, error) {
	accessToken, err := GetPayPalAccessToken(client)
	if err != nil {
		return nil, err
	}

	orderReq := PayPalOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []PurchaseUnit{
			{
				Amount: Amount{
					CurrencyCode: "EUR",
					Value:        fmt.Sprintf("%.2f", amount),
				},
				Description: description,
			},
		},
		ApplicationContext: ApplicationContext{
			ReturnURL: returnURL,
			CancelURL: cancelURL,
		},
	}

	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", PayPalBaseURL+"/v2/checkout/orders", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		log.Printf("PayPal order creation error: %s", string(body))
		return nil, fmt.Errorf("PayPal order creation failed: %s", string(body))
	}

	var orderResp PayPalOrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, err
	}

	return &orderResp, nil
}

// CapturePayPalOrder capture un paiement PayPal
func CapturePayPalOrder(client *http.Client, orderID string) (*PayPalCaptureResponse, error) {
	accessToken, err := GetPayPalAccessToken(client)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", PayPalBaseURL+"/v2/checkout/orders/"+orderID+"/capture", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Printf("PayPal capture error: %s", string(body))
		return nil, fmt.Errorf("PayPal capture failed: %s", string(body))
	}

	var captureResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}

	if err := json.Unmarshal(body, &captureResp); err != nil {
		return nil, err
	}

	if len(captureResp.PurchaseUnits) > 0 && len(captureResp.PurchaseUnits[0].Payments.Captures) > 0 {
		return &PayPalCaptureResponse{
			ID:     captureResp.PurchaseUnits[0].Payments.Captures[0].ID,
			Status: captureResp.PurchaseUnits[0].Payments.Captures[0].Status,
		}, nil
	}

	return &PayPalCaptureResponse{
		ID:     captureResp.ID,
		Status: captureResp.Status,
	}, nil
}

