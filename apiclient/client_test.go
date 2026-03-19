package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewApiClientDefaults(t *testing.T) {
	client := NewApiClient("test-key", nil)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.apiKey != "test-key" {
		t.Fatalf("expected api key to be set, got %q", client.apiKey)
	}
	if client.httpClient == nil {
		t.Fatal("expected non-nil http client")
	}
	if client.hindsightBaseURL != defaultHindsightBaseURL {
		t.Fatalf("unexpected hindsight base URL: %s", client.hindsightBaseURL)
	}
	if client.dealerBaseURL != defaultDealerBaseURL {
		t.Fatalf("unexpected dealer base URL: %s", client.dealerBaseURL)
	}
	if client.amtBaseURL != defaultAMTBaseURL {
		t.Fatalf("unexpected amt base URL: %s", client.amtBaseURL)
	}
}

func TestGetHindsightData_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getData" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_date"); got != "2025-01-01" {
			t.Fatalf("expected start_date query param, got %q", got)
		}
		if got := r.URL.Query().Get("end_date"); got != "2025-01-02" {
			t.Fatalf("expected end_date query param, got %q", got)
		}
		if got := r.URL.Query().Get("country"); got != "US" {
			t.Fatalf("expected country query param, got %q", got)
		}
		if got := r.URL.Query().Get("min_volatility"); got != "2" {
			t.Fatalf("expected min_volatility query param, got %q", got)
		}
		if got := r.URL.Query().Get("event"); got != "FOMC" {
			t.Fatalf("expected event query param, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "hindsight-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [{
				"id": 1,
				"event_id": "evt_1",
				"date": "2025-01-01",
				"time": "08:30",
				"timezone": "America/New_York",
				"country": "US",
				"country_code": "US",
				"event_name": "FOMC Meeting Minutes",
				"volatility": 2,
				"actual": "2.1%",
				"forecast": "2.0%",
				"previous": "1.9%"
			}]
		}`))
	}))
	defer server.Close()

	client := NewApiClient("hindsight-key", nil)
	client.hindsightBaseURL = server.URL

	rows, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
		StartDate:     "2025-01-01",
		EndDate:       "2025-01-02",
		Country:       "US",
		MinVolatility: 2,
		Event:         "FOMC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].EventName != "FOMC Meeting Minutes" {
		t.Fatalf("unexpected event_name: %s", rows[0].EventName)
	}
}

func TestGetHindsightData_RawArrayFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 2,
				"event_id": "evt_2",
				"date": "2025-01-02",
				"time": "09:00",
				"timezone": "America/New_York",
				"country": "US",
				"country_code": "US",
				"event_name": "CPI",
				"volatility": 3,
				"actual": "3.1%",
				"forecast": "3.0%",
				"previous": "2.9%"
			}
		]`))
	}))
	defer server.Close()

	client := NewApiClient("hindsight-key", nil)
	client.hindsightBaseURL = server.URL

	rows, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
		StartDate: "2025-01-01",
		EndDate:   "2025-01-03",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].EventID != "evt_2" {
		t.Fatalf("unexpected event_id: %s", rows[0].EventID)
	}
}

func TestGetHindsightSample_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getSample" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "hindsight-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [{
				"id": 3,
				"event_id": "sample_1",
				"date": "2023-08-01",
				"time": "08:30",
				"timezone": "America/New_York",
				"country": "US",
				"country_code": "US",
				"event_name": "Sample Event",
				"volatility": 2,
				"actual": null,
				"forecast": null,
				"previous": null
			}]
		}`))
	}))
	defer server.Close()

	client := NewApiClient("hindsight-key", nil)
	client.hindsightBaseURL = server.URL

	rows, err := client.GetHindsightSample(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].EventID != "sample_1" {
		t.Fatalf("unexpected event_id: %s", rows[0].EventID)
	}
}

func TestGetDealerMinuteSurfaces_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getMinuteSurfaces" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "SPY" {
			t.Fatalf("expected symbol query param, got %q", got)
		}
		if got := r.URL.Query().Get("trade_date"); got != "2026-03-10" {
			t.Fatalf("expected trade_date query param, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "dealer-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [
				{
					"id": "a04fe5f8-27d4-4f8d-b433-22a9c9d4b30d",
					"run_at": "2026-03-10T14:31:00Z",
					"symbol": "SPY",
					"trade_date": "2026-03-10",
					"minute_ts": "2026-03-10T14:30:00Z",
					"session_minute": 61,
					"spot": 512.25,
					"vix": 19.3,
					"surfaces": {
						"gamma": [{"strike": 510, "value": 1200000}],
						"vanna": [{"strike": 510, "value": -80000}],
						"charm": [{"strike": 510, "value": 35000}],
						"iv": [{"strike": 510, "value": 0.24}]
					},
					"metadata": {
						"source": "calc-engine",
						"version": "v1"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewApiClient("dealer-key", nil)
	client.dealerBaseURL = server.URL

	rows, err := client.GetDealerMinuteSurfaces(context.Background(), DealerMinuteSurfacesRequest{
		Symbol:    "SPY",
		TradeDate: "2026-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Symbol != "SPY" {
		t.Fatalf("unexpected symbol: %s", rows[0].Symbol)
	}
	if rows[0].SessionMinute != 61 {
		t.Fatalf("unexpected session minute: %d", rows[0].SessionMinute)
	}
	if got := rows[0].Metadata["source"]; got != "calc-engine" {
		t.Fatalf("unexpected metadata source: %#v", got)
	}
	if len(rows[0].Surfaces.Gamma) != 1 {
		t.Fatalf("expected gamma surface point, got %d", len(rows[0].Surfaces.Gamma))
	}
}

func TestGetAMTSessionStats_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getSessionStats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "MNQ" {
			t.Fatalf("expected symbol query param, got %q", got)
		}
		if got := r.URL.Query().Get("session_id"); got != "2026-03-10" {
			t.Fatalf("expected session_id query param, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "amt-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [
				{
					"symbol": "MNQ",
					"session_id": "2026-03-10",
					"session_data": {
						"sessionType": "Trend Up",
						"tpos": 245
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewApiClient("amt-key", nil)
	client.amtBaseURL = server.URL

	rows, err := client.GetAMTSessionStats(context.Background(), AMTRequest{
		Symbol:    "mnq",
		SessionID: "2026-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Symbol != "MNQ" {
		t.Fatalf("unexpected symbol: %s", rows[0].Symbol)
	}
	if got := rows[0].SessionData["sessionType"]; got != "Trend Up" {
		t.Fatalf("unexpected session data field: %#v", got)
	}
}

func TestGetAMTSessionStats_RawArrayFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"symbol": "ES",
				"session_id": "2026-03-11",
				"session_data": {
					"sessionType": "Balanced"
				}
			}
		]`))
	}))
	defer server.Close()

	client := NewApiClient("amt-key", nil)
	client.amtBaseURL = server.URL

	rows, err := client.GetAMTSessionStats(context.Background(), AMTRequest{
		Symbol:    "ES",
		SessionID: "2026-03-11",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].SessionID != "2026-03-11" {
		t.Fatalf("unexpected session id: %s", rows[0].SessionID)
	}
}

func TestGetAMTEvents_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getAMTEvents" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "NQ" {
			t.Fatalf("expected symbol query param, got %q", got)
		}
		if got := r.URL.Query().Get("session_id"); got != "2026-03-10" {
			t.Fatalf("expected session_id query param, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "amt-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [
				{
					"symbol": "NQ",
					"session_id": "2026-03-10",
					"events": [
						{
							"timestamp": 1710077400000,
							"event_messages": ["Poor high"]
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewApiClient("amt-key", nil)
	client.amtBaseURL = server.URL

	rows, err := client.GetAMTEvents(context.Background(), AMTRequest{
		Symbol:    "NQ",
		SessionID: "2026-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Symbol != "NQ" {
		t.Fatalf("unexpected symbol: %s", rows[0].Symbol)
	}
	if len(rows[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rows[0].Events))
	}
}

func TestGetAMTEvents_EnvelopeError200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error":"not allowed"}`))
	}))
	defer server.Close()

	client := NewApiClient("amt-key", nil)
	client.amtBaseURL = server.URL

	_, err := client.GetAMTEvents(context.Background(), AMTRequest{
		Symbol:    "NQ",
		SessionID: "2026-03-10",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, apiErr.StatusCode)
	}
	if apiErr.Message != "not allowed" {
		t.Fatalf("unexpected message: %s", apiErr.Message)
	}
}

func TestGetHindsightData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"error":"Invalid API key"}`))
	}))
	defer server.Close()

	client := NewApiClient("bad-key", nil)
	client.hindsightBaseURL = server.URL

	_, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
		StartDate: "2025-01-01",
		EndDate:   "2025-01-02",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, apiErr.StatusCode)
	}
	if apiErr.Message != "Invalid API key" {
		t.Fatalf("unexpected message: %s", apiErr.Message)
	}
}

func TestGetHindsightData_SubscriptionLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{
			"success": false,
			"error": "Requested end_date exceeds your subscription limit",
			"subscriptionEnd": "2022-03-11"
		}`))
	}))
	defer server.Close()

	client := NewApiClient("some-key", nil)
	client.hindsightBaseURL = server.URL

	_, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
		StartDate: "2025-01-01",
		EndDate:   "2025-01-02",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.SubscriptionEnd != "2022-03-11" {
		t.Fatalf("unexpected subscription end: %s", apiErr.SubscriptionEnd)
	}
}

func TestGetHindsightData_APIErrorStatusCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedSubstr string
	}{
		{
			name:           "400 bad request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"success":false,"message":"bad request params"}`,
			expectedSubstr: "bad request params",
		},
		{
			name:           "401 unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"success":false,"error":"Unauthorized"}`,
			expectedSubstr: "Unauthorized",
		},
		{
			name:           "500 server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `internal server error`,
			expectedSubstr: "internal server error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			client := NewApiClient("key", nil)
			client.hindsightBaseURL = server.URL

			_, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
				StartDate: "2025-01-01",
				EndDate:   "2025-01-02",
			})
			if err == nil {
				t.Fatal("expected error")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %T", err)
			}
			if apiErr.StatusCode != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, apiErr.StatusCode)
			}
			if !strings.Contains(apiErr.Message, tc.expectedSubstr) {
				t.Fatalf("expected message to contain %q, got %q", tc.expectedSubstr, apiErr.Message)
			}
			if strings.TrimSpace(apiErr.RawBody) == "" {
				t.Fatalf("expected raw body to be populated")
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	t.Run("missing api key for authenticated endpoint", func(t *testing.T) {
		client := NewApiClient("", nil)
		_, err := client.GetDealerMinuteSurfaces(context.Background(), DealerMinuteSurfacesRequest{
			Symbol:    "SPY",
			TradeDate: "2026-03-10",
		})
		if err == nil || !strings.Contains(err.Error(), "api key is required") {
			t.Fatalf("expected api key validation error, got %v", err)
		}
	})

	t.Run("missing api key for hindsight sample", func(t *testing.T) {
		client := NewApiClient("", nil)
		_, err := client.GetHindsightSample(context.Background())
		if err == nil || !strings.Contains(err.Error(), "api key is required") {
			t.Fatalf("expected api key validation error, got %v", err)
		}
	})

	t.Run("invalid hindsight dates", func(t *testing.T) {
		client := NewApiClient("x", nil)
		_, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
			StartDate: "2025-02-30",
			EndDate:   "2025-01-01",
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("invalid hindsight min volatility", func(t *testing.T) {
		client := NewApiClient("x", nil)
		_, err := client.GetHindsightData(context.Background(), HindsightDataRequest{
			StartDate:     "2025-01-01",
			EndDate:       "2025-01-02",
			MinVolatility: 4,
		})
		if err == nil || !strings.Contains(err.Error(), "min_volatility") {
			t.Fatalf("expected min_volatility validation error, got %v", err)
		}
	})

	t.Run("missing dealer symbol", func(t *testing.T) {
		client := NewApiClient("x", nil)
		_, err := client.GetDealerMinuteSurfaces(context.Background(), DealerMinuteSurfacesRequest{
			Symbol:    "",
			TradeDate: "2026-03-10",
		})
		if err == nil || !strings.Contains(err.Error(), "symbol is required") {
			t.Fatalf("expected symbol validation error, got %v", err)
		}
	})

	t.Run("invalid dealer trade date", func(t *testing.T) {
		client := NewApiClient("x", nil)
		_, err := client.GetDealerMinuteSurfaces(context.Background(), DealerMinuteSurfacesRequest{
			Symbol:    "SPY",
			TradeDate: "03-10-2026",
		})
		if err == nil || !strings.Contains(err.Error(), "trade_date") {
			t.Fatalf("expected trade_date validation error, got %v", err)
		}
	})

	t.Run("invalid amt session request", func(t *testing.T) {
		client := NewApiClient("x", nil)
		_, err := client.GetAMTSessionStats(context.Background(), AMTRequest{
			Symbol:    "",
			SessionID: "2026-03-10",
		})
		if err == nil || !strings.Contains(err.Error(), "symbol is required") {
			t.Fatalf("expected symbol validation error, got %v", err)
		}

		_, err = client.GetAMTEvents(context.Background(), AMTRequest{
			Symbol:    "NQ",
			SessionID: "03-10-2026",
		})
		if err == nil || !strings.Contains(err.Error(), "session_id") {
			t.Fatalf("expected session_id validation error, got %v", err)
		}
	})
}

func TestGetWheelScreenerData_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get-options" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("strategy"); got != "CC" {
			t.Fatalf("expected strategy=CC, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "ws-key" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [
				{"ticker": "AAPL", "strategy": "CC", "score": 85.5, "delta": 0.3},
				{"ticker": "MSFT", "strategy": "CC", "score": 82.1, "delta": 0.25}
			],
			"total": 150,
			"page": 1,
			"page_size": 50
		}`))
	}))
	defer server.Close()

	client := NewApiClient("ws-key", nil)
	client.wheelScreenerBaseURL = server.URL

	resp, err := client.GetWheelScreenerData(context.Background(), OptionsScreenerRequest{
		Strategy: "CC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Data))
	}
	if resp.Total != 150 {
		t.Fatalf("expected total 150, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Page)
	}
	if resp.PageSize != 50 {
		t.Fatalf("expected page_size 50, got %d", resp.PageSize)
	}
	if ticker, ok := resp.Data[0]["ticker"].(string); !ok || ticker != "AAPL" {
		t.Fatalf("unexpected first ticker: %v", resp.Data[0]["ticker"])
	}
}

func TestGetLeapsScreenerData_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get-options" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("strategy"); got != "LC" {
			t.Fatalf("expected strategy=LC, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [{"ticker": "NVDA", "strategy": "LC", "score": 90.2}],
			"total": 75,
			"page": 1,
			"page_size": 50
		}`))
	}))
	defer server.Close()

	client := NewApiClient("ls-key", nil)
	client.leapsScreenerBaseURL = server.URL

	resp, err := client.GetLeapsScreenerData(context.Background(), OptionsScreenerRequest{
		Strategy: "LC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp.Data))
	}
	if resp.Total != 75 {
		t.Fatalf("expected total 75, got %d", resp.Total)
	}
}

func TestGetOptionScreenerData_RequestConstructionAndDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getOptionsData" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("strategy"); got != "CDS" {
			t.Fatalf("expected strategy=CDS, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": [
				{"ticker": "TSLA", "strategy": "CDS", "score": 78.3, "max_profit": 250.0}
			],
			"total": 42,
			"page": 2,
			"page_size": 10
		}`))
	}))
	defer server.Close()

	client := NewApiClient("os-key", nil)
	client.optionScreenerBaseURL = server.URL

	resp, err := client.GetOptionScreenerData(context.Background(), OptionsScreenerRequest{
		Strategy: "CDS",
		ExtraParams: map[string]string{
			"page":      "2",
			"page_size": "10",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp.Data))
	}
	if resp.Total != 42 {
		t.Fatalf("expected total 42, got %d", resp.Total)
	}
	if resp.Page != 2 {
		t.Fatalf("expected page 2, got %d", resp.Page)
	}
}

func TestGetWheelScreenerData_ExtraParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("min_score"); got != "70" {
			t.Fatalf("expected min_score=70, got %q", got)
		}
		if got := r.URL.Query().Get("sector"); got != "Technology" {
			t.Fatalf("expected sector=Technology, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[],"total":0,"page":1,"page_size":50}`))
	}))
	defer server.Close()

	client := NewApiClient("ws-key", nil)
	client.wheelScreenerBaseURL = server.URL

	resp, err := client.GetWheelScreenerData(context.Background(), OptionsScreenerRequest{
		Strategy: "CC",
		ExtraParams: map[string]string{
			"min_score": "70",
			"sector":    "Technology",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
}

func TestGetOptionsScreenerData_MissingStrategy(t *testing.T) {
	client := NewApiClient("key", nil)
	_, err := client.GetWheelScreenerData(context.Background(), OptionsScreenerRequest{Strategy: ""})
	if err == nil || !strings.Contains(err.Error(), "strategy is required") {
		t.Fatalf("expected strategy validation error, got %v", err)
	}
}

func TestGetOptionsScreenerData_EnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error":"Invalid or inactive API key"}`))
	}))
	defer server.Close()

	client := NewApiClient("bad-key", nil)
	client.wheelScreenerBaseURL = server.URL

	_, err := client.GetWheelScreenerData(context.Background(), OptionsScreenerRequest{Strategy: "CC"})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Message != "Invalid or inactive API key" {
		t.Fatalf("unexpected message: %s", apiErr.Message)
	}
}

func TestGetOptionsScreenerData_RawArrayFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"ticker":"AAPL","score":85}]`))
	}))
	defer server.Close()

	client := NewApiClient("key", nil)
	client.optionScreenerBaseURL = server.URL

	resp, err := client.GetOptionScreenerData(context.Background(), OptionsScreenerRequest{Strategy: "CC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(resp.Data))
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
}

func TestNewApiClientScreenerDefaults(t *testing.T) {
	client := NewApiClient("test-key", nil)

	if client.wheelScreenerBaseURL != defaultWheelScreenerBaseURL {
		t.Fatalf("unexpected wheel screener base URL: %s", client.wheelScreenerBaseURL)
	}
	if client.leapsScreenerBaseURL != defaultLeapsScreenerBaseURL {
		t.Fatalf("unexpected leaps screener base URL: %s", client.leapsScreenerBaseURL)
	}
	if client.optionScreenerBaseURL != defaultOptionScreenerBaseURL {
		t.Fatalf("unexpected option screener base URL: %s", client.optionScreenerBaseURL)
	}
}

