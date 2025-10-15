# Monitoring Setup for Stage 4

## Current Status

⚠️ **Monitoring infrastructure is PREPARED but NOT ACTIVE**

The Prometheus and Grafana services are defined in `docker-compose.yml` but commented out because:
- Metrics export is not yet implemented in the bot code
- Grafana dashboards are not yet created

## When Ready to Enable

### Step 1: Implement Metrics Export

Create `internal/metrics/prometheus.go` with the following metrics:

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // AI Decision Metrics
    AIDecisionTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_decision_total",
            Help: "Total AI decisions made",
        },
        []string{"regime", "approved"},
    )

    AIDecisionConfidence = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "ai_decision_confidence",
            Help: "Latest AI decision confidence score",
        },
    )

    // Policy Engine Metrics
    PolicyValidationsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "policy_validations_total",
            Help: "Total policy validations",
        },
        []string{"result"},
    )

    PolicyRiskScore = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "policy_risk_score",
            Help: "Current risk score (0.0-1.0)",
        },
    )

    // Trading Metrics
    TradesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "trades_total",
            Help: "Total trades executed",
        },
        []string{"side", "symbol", "strategy"},
    )

    PortfolioValueUSDT = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "portfolio_value_usdt",
            Help: "Current portfolio value in USDT",
        },
    )

    PnLRealizedUSDT = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "pnl_realized_usdt",
            Help: "Realized profit/loss in USDT",
        },
    )

    // Circuit Breaker Metrics
    CircuitBreakerActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "circuit_breaker_active",
            Help: "Whether circuit breaker is active (1=active, 0=inactive)",
        },
    )
)
```

### Step 2: Add Metrics Endpoint

In `internal/api/server.go`, add:

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) setupRoutes() {
    // Existing routes...

    // Prometheus metrics endpoint
    s.router.Handle("/metrics", promhttp.Handler())
}
```

### Step 3: Instrument Code

Add metric updates throughout the codebase:

```go
// In orchestrator.go
metrics.AIDecisionTotal.WithLabelValues(decision.Regime, "true").Inc()
metrics.AIDecisionConfidence.Set(decision.Confidence)

// In policy/engine.go
if validation.Approved {
    metrics.PolicyValidationsTotal.WithLabelValues("approved").Inc()
} else {
    metrics.PolicyValidationsTotal.WithLabelValues("rejected").Inc()
}
metrics.PolicyRiskScore.Set(validation.RiskScore)

// In executor.go
if result.Success {
    metrics.TradesTotal.WithLabelValues(side, symbol, "auto").Inc()
}
```

### Step 4: Enable Monitoring Services

Uncomment the monitoring services in `docker-compose.yml`:

```yaml
# Remove the # from prometheus, grafana services and their volumes
```

### Step 5: Start Services

```bash
docker-compose up -d prometheus grafana
```

Access:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### Step 6: Create Grafana Dashboards

Create JSON dashboards in `configs/grafana/dashboards/`:

**dashboard-stage4-overview.json** - Main Stage 4 metrics
**dashboard-trading.json** - Trading performance
**dashboard-risk.json** - Risk management metrics

## Useful Prometheus Queries

```promql
# AI decision approval rate (last 24h)
rate(ai_decision_total{approved="true"}[24h]) / rate(ai_decision_total[24h]) * 100

# Average risk score
avg_over_time(policy_risk_score[1h])

# Trades per hour
rate(trades_total[1h]) * 3600

# Current portfolio value
portfolio_value_usdt

# Circuit breaker status
circuit_breaker_active

# Policy violation rate
rate(policy_violations_total[1h])
```

## Architecture

```
Bot (port 8080)
  └── /metrics endpoint
       ↓
  Prometheus (port 9090)
       ↓ scrape every 15s
  Grafana (port 3000)
       └── Dashboards
```

## Security Notes

When enabling monitoring:
1. Change default Grafana password immediately
2. Consider adding authentication to Prometheus
3. Use TLS if exposing metrics externally
4. Restrict network access to monitoring ports

## Performance Impact

- Metrics collection: ~1-2ms per operation
- Prometheus scraping: minimal (15s intervals)
- Storage: ~100-200MB for 30 days of metrics
- Memory: +50-100MB for Prometheus

## Future Enhancements

- [ ] Alert rules for critical conditions
- [ ] Automated anomaly detection
- [ ] Performance optimization alerts
- [ ] Custom Grafana dashboards per strategy
- [ ] Metrics export to external services

---

**Status**: Ready for implementation when metrics export is added to bot code
