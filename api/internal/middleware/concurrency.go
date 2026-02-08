package middleware

import (
	"net/http"
)

// 동시 실행 제한 미들웨어(세마포어 방식)
// max가 0 이하면 제한하지 않음
func ConcurrencyLimit(max int) func(http.Handler) http.Handler {
	if max <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	// 슬롯 채널 생성
	sem := make(chan struct{}, max)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 실행 슬롯이 없으면 즉시 429 반환
			select {
			case sem <- struct{}{}:
				// 슬롯 확보 성공 → 종료 시 반납
				defer func() { <-sem }()
			default:
				// 슬롯 부족이면 429
				http.Error(w, "too many requests", http.StatusTooManyRequests) // 429
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
