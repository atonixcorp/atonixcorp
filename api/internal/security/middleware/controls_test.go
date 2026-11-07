package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestPrincipalLimiterEnforcesWindowAndExpires(t *testing.T) {
	limiter := &principalLimiter{
		buckets: make(map[uint]rateBucket),
		max:     2,
		window:  time.Minute,
	}
	start := time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC)

	if _, limited := limiter.take(7, start); limited {
		t.Fatal("first request should pass")
	}
	if _, limited := limiter.take(7, start.Add(time.Second)); limited {
		t.Fatal("second request should pass")
	}
	if retry, limited := limiter.take(7, start.Add(2*time.Second)); !limited || retry <= 0 {
		t.Fatalf("third request should be rate limited with retry delay, got %v, %v", retry, limited)
	}
	if _, limited := limiter.take(7, start.Add(time.Minute)); limited {
		t.Fatal("request after the fixed window should pass")
	}
	if _, limited := limiter.take(8, start.Add(time.Minute)); limited {
		t.Fatal("rate limits should be isolated by authenticated subject")
	}
}

func TestPrincipalLimiterBoundsSubjectTable(t *testing.T) {
	limiter := &principalLimiter{
		buckets: make(map[uint]rateBucket, 10000),
		max:     1,
		window:  time.Minute,
	}
	now := time.Now()
	for id := uint(1); id <= 10000; id++ {
		limiter.buckets[id] = rateBucket{started: now, count: 1}
	}
	if retry, limited := limiter.take(10001, now); !limited || retry != time.Minute {
		t.Fatalf("new subject at capacity should be rejected, got %v, %v", retry, limited)
	}
}

func TestIPRateLimitDoesNotRequireAuthenticatedPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/internal", IPRateLimit(1, time.Minute), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPut, "/internal", nil)
	request.RemoteAddr = "192.0.2.10:43210"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("first unauthenticated operator request returned %d, want %d", response.Code, http.StatusNoContent)
	}

	request = httptest.NewRequest(http.MethodPut, "/internal", nil)
	request.RemoteAddr = "192.0.2.10:43211"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from the same IP returned %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestRequestClientIPUsesRemoteAddressNotForwardedHeader(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{remoteAddr: "192.0.2.10:443", want: "192.0.2.10"},
		{remoteAddr: "[2001:db8::1]:443", want: "2001:db8::1"},
		{remoteAddr: "unparseable", want: "unknown"},
	}
	for _, test := range tests {
		if got := requestClientIP(test.remoteAddr); got != test.want {
			t.Errorf("requestClientIP(%q) = %q, want %q", test.remoteAddr, got, test.want)
		}
	}
}
