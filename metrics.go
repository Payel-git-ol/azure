package azure

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Path        string    // Путь для отдачи метрик (по умолчанию /metrics)
	Enabled     bool      // Включить сбор метрик
	Buckets     []float64 // Гистограммы для времени ответа
	EnableStats bool      // Включить детальную статистику
}

// Metrics метрики фреймворка
type Metrics struct {
	config        MetricsConfig
	requests      int64
	errors        int64
	activeReqs    int64
	totalDuration int64 // в наносекундах

	// Статистика по методам
	getRequests    int64
	postRequests   int64
	putRequests    int64
	deleteRequests int64

	// Статистика по статусам
	status2xx int64
	status4xx int64
	status5xx int64

	// Гистограмма времени ответа (упрощённая)
	durationBuckets []int64

	mu sync.RWMutex
}

// Новые метрики
func NewMetrics(config MetricsConfig) *Metrics {
	if config.Path == "" {
		config.Path = "/metrics"
	}
	if config.Enabled == false {
		config.Enabled = true
	}
	if config.Buckets == nil {
		config.Buckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10}
	}

	return &Metrics{
		config:          config,
		durationBuckets: make([]int64, len(config.Buckets)),
	}
}

// Middleware для сбора метрик
func (m *Metrics) Middleware() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		if !m.config.Enabled {
			next(c.ultra)
			return
		}

		start := time.Now()
		atomic.AddInt64(&m.requests, 1)
		atomic.AddInt64(&m.activeReqs, 1)

		// Считаем по методам
		method := c.ultra.GetMethod()
		switch method {
		case "GET":
			atomic.AddInt64(&m.getRequests, 1)
		case "POST":
			atomic.AddInt64(&m.postRequests, 1)
		case "PUT":
			atomic.AddInt64(&m.putRequests, 1)
		case "DELETE":
			atomic.AddInt64(&m.deleteRequests, 1)
		}

		// Вызываем следующий хендлер
		next(c.ultra)

		// Считаем длительность
		duration := time.Since(start)
		atomic.AddInt64(&m.totalDuration, int64(duration))
		atomic.AddInt64(&m.activeReqs, -1)

		// Обновляем гистограмму
		m.updateHistogram(duration.Seconds())

		// Считаем по статусам
		status := c.ultra.Response.Status
		if status >= 200 && status < 300 {
			atomic.AddInt64(&m.status2xx, 1)
		} else if status >= 400 && status < 500 {
			atomic.AddInt64(&m.status4xx, 1)
		} else if status >= 500 {
			atomic.AddInt64(&m.status5xx, 1)
			atomic.AddInt64(&m.errors, 1)
		}
	}
}

// Обновление гистограммы
func (m *Metrics) updateHistogram(duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, bucket := range m.config.Buckets {
		if duration <= bucket {
			m.durationBuckets[i]++
		}
	}
}

// Отдача метрик в Prometheus формате
func (m *Metrics) ServeMetrics() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		path := c.ultra.GetPath()
		if path == m.config.Path {
			c.ultra.SetHeader("Content-Type", "text/plain; version=0.0.4")
			c.ultra.SetText(m.formatMetrics())
			return
		}
		next(c.ultra)
	}
}

// Форматирование метрик
func (m *Metrics) formatMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := atomic.LoadInt64(&m.requests)
	errors := atomic.LoadInt64(&m.errors)
	activeReqs := atomic.LoadInt64(&m.activeReqs)
	totalDuration := atomic.LoadInt64(&m.totalDuration)

	getReqs := atomic.LoadInt64(&m.getRequests)
	postReqs := atomic.LoadInt64(&m.postRequests)
	putReqs := atomic.LoadInt64(&m.putRequests)
	deleteReqs := atomic.LoadInt64(&m.deleteRequests)

	status2xx := atomic.LoadInt64(&m.status2xx)
	status4xx := atomic.LoadInt64(&m.status4xx)
	status5xx := atomic.LoadInt64(&m.status5xx)

	avgDuration := float64(0)
	if requests > 0 {
		avgDuration = float64(totalDuration) / float64(requests) / 1e6 // в мс
	}

	// Формируем вывод в формате Prometheus
	out := "# HELP azure_requests_total Total number of requests\n"
	out += "# TYPE azure_requests_total counter\n"
	out += fmt.Sprintf("azure_requests_total %d\n\n", requests)

	out += "# HELP azure_errors_total Total number of errors\n"
	out += "# TYPE azure_errors_total counter\n"
	out += fmt.Sprintf("azure_errors_total %d\n\n", errors)

	out += "# HELP azure_active_requests Current number of active requests\n"
	out += "# TYPE azure_active_requests gauge\n"
	out += fmt.Sprintf("azure_active_requests %d\n\n", activeReqs)

	out += "# HELP azure_request_duration_seconds Request duration in seconds\n"
	out += "# TYPE azure_request_duration_seconds histogram\n"
	for i, bucket := range m.config.Buckets {
		out += fmt.Sprintf("azure_request_duration_seconds_bucket{le=\"%.3f\"} %d\n", bucket, m.durationBuckets[i])
	}
	out += fmt.Sprintf("azure_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", requests)
	out += fmt.Sprintf("azure_request_duration_seconds_sum %.3f\n", float64(totalDuration)/1e9)
	out += fmt.Sprintf("azure_request_duration_seconds_count %d\n\n", requests)

	out += "# HELP azure_requests_by_method Requests by HTTP method\n"
	out += "# TYPE azure_requests_by_method counter\n"
	out += fmt.Sprintf("azure_requests_by_method{method=\"GET\"} %d\n", getReqs)
	out += fmt.Sprintf("azure_requests_by_method{method=\"POST\"} %d\n", postReqs)
	out += fmt.Sprintf("azure_requests_by_method{method=\"PUT\"} %d\n", putReqs)
	out += fmt.Sprintf("azure_requests_by_method{method=\"DELETE\"} %d\n\n", deleteReqs)

	out += "# HELP azure_requests_by_status Requests by status code\n"
	out += "# TYPE azure_requests_by_status counter\n"
	out += fmt.Sprintf("azure_requests_by_status{code=\"2xx\"} %d\n", status2xx)
	out += fmt.Sprintf("azure_requests_by_status{code=\"4xx\"} %d\n", status4xx)
	out += fmt.Sprintf("azure_requests_by_status{code=\"5xx\"} %d\n", status5xx)

	out += "# HELP azure_avg_request_duration_ms Average request duration in milliseconds\n"
	out += "# TYPE azure_avg_request_duration_ms gauge\n"
	out += fmt.Sprintf("azure_avg_request_duration_ms %.2f\n", avgDuration)

	return out
}

// GetStats возвращает текущую статистику
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgDuration := float64(0)
	if m.requests > 0 {
		avgDuration = float64(m.totalDuration) / float64(m.requests) / 1e6
	}

	// Копируем durationBuckets
	buckets := make([]int64, len(m.durationBuckets))
	copy(buckets, m.durationBuckets)

	return map[string]interface{}{
		"requests":        m.requests,
		"errors":          m.errors,
		"active_requests": m.activeReqs,
		"avg_duration_ms": avgDuration,
		"requests_by_method": map[string]int64{
			"GET":    m.getRequests,
			"POST":   m.postRequests,
			"PUT":    m.putRequests,
			"DELETE": m.deleteRequests,
		},
		"requests_by_status": map[string]int64{
			"2xx": m.status2xx,
			"4xx": m.status4xx,
			"5xx": m.status5xx,
		},
		"duration_buckets": buckets,
	}
}

// Global metrics instance
var globalMetrics = NewMetrics(MetricsConfig{
	Path:    "/metrics",
	Enabled: true,
})

// UseMetrics включает сбор метрик
func UseMetrics(a *Azure) {
	// Middleware для сбора
	a.Use(func(c *Context, next ultrahttp.RouteHandler) {
		if !globalMetrics.config.Enabled {
			return
		}

		start := time.Now()
		atomic.AddInt64(&globalMetrics.requests, 1)
		atomic.AddInt64(&globalMetrics.activeReqs, 1)

		method := c.ultra.GetMethod()
		switch method {
		case "GET":
			atomic.AddInt64(&globalMetrics.getRequests, 1)
		case "POST":
			atomic.AddInt64(&globalMetrics.postRequests, 1)
		case "PUT":
			atomic.AddInt64(&globalMetrics.putRequests, 1)
		case "DELETE":
			atomic.AddInt64(&globalMetrics.deleteRequests, 1)
		}

		// Вызываем следующий хендлер напрямую через ultrahttp
		next(c.ultra)

		duration := time.Since(start)
		atomic.AddInt64(&globalMetrics.totalDuration, int64(duration))
		atomic.AddInt64(&globalMetrics.activeReqs, -1)
		globalMetrics.updateHistogram(duration.Seconds())

		status := c.ultra.Response.Status
		if status >= 200 && status < 300 {
			atomic.AddInt64(&globalMetrics.status2xx, 1)
		} else if status >= 400 && status < 500 {
			atomic.AddInt64(&globalMetrics.status4xx, 1)
		} else if status >= 500 {
			atomic.AddInt64(&globalMetrics.status5xx, 1)
			atomic.AddInt64(&globalMetrics.errors, 1)
		}
	})

	// Роут для отдачи метрик
	a.Get(globalMetrics.config.Path, func(c *Context) {
		c.ultra.SetHeader("Content-Type", "text/plain; version=0.0.4")
		c.ultra.SetText(globalMetrics.formatMetrics())
	})
}

// GetMetrics возвращает глобальные метрики
func GetMetrics() map[string]interface{} {
	return globalMetrics.GetStats()
}
