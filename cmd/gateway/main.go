package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	asyncqueue "github.com/example/car-mall-intelligent-agent/internal/async"
	"github.com/example/car-mall-intelligent-agent/internal/cache"
	"github.com/example/car-mall-intelligent-agent/internal/catalog"
	"github.com/example/car-mall-intelligent-agent/internal/config"
	"github.com/example/car-mall-intelligent-agent/internal/intent"
	"github.com/example/car-mall-intelligent-agent/internal/llm"
	"github.com/example/car-mall-intelligent-agent/internal/metrics"
	"github.com/example/car-mall-intelligent-agent/internal/model"
	"github.com/example/car-mall-intelligent-agent/internal/pricing"
)

func main() {
	_ = config.LoadDotEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	intentSvc := intent.NewService()
	pricingSvc := pricing.NewService()
	llmClient := llm.NewOpenRouterClientFromEnv()
	sessionStore := cache.NewSessionStore()
	metricsStore := metrics.NewStore()
	queue := asyncqueue.NewInMemoryQueue(1024)
	queue.RunConsumer(ctx, pricingSvc, func(task model.PricingTask, result model.PricingResult, _ time.Duration) {
		metricsStore.RecordPricingTaskDone(task.TaskID, result)
	})
	orderStore := newOrderStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/", serveMall)
	mux.HandleFunc("/mall", serveMall)
	mux.HandleFunc("/payment", servePayment)
	mux.HandleFunc("/dashboard", serveDashboard)
	mux.HandleFunc("/api/v1/cars", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, catalog.ListCars())
	})
	mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, metricsStore.Snapshot())
	})
	mux.HandleFunc("/api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			CarID        string  `json:"car_id"`
			CarName      string  `json:"car_name"`
			Price        float64 `json:"price"`
			CustomerName string  `json:"customer_name"`
			Phone        string  `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.CarID == "" || req.CarName == "" || req.CustomerName == "" || req.Phone == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
			return
		}
		order := orderStore.Create(req.CarID, req.CarName, req.Price, req.CustomerName, req.Phone)
		writeJSON(w, http.StatusOK, order)
	})
	mux.HandleFunc("/api/v1/orders/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")
		path = strings.Trim(path, "/")
		if path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order id is required"})
			return
		}
		parts := strings.Split(path, "/")
		orderID := parts[0]
		if len(parts) == 1 && r.Method == http.MethodGet {
			order, ok := orderStore.Get(orderID)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
				return
			}
			writeJSON(w, http.StatusOK, order)
			return
		}
		if len(parts) == 2 && parts[1] == "pay" && r.Method == http.MethodPost {
			order, ok := orderStore.Pay(orderID)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
				return
			}
			writeJSON(w, http.StatusOK, order)
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unsupported order route"})
	})

	mux.HandleFunc("/api/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var req model.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.SessionID == "" || req.Message == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and message are required"})
			return
		}
		if req.UserID == "" {
			req.UserID = "guest"
		}

		intentName := intentSvc.Recognize(req.Message)
		traceID := "trace-" + time.Now().Format("20060102150405.000")
		lastSession, _ := sessionStore.Get(req.SessionID)
		recommendedCars := catalog.RecommendCars(req.Message, 3)
		reply, replySource := buildReply(
			r.Context(),
			llmClient,
			intentName,
			req.Message,
			lastSession.LastIntent,
			lastSession.LastReply,
			recommendedCars,
		)

		if intentName == intent.IntentPricing {
			// Select the first recommended car or fallback to EV Pro
			modelName := "EV Pro"
			basePrice := 269000.0
			if len(recommendedCars) > 0 {
				modelName = recommendedCars[0].Name
				basePrice = recommendedCars[0].BasePrice
			}
			task := model.PricingTask{
				TaskID:       "task-" + time.Now().Format("150405.000"),
				SessionID:    req.SessionID,
				UserID:       req.UserID,
				Model:        modelName,
				BasePrice:    basePrice,
				CustomerTier: "silver",
				CreatedAt:    time.Now(),
			}
			metricsStore.RecordPricingTaskQueued(task)
			queue.Publish(task)
		}

		sessionStore.Set(req.SessionID, cache.SessionContext{
			LastIntent: intentName,
			LastReply:  reply,
			UpdatedAt:  time.Now(),
		})

		resp := model.ChatResponse{
			SessionID:       req.SessionID,
			Intent:          intentName,
			Reply:           replySource + ": " + reply,
			TraceID:         traceID,
			LatencyMS:       time.Since(start).Milliseconds(),
			RecommendedCars: toCarCardBriefs(recommendedCars),
		}
		metricsStore.RecordChat(intentName, resp.LatencyMS)
		writeJSON(w, http.StatusOK, resp)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	log.Println("car mall gateway listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func buildReply(
	ctx context.Context,
	llmClient *llm.Client,
	intentName string,
	message string,
	lastIntent string,
	lastReply string,
	recommendedCars []catalog.Car,
) (string, string) {
	candidateSummary := buildCandidateSummary(recommendedCars)
	if llmClient != nil && llmClient.Enabled() {
		reply, err := llmClient.GenerateSalesReply(ctx, message, intentName, lastIntent, lastReply, candidateSummary)
		if err == nil {
			return reply, "openrouter/" + llmClient.Model()
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("openrouter fallback: %v", err)
		}
	}

	switch intentName {
	case intent.IntentPricing:
		return fallbackReplyWithCars("I've matched several models for detailed comparison and initiated a smart pricing task.", recommendedCars), "fallback"
	case intent.IntentPurchase:
		return fallbackReplyWithCars("I've selected several models suitable for purchase and test drives.", recommendedCars), "fallback"
	case intent.IntentLoan:
		return fallbackReplyWithCars("Based on your budget and financing needs, I'll first recommend several models with easier installment plans.", recommendedCars), "fallback"
	default:
		if strings.Contains(strings.ToLower(message), "hello") {
			return "Hello, I'm the Car Mall intelligent sales assistant.", "fallback"
		}
		return fallbackReplyWithCars("I've first filtered several models from our current inventory based on your needs.", recommendedCars), "fallback"
	}
}

func toCarCardBriefs(cars []catalog.Car) []model.CarCardBrief {
	if len(cars) == 0 {
		return nil
	}
	result := make([]model.CarCardBrief, 0, len(cars))
	for _, car := range cars {
		result = append(result, model.CarCardBrief{
			ID:               car.ID,
			Name:             car.Name,
			Type:             car.Type,
			RangeKM:          car.RangeKM,
			Acceleration0100: car.Acceleration0100,
			BasePrice:        car.BasePrice,
			Image:            car.Image,
		})
	}
	return result
}

func buildCandidateSummary(cars []catalog.Car) string {
	if len(cars) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cars))
	for _, car := range cars {
		parts = append(parts, fmt.Sprintf("%s (%s, %.0f yuan, %d km range)", car.Name, car.Type, car.BasePrice, car.RangeKM))
	}
	return strings.Join(parts, "; ")
}

func fallbackReplyWithCars(prefix string, cars []catalog.Car) string {
	if len(cars) == 0 {
		return prefix + " Please tell me your model preferences, budget, or financing needs."
	}
	names := make([]string, 0, len(cars))
	for _, car := range cars {
		names = append(names, car.Name)
	}
	return prefix + " Recommended to view first: " + strings.Join(names, ", ") + "."
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	serveWebFile(w, r, "dashboard.html")
}

func serveMall(w http.ResponseWriter, r *http.Request) {
	serveWebFile(w, r, "mall.html")
}

func servePayment(w http.ResponseWriter, r *http.Request) {
	serveWebFile(w, r, "payment.html")
}

func serveWebFile(w http.ResponseWriter, r *http.Request, filename string) {
	candidates := []string{
		filepath.Join("web", filename),
		filepath.Join("..", "..", "web", filename),
		filepath.Join("/app", "web", filename),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("web file not found: " + filename))
}

type order struct {
	OrderID      string    `json:"order_id"`
	CarID        string    `json:"car_id"`
	CarName      string    `json:"car_name"`
	Price        float64   `json:"price"`
	CustomerName string    `json:"customer_name"`
	Phone        string    `json:"phone"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type orderStore struct {
	mu     sync.RWMutex
	seq    int64
	orders map[string]order
}

func newOrderStore() *orderStore {
	return &orderStore{orders: make(map[string]order)}
}

func (s *orderStore) Create(carID, carName string, price float64, customerName, phone string) order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := "ord-" + strconv.FormatInt(s.seq, 10)
	o := order{
		OrderID:      id,
		CarID:        carID,
		CarName:      carName,
		Price:        price,
		CustomerName: customerName,
		Phone:        phone,
		Status:       "pending_payment",
		CreatedAt:    time.Now(),
	}
	s.orders[id] = o
	return o
}

func (s *orderStore) Get(id string) (order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *orderStore) Pay(id string) (order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return order{}, false
	}
	o.Status = "paid"
	s.orders[id] = o
	return o, true
}
