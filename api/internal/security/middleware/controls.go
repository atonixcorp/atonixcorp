package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var safeID = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,128}$`)

func SecurityRequestMetadata() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := ctx.GetHeader("X-Request-ID")
		if !safeID.MatchString(requestID) {
			var bytes [16]byte
			if _, err := rand.Read(bytes[:]); err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "The security request could not be completed.",
					"code":  "REQUEST_ID_UNAVAILABLE",
				})
				return
			}
			requestID = hex.EncodeToString(bytes[:])
		}
		correlationID := ctx.GetHeader("X-Correlation-ID")
		if !safeID.MatchString(correlationID) {
			correlationID = requestID
		}
		ctx.Set("security_request_id", requestID)
		ctx.Set("security_correlation_id", correlationID)
		ctx.Header("X-Request-ID", requestID)
		ctx.Header("Cache-Control", "no-store")
		ctx.Next()
	}
}

type rateBucket struct {
	started time.Time
	count   int
}

type principalLimiter struct {
	mu      sync.Mutex
	buckets map[uint]rateBucket
	max     int
	window  time.Duration
}

func SecurityRateLimit(max int, window time.Duration) gin.HandlerFunc {
	limiter := &principalLimiter{
		buckets: make(map[uint]rateBucket),
		max:     max,
		window:  window,
	}
	return func(ctx *gin.Context) {
		value, exists := ctx.Get("user_id")
		userID, ok := value.(uint)
		if !exists || !ok || userID == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required.", "code": "AUTHENTICATION_REQUIRED",
				"correlation_id": ctx.GetString("security_correlation_id"),
			})
			return
		}
		retryAfter, limited := limiter.take(userID, time.Now())
		if limited {
			ctx.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":          "Security request rate limit exceeded.",
				"code":           "RATE_LIMITED",
				"correlation_id": ctx.GetString("security_correlation_id"),
			})
			return
		}
		ctx.Next()
	}
}

func (l *principalLimiter) take(userID uint, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, exists := l.buckets[userID]
	if !exists || now.Sub(bucket.started) >= l.window {
		if !exists && len(l.buckets) >= 10000 {
			for id, existing := range l.buckets {
				if now.Sub(existing.started) >= l.window {
					delete(l.buckets, id)
				}
			}
			if len(l.buckets) >= 10000 {
				return l.window, true
			}
		}
		l.buckets[userID] = rateBucket{started: now, count: 1}
		return 0, false
	}
	if bucket.count >= l.max {
		return l.window - now.Sub(bucket.started), true
	}
	bucket.count++
	l.buckets[userID] = bucket
	return 0, false
}
