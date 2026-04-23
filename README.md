# Car Mall (Cloud-Native Demo)

This project implements a runnable prototype of a car sales system focusing on:

- Cloud-native microservices architecture (Go + Kubernetes)
- Intent Recognition
- Smart Pricing
- Asynchronous processing (Message Queue style)
- Multi-turn context caching (Redis style)
- High concurrency load testing and 10x traffic impact verification

## Architecture Overview

This codebase is an **educational and experimental version** designed to be flexible for component replacement with production-grade solutions later:

- `cmd/gateway`: Unified API entry point (can be replaced with Kitex gateway)
- `internal/intent`: Intent recognition module (current: rule engine)
- `internal/pricing`: Smart pricing module
- `internal/async`: Asynchronous queue abstraction (current: in-memory, can be replaced with RocketMQ)
- `internal/cache`: Session cache abstraction (current: in-memory, can be replaced with Redis)

## Quick Start

### Local Run

```bash
go run ./cmd/gateway
```

## Configuration

You can use local `.env` file:
```bash
cp .env.example .env
```

Start the service:
```bash
go run ./cmd/gateway
```

Health check:
```bash
curl http://localhost:8080/healthz
```

Chat request:
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id":"session-1",
    "user_id":"user-1",
    "message":"Can I get a discount price for EV-Pro?"
  }'
```

Visual dashboard:
```bash
open http://localhost:8080/dashboard
```

Dashboard features:
- Real-time total requests, requests in last minute
- Average latency and P95 latency
- Intent distribution pie chart
- Request rate trend chart
- Asynchronous smart pricing task status and results

Car e-commerce mall page:
```bash
open http://localhost:8080/mall
```

The page includes car model list, price filters, consultation entrance, and order interaction.

Current demo mall has 2000 built-in car models, maintained in `internal/catalog/cars.go` and generated in batch from template models.

Order payment page:
```bash
open "http://localhost:8080/payment?order_id=ord-1"
```

You can automatically jump to this page after adding to cart and submitting order in the mall page, or manually access with `order_id` to check status and simulate payment completion.

### Docker Compose (with Redis + RocketMQ)

```bash
cd deploy
docker compose up
```

## Load Testing and Performance Targets

Load testing script: `benchmark/k6-chat.js`
- `baseline`: 40 RPS for 60 seconds
- `flash_sale_10x`: 400 RPS for 120 seconds
- Threshold: `p95 < 3000ms`

Run load test:
```bash
k6 run benchmark/k6-chat.js
```

## Kubernetes and Auto Scaling

Deployment manifests in `deploy/k8s`:
- `gateway-deployment.yaml`: Gateway Deployment + Service
- `gateway-hpa.yaml`: HPA auto-scaling policy
- `pricing-worker-spot.yaml`: Pricing Worker on Spot instances (cost reduction)

## Cost Analysis Recommendations

For experiment reports, it is recommended to compare three models:
1. All on-demand instances (baseline cost)
2. Pricing Workers using Spot instances
3. Spot + Auto Scaling

Metrics to record:
- Cost per hour (USD / CNY)
- 99th percentile response time
- Peak throughput (QPS)
- Failure rate (HTTP 5xx / timeouts)

Target: 40%~60% cost reduction on non-critical computing tasks.

## Next Steps for Enhancement

1. Use Kitex to split `gateway/intent/pricing` into independent RPC services
2. Integrate real Redis and RocketMQ clients
3. Introduce OpenTelemetry + Prometheus + Grafana for observability
4. Reproduce experiments with JMeter and export reports
