package azure

import (
	"sync"
	"time"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// tokenBucket реализует алгоритм Token Bucket
type tokenBucket struct {
	tokens     float64   // текущее количество токенов
	maxTokens  float64   // максимальное количество токенов
	refillRate float64   // скорость пополнения (токенов в секунду)
	lastRefill time.Time // время последнего пополнения
}

// RateLimiter конфигурация ограничителя запросов
type RateLimiterConfig struct {
	RequestsPerSecond float64       // Максимальное количество запросов в секунду
	BurstSize         int           // Максимальный размер "пачки" запросов
	ByIP              bool          // Ограничивать по IP (true) или глобально (false)
	CleanupInterval   time.Duration // Интервал очистки старых записей
}

// RateLimiter ограничитель запросов
type RateLimiter struct {
	config  RateLimiterConfig
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
}

// Новый RateLimiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 10
	}
	if config.BurstSize <= 0 {
		config.BurstSize = int(config.RequestsPerSecond) * 2
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = time.Minute
	}

	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
	}

	// Запускаем фоновую очистку старых записей
	go rl.cleanup()

	return rl
}

// Очистка старых записей
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			bucket.refill()
			if bucket.tokens >= bucket.maxTokens {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Создать или получить bucket для ключа
func (rl *RateLimiter) getBucket(key string) *tokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Проверяем ещё раз после захвата блокировки
	if bucket, exists = rl.buckets[key]; exists {
		return bucket
	}

	bucket = &tokenBucket{
		tokens:     float64(rl.config.BurstSize),
		maxTokens:  float64(rl.config.BurstSize),
		refillRate: rl.config.RequestsPerSecond,
		lastRefill: time.Now(),
	}
	rl.buckets[key] = bucket
	return bucket
}

// Пополнение токенов
func (tb *tokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now
}

// Проверка доступности токена
func (tb *tokenBucket) allow() bool {
	tb.refill()
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Middleware для Rate Limiter
func (rl *RateLimiter) Middleware() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		// Получаем ключ для ограничения
		key := "global"
		if rl.config.ByIP {
			key = c.ultra.GetRemoteAddr()
		}

		bucket := rl.getBucket(key)

		if !bucket.allow() {
			c.ultra.SetStatus(429, "Too Many Requests")
			c.ultra.SetJSON(M{
				"error":       "Too Many Requests",
				"retry_after": int(1.0 / rl.config.RequestsPerSecond),
			})
			return
		}

		next(c.ultra)
	}
}

// RateLimitByIP создаёт Rate Limiter с ограничением по IP
func RateLimitByIP(requestsPerSecond float64, burstSize int) *RateLimiter {
	return NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         burstSize,
		ByIP:              true,
	})
}

// RateLimitGlobal создаёт глобальный Rate Limiter
func RateLimitGlobal(requestsPerSecond float64, burstSize int) *RateLimiter {
	return NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         burstSize,
		ByIP:              false,
	})
}
