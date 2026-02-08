package handler

import (
	"net/http"
	"sync/atomic"
)

// 처리중 요청 수
var inFlight int64

// 증가
func incInFlight() { atomic.AddInt64(&inFlight, 1) }

// 감소
func decInFlight() { atomic.AddInt64(&inFlight, -1) }

// 메트릭 출력
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	// 텍스트 포맷
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("in_flight " + itoa64(atomic.LoadInt64(&inFlight)) + "\n"))
}

// int64 → string 변환
func itoa64(v int64) string {
	// 0 처리
	if v == 0 {
		return "0"
	}
	// 부호 처리
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	// 자리수 변환
	buf := make([]byte, 0, 20)
	for v > 0 {
		d := v % 10
		buf = append([]byte{byte('0' + d)}, buf...)
		v /= 10
	}
	return sign + string(buf)
}
