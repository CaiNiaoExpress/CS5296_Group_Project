package model

import "time"

type ChatRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	SessionID       string         `json:"session_id"`
	Intent          string         `json:"intent"`
	Reply           string         `json:"reply"`
	TraceID         string         `json:"trace_id"`
	LatencyMS       int64          `json:"latency_ms"`
	RecommendedCars []CarCardBrief `json:"recommended_cars,omitempty"`
}

type CarCardBrief struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	RangeKM          int     `json:"range_km"`
	Acceleration0100 float64 `json:"acceleration_0_100"`
	BasePrice        float64 `json:"base_price"`
	Image            string  `json:"image"`
}

type PricingTask struct {
	TaskID       string    `json:"task_id"`
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Model        string    `json:"model"`
	BasePrice    float64   `json:"base_price"`
	CustomerTier string    `json:"customer_tier"`
	CreatedAt    time.Time `json:"created_at"`
}

type PricingResult struct {
	TaskID         string  `json:"task_id"`
	FinalPrice     float64 `json:"final_price"`
	Discount       float64 `json:"discount"`
	EstimatedInSec int     `json:"estimated_in_sec"`
}
