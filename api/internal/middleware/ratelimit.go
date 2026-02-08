package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IP별 레이트리밋
func RateLimitPerIP(rps float64, burst int) func(http.Handler) http.Handler {
	// 클라이언트 상태
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// 동기화
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// 오래된 엔트리 정리
	go func() {
		for {
			// 주기적 정리
			time.Sleep(1 * time.Minute)

			mu.Lock()
			for ip, c := range clients {
				// 5분 이상 미사용이면 제거
				if time.Since(c.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// IP 추출
			ip := clientIP(r)

			// limiter 조회/생성
			mu.Lock()
			c, ok := clients[ip]
			if !ok {
				c = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				clients[ip] = c
			}
			// 마지막 접근 갱신
			c.lastSeen = time.Now()
			lim := c.limiter
			mu.Unlock()

			// 토큰 없으면 거절
			if !lim.Allow() {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			// 다음 핸들러 수행
			next.ServeHTTP(w, r)
		})
	}
}

// 클라이언트 IP 추출
func clientIP(r *http.Request) string {
	// 프록시 환경 고려
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// 첫 IP만 사용
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// RemoteAddr 분해
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
