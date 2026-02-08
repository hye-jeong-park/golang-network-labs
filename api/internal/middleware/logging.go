package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// 간단 요청 로거
func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 시작 시각
			start := time.Now()
			// 핸들러 실행
			next.ServeHTTP(w, r)
			//처리 시간 계산
			elapsed := time.Since(start)

			fmt.Printf("[http] %s %s (%s)\n", r.Method, r.URL.Path, elapsed)
		})
	}
}
