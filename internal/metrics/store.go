package metrics

import (
	"sort"
	"sync"
	"time"

	"github.com/example/car-mall-intelligent-agent/internal/model"
)

type pricingTaskState struct {
	TaskID     string               `json:"task_id"`
	Status     string               `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	FinishedAt *time.Time           `json:"finished_at,omitempty"`
	Result     *model.PricingResult `json:"result,omitempty"`
}

type Snapshot struct {
	Timestamp              time.Time          `json:"timestamp"`
	TotalRequests          int64              `json:"total_requests"`
	RequestsLastMinute     int64              `json:"requests_last_minute"`
	AvgLatencyMS           float64            `json:"avg_latency_ms"`
	P95LatencyMS           int64              `json:"p95_latency_ms"`
	IntentDistribution     map[string]int64   `json:"intent_distribution"`
	PricingTasksTotal      int64              `json:"pricing_tasks_total"`
	PricingTasksDone       int64              `json:"pricing_tasks_done"`
	PricingTasksInProgress int64              `json:"pricing_tasks_in_progress"`
	RecentPricingTasks     []pricingTaskState `json:"recent_pricing_tasks"`
}

type Store struct {
	mu                  sync.RWMutex
	startedAt           time.Time
	totalRequests       int64
	latenciesMS         []int64
	intentCount         map[string]int64
	requestTimeline     []time.Time
	pricingTasksTotal   int64
	pricingTasksDone    int64
	pricingTaskByID     map[string]pricingTaskState
	pricingTaskOrderIDs []string
}

func NewStore() *Store {
	return &Store{
		startedAt:       time.Now(),
		intentCount:     make(map[string]int64),
		pricingTaskByID: make(map[string]pricingTaskState),
	}
}

func (s *Store) RecordChat(intent string, latencyMS int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalRequests++
	s.intentCount[intent]++
	s.latenciesMS = append(s.latenciesMS, latencyMS)
	s.requestTimeline = append(s.requestTimeline, time.Now())
}

func (s *Store) RecordPricingTaskQueued(task model.PricingTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pricingTasksTotal++
	s.pricingTaskByID[task.TaskID] = pricingTaskState{
		TaskID:    task.TaskID,
		Status:    "queued",
		CreatedAt: task.CreatedAt,
	}
	s.pricingTaskOrderIDs = append(s.pricingTaskOrderIDs, task.TaskID)
}

func (s *Store) RecordPricingTaskDone(taskID string, result model.PricingResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.pricingTaskByID[taskID]
	if !ok {
		now := time.Now()
		s.pricingTaskByID[taskID] = pricingTaskState{
			TaskID:     taskID,
			Status:     "done",
			CreatedAt:  now,
			FinishedAt: &now,
			Result:     &result,
		}
		s.pricingTaskOrderIDs = append(s.pricingTaskOrderIDs, taskID)
		s.pricingTasksDone++
		return
	}
	now := time.Now()
	task.Status = "done"
	task.FinishedAt = &now
	task.Result = &result
	s.pricingTaskByID[taskID] = task
	s.pricingTasksDone++
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	requestsLastMinute := int64(0)
	for _, ts := range s.requestTimeline {
		if now.Sub(ts) <= time.Minute {
			requestsLastMinute++
		}
	}

	avg := 0.0
	if len(s.latenciesMS) > 0 {
		var sum int64
		for _, v := range s.latenciesMS {
			sum += v
		}
		avg = float64(sum) / float64(len(s.latenciesMS))
	}

	p95 := int64(0)
	if len(s.latenciesMS) > 0 {
		cp := make([]int64, len(s.latenciesMS))
		copy(cp, s.latenciesMS)
		sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
		idx := int(float64(len(cp))*0.95) - 1
		if idx < 0 {
			idx = 0
		}
		p95 = cp[idx]
	}

	intentDistribution := make(map[string]int64, len(s.intentCount))
	for k, v := range s.intentCount {
		intentDistribution[k] = v
	}

	recent := make([]pricingTaskState, 0, 10)
	for i := len(s.pricingTaskOrderIDs) - 1; i >= 0 && len(recent) < 10; i-- {
		id := s.pricingTaskOrderIDs[i]
		recent = append(recent, s.pricingTaskByID[id])
	}

	return Snapshot{
		Timestamp:              now,
		TotalRequests:          s.totalRequests,
		RequestsLastMinute:     requestsLastMinute,
		AvgLatencyMS:           avg,
		P95LatencyMS:           p95,
		IntentDistribution:     intentDistribution,
		PricingTasksTotal:      s.pricingTasksTotal,
		PricingTasksDone:       s.pricingTasksDone,
		PricingTasksInProgress: s.pricingTasksTotal - s.pricingTasksDone,
		RecentPricingTasks:     recent,
	}
}
