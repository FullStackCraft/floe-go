// Package apiclient provides HTTP clients for Hindsight, dealer minute-surface, and AMT datasets.
package apiclient

import (
	"fmt"
	"net/http"
	"time"
)

// HindsightDataRequest filters economic event data.
type HindsightDataRequest struct {
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Country       string `json:"country,omitempty"`
	MinVolatility int    `json:"min_volatility,omitempty"`
	Event         string `json:"event,omitempty"`
}

// DealerMinuteSurfacesRequest filters historical minute surfaces.
type DealerMinuteSurfacesRequest struct {
	Symbol    string `json:"symbol"`
	TradeDate string `json:"trade_date"`
}

// AMTRequest identifies one AMT dataset row.
type AMTRequest struct {
	Symbol    string `json:"symbol"`
	SessionID string `json:"session_id"`
}

// HindsightEvent is one economic event row.
type HindsightEvent struct {
	ID          int        `json:"id"`
	EventID     string     `json:"event_id"`
	Date        string     `json:"date"`
	Time        string     `json:"time"`
	Timezone    string     `json:"timezone"`
	Country     string     `json:"country"`
	CountryCode string     `json:"country_code"`
	EventName   string     `json:"event_name"`
	Volatility  int        `json:"volatility"`
	Actual      *string    `json:"actual"`
	Forecast    *string    `json:"forecast"`
	Previous    *string    `json:"previous"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// SurfacePoint is one point in a minute-level surface.
type SurfacePoint struct {
	Strike float64 `json:"strike,omitempty"`
	Value  float64 `json:"value,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
}

// MinuteSurface stores all required minute-level surfaces.
type MinuteSurface struct {
	Gamma []SurfacePoint `json:"gamma"`
	Vanna []SurfacePoint `json:"vanna"`
	Charm []SurfacePoint `json:"charm"`
	IV    []SurfacePoint `json:"iv"`
}

// DealerMinuteSurface is one historical minute-surface row.
type DealerMinuteSurface struct {
	ID            string         `json:"id"`
	RunAt         *time.Time     `json:"run_at,omitempty"`
	Symbol        string         `json:"symbol"`
	TradeDate     string         `json:"trade_date"`
	MinuteTS      *time.Time     `json:"minute_ts,omitempty"`
	SessionMinute int            `json:"session_minute"`
	Spot          float64        `json:"spot"`
	VIX           float64        `json:"vix"`
	Surfaces      MinuteSurface  `json:"surfaces"`
	Metadata      map[string]any `json:"metadata"`
}

// AMTSessionStatsRow is one AMT session stats row.
type AMTSessionStatsRow struct {
	Symbol      string         `json:"symbol"`
	SessionID   string         `json:"session_id"`
	SessionData map[string]any `json:"session_data"`
}

// AMTEventsRow is one AMT minute-events row.
type AMTEventsRow struct {
	Symbol    string           `json:"symbol"`
	SessionID string           `json:"session_id"`
	Events    []map[string]any `json:"events"`
}

// APIError represents an API response error.
type APIError struct {
	StatusCode      int
	Message         string
	SubscriptionEnd string
	RawBody         string
}

// Error returns a human-readable error string.
func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}

	statusText := ""
	if e.StatusCode > 0 {
		statusText = fmt.Sprintf("%d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}

	switch {
	case statusText != "" && e.Message != "":
		return fmt.Sprintf("api error: %s: %s", statusText, e.Message)
	case e.Message != "":
		return fmt.Sprintf("api error: %s", e.Message)
	case statusText != "":
		return fmt.Sprintf("api error: %s", statusText)
	default:
		return "api error"
	}
}
