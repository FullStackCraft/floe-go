package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHindsightBaseURL = "https://hindsightapi.com/api"
	defaultDealerBaseURL    = "https://vannacharm.com/api"
	defaultAMTBaseURL       = "https://amtjoy.com/api"
	dateLayout              = "2006-01-02"
	defaultHTTPTimeout      = 30 * time.Second
)

// ApiClient is a unified API client for Hindsight, dealer minute-surface, and AMT data.
type ApiClient struct {
	apiKey           string
	httpClient       *http.Client
	hindsightBaseURL string
	dealerBaseURL    string
	amtBaseURL       string
}

// NewApiClient returns a configured API client.
func NewApiClient(apiKey string, httpClient *http.Client) *ApiClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &ApiClient{
		apiKey:           strings.TrimSpace(apiKey),
		httpClient:       httpClient,
		hindsightBaseURL: defaultHindsightBaseURL,
		dealerBaseURL:    defaultDealerBaseURL,
		amtBaseURL:       defaultAMTBaseURL,
	}
}

// GetHindsightData retrieves economic event data by filter.
func (c *ApiClient) GetHindsightData(ctx context.Context, req HindsightDataRequest) ([]HindsightEvent, error) {
	if err := validateHindsightDataRequest(req); err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("start_date", strings.TrimSpace(req.StartDate))
	query.Set("end_date", strings.TrimSpace(req.EndDate))

	if country := strings.TrimSpace(req.Country); country != "" {
		query.Set("country", country)
	}
	if req.MinVolatility > 0 {
		query.Set("min_volatility", fmt.Sprintf("%d", req.MinVolatility))
	}
	if event := strings.TrimSpace(req.Event); event != "" {
		query.Set("event", event)
	}

	body, err := c.getRaw(ctx, c.hindsightBaseURL, "/getData", query, true)
	if err != nil {
		return nil, err
	}

	return decodeHindsightEvents(body)
}

// GetHindsightSample retrieves sample economic event data.
func (c *ApiClient) GetHindsightSample(ctx context.Context) ([]HindsightEvent, error) {
	body, err := c.getRaw(ctx, c.hindsightBaseURL, "/getSample", nil, true)
	if err != nil {
		return nil, err
	}

	return decodeHindsightEvents(body)
}

// GetDealerMinuteSurfaces retrieves historical minute-surface rows.
func (c *ApiClient) GetDealerMinuteSurfaces(ctx context.Context, req DealerMinuteSurfacesRequest) ([]DealerMinuteSurface, error) {
	if err := validateDealerMinuteSurfacesRequest(req); err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("symbol", strings.TrimSpace(req.Symbol))
	query.Set("trade_date", strings.TrimSpace(req.TradeDate))

	body, err := c.getRaw(ctx, c.dealerBaseURL, "/getMinuteSurfaces", query, true)
	if err != nil {
		return nil, err
	}

	return decodeDealerMinuteSurfaces(body)
}

// GetAMTSessionStats retrieves one AMT session-stats row.
func (c *ApiClient) GetAMTSessionStats(ctx context.Context, req AMTRequest) ([]AMTSessionStatsRow, error) {
	if err := validateAMTRequest(req); err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(strings.TrimSpace(req.Symbol)))
	query.Set("session_id", strings.TrimSpace(req.SessionID))

	body, err := c.getRaw(ctx, c.amtBaseURL, "/getSessionStats", query, true)
	if err != nil {
		return nil, err
	}

	return decodeAMTSessionStats(body)
}

// GetAMTEvents retrieves one AMT minute-events row.
func (c *ApiClient) GetAMTEvents(ctx context.Context, req AMTRequest) ([]AMTEventsRow, error) {
	if err := validateAMTRequest(req); err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(strings.TrimSpace(req.Symbol)))
	query.Set("session_id", strings.TrimSpace(req.SessionID))

	body, err := c.getRaw(ctx, c.amtBaseURL, "/getAMTEvents", query, true)
	if err != nil {
		return nil, err
	}

	return decodeAMTEvents(body)
}

func (c *ApiClient) getRaw(
	ctx context.Context,
	baseURL string,
	path string,
	query url.Values,
	requiresAPIKey bool,
) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if requiresAPIKey && strings.TrimSpace(c.apiKey) == "" {
		return nil, errors.New("api key is required")
	}

	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("base URL is required")
	}

	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if requiresAPIKey {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAPIError(resp.StatusCode, body)
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, errors.New("empty response body")
	}

	return body, nil
}

func validateHindsightDataRequest(req HindsightDataRequest) error {
	startDate := strings.TrimSpace(req.StartDate)
	endDate := strings.TrimSpace(req.EndDate)

	if startDate == "" {
		return errors.New("start_date is required")
	}
	if endDate == "" {
		return errors.New("end_date is required")
	}

	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return fmt.Errorf("start_date must be in YYYY-MM-DD format")
	}

	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return fmt.Errorf("end_date must be in YYYY-MM-DD format")
	}

	if end.Before(start) {
		return errors.New("end_date must be on or after start_date")
	}

	if req.MinVolatility != 0 && (req.MinVolatility < 1 || req.MinVolatility > 3) {
		return errors.New("min_volatility must be between 1 and 3 when provided")
	}

	return nil
}

func validateDealerMinuteSurfacesRequest(req DealerMinuteSurfacesRequest) error {
	if strings.TrimSpace(req.Symbol) == "" {
		return errors.New("symbol is required")
	}
	if strings.TrimSpace(req.TradeDate) == "" {
		return errors.New("trade_date is required")
	}

	if _, err := time.Parse(dateLayout, strings.TrimSpace(req.TradeDate)); err != nil {
		return fmt.Errorf("trade_date must be in YYYY-MM-DD format")
	}

	return nil
}

func validateAMTRequest(req AMTRequest) error {
	if strings.TrimSpace(req.Symbol) == "" {
		return errors.New("symbol is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return errors.New("session_id is required")
	}

	if _, err := time.Parse(dateLayout, strings.TrimSpace(req.SessionID)); err != nil {
		return fmt.Errorf("session_id must be in YYYY-MM-DD format")
	}

	return nil
}

func decodeAPIError(statusCode int, body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return &APIError{StatusCode: statusCode, Message: http.StatusText(statusCode)}
	}

	var payload struct {
		Success              *bool  `json:"success"`
		Error                string `json:"error"`
		Message              string `json:"message"`
		SubscriptionEnd      string `json:"subscriptionEnd"`
		SubscriptionEndSnake string `json:"subscription_end"`
	}

	message := ""
	subscriptionEnd := ""
	if err := json.Unmarshal(body, &payload); err == nil {
		message = firstNonEmpty(payload.Error, payload.Message)
		subscriptionEnd = firstNonEmpty(payload.SubscriptionEnd, payload.SubscriptionEndSnake)
	}

	if message == "" {
		message = truncateForError(trimmed)
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	return &APIError{
		StatusCode:      statusCode,
		Message:         message,
		SubscriptionEnd: subscriptionEnd,
		RawBody:         string(body),
	}
}

func decodeHindsightEvents(body []byte) ([]HindsightEvent, error) {
	var envelope struct {
		Success              bool             `json:"success"`
		Data                 []HindsightEvent `json:"data"`
		Error                string           `json:"error"`
		Message              string           `json:"message"`
		SubscriptionEnd      string           `json:"subscriptionEnd"`
		SubscriptionEndSnake string           `json:"subscription_end"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil {
		isEnvelope := envelope.Success || envelope.Data != nil || envelope.Error != "" || envelope.Message != "" || envelope.SubscriptionEnd != "" || envelope.SubscriptionEndSnake != ""
		if isEnvelope {
			if !envelope.Success {
				return nil, &APIError{
					StatusCode:      http.StatusOK,
					Message:         firstNonEmpty(envelope.Error, envelope.Message, "request failed"),
					SubscriptionEnd: firstNonEmpty(envelope.SubscriptionEnd, envelope.SubscriptionEndSnake),
					RawBody:         string(body),
				}
			}
			return envelope.Data, nil
		}
	}

	var events []HindsightEvent
	if err := json.Unmarshal(body, &events); err == nil {
		return events, nil
	}

	return nil, errors.New("failed to decode hindsight response")
}

func decodeDealerMinuteSurfaces(body []byte) ([]DealerMinuteSurface, error) {
	var envelope struct {
		Success              bool                  `json:"success"`
		Data                 []DealerMinuteSurface `json:"data"`
		Error                string                `json:"error"`
		Message              string                `json:"message"`
		SubscriptionEnd      string                `json:"subscriptionEnd"`
		SubscriptionEndSnake string                `json:"subscription_end"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil {
		isEnvelope := envelope.Success || envelope.Data != nil || envelope.Error != "" || envelope.Message != "" || envelope.SubscriptionEnd != "" || envelope.SubscriptionEndSnake != ""
		if isEnvelope {
			if !envelope.Success {
				return nil, &APIError{
					StatusCode:      http.StatusOK,
					Message:         firstNonEmpty(envelope.Error, envelope.Message, "request failed"),
					SubscriptionEnd: firstNonEmpty(envelope.SubscriptionEnd, envelope.SubscriptionEndSnake),
					RawBody:         string(body),
				}
			}
			return envelope.Data, nil
		}
	}

	var rows []DealerMinuteSurface
	if err := json.Unmarshal(body, &rows); err == nil {
		return rows, nil
	}

	return nil, errors.New("failed to decode dealer minute surfaces response")
}

func decodeAMTSessionStats(body []byte) ([]AMTSessionStatsRow, error) {
	var envelope struct {
		Success              bool                 `json:"success"`
		Data                 []AMTSessionStatsRow `json:"data"`
		Error                string               `json:"error"`
		Message              string               `json:"message"`
		SubscriptionEnd      string               `json:"subscriptionEnd"`
		SubscriptionEndSnake string               `json:"subscription_end"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil {
		isEnvelope := envelope.Success || envelope.Data != nil || envelope.Error != "" || envelope.Message != "" || envelope.SubscriptionEnd != "" || envelope.SubscriptionEndSnake != ""
		if isEnvelope {
			if !envelope.Success {
				return nil, &APIError{
					StatusCode:      http.StatusOK,
					Message:         firstNonEmpty(envelope.Error, envelope.Message, "request failed"),
					SubscriptionEnd: firstNonEmpty(envelope.SubscriptionEnd, envelope.SubscriptionEndSnake),
					RawBody:         string(body),
				}
			}
			return envelope.Data, nil
		}
	}

	var rows []AMTSessionStatsRow
	if err := json.Unmarshal(body, &rows); err == nil {
		return rows, nil
	}

	return nil, errors.New("failed to decode amt session stats response")
}

func decodeAMTEvents(body []byte) ([]AMTEventsRow, error) {
	var envelope struct {
		Success              bool           `json:"success"`
		Data                 []AMTEventsRow `json:"data"`
		Error                string         `json:"error"`
		Message              string         `json:"message"`
		SubscriptionEnd      string         `json:"subscriptionEnd"`
		SubscriptionEndSnake string         `json:"subscription_end"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil {
		isEnvelope := envelope.Success || envelope.Data != nil || envelope.Error != "" || envelope.Message != "" || envelope.SubscriptionEnd != "" || envelope.SubscriptionEndSnake != ""
		if isEnvelope {
			if !envelope.Success {
				return nil, &APIError{
					StatusCode:      http.StatusOK,
					Message:         firstNonEmpty(envelope.Error, envelope.Message, "request failed"),
					SubscriptionEnd: firstNonEmpty(envelope.SubscriptionEnd, envelope.SubscriptionEndSnake),
					RawBody:         string(body),
				}
			}
			return envelope.Data, nil
		}
	}

	var rows []AMTEventsRow
	if err := json.Unmarshal(body, &rows); err == nil {
		return rows, nil
	}

	return nil, errors.New("failed to decode amt events response")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncateForError(value string) string {
	const maxLen = 300
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
