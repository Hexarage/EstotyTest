package main

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*userLimiter
	mu       sync.RWMutex
	cleanup  *time.Ticker
}

type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var globalRateLimiter *RateLimiter

func init() {
	globalRateLimiter = &RateLimiter{
		limiters: make(map[string]*userLimiter),
		cleanup:  time.NewTicker(5 * time.Minute),
	}

	go globalRateLimiter.cleanupLoop()
}

// get rid of old limiters
func (rl *RateLimiter) cleanupLoop() {
	for range rl.cleanup.C {
		rl.mu.Lock()
		now := time.Now()
		for userId, ul := range rl.limiters {
			if now.Sub(ul.lastSeen) > 10*time.Minute {
				delete(rl.limiters, userId)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) CheckRateLimit(userId string, rps int, burst int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	ul, exists := rl.limiters[userId]
	if !exists {
		ul = &userLimiter{
			limiter: rate.NewLimiter(rate.Limit(rps), burst),
		}
		rl.limiters[userId] = ul
	}

	ul.lastSeen = time.Now()

	if !ul.limiter.Allow() {
		return errors.New("Rate limit exceeded: too many requests")
	}

	return nil
}

func RateLimitMiddleware(
	rpcFunc func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error),
	rps int,
	burst int,
) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {

	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", errors.New("authentication required")
		}

		if err := globalRateLimiter.CheckRateLimit(userID, rps, burst); err != nil {
			logger.Warn("Rate limit exceeded for user %s", userID)
			return "", runtime.NewError("Rate limit exceeded", 429)
		}

		return rpcFunc(ctx, logger, db, nk, payload)
	}
}

type RateLimitResponse struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after_seconds"`
}
