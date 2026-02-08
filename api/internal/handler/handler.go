package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"golang-network-labs/api/internal/tcpclient"
)

// 핸들러 의존성
type Deps struct {
	DB  *sql.DB
	TCP *tcpclient.Client
}

// 핸들러 본체
type Handler struct {
	db  *sql.DB
	tcp *tcpclient.Client
}

func New(d Deps) *Handler {
	return &Handler{db: d.DB, tcp: d.TCP}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// user_id 헤더 추출
func userIDFromReq(rh interface{ Get(string) string }) string {
	// 헤더 값 읽기
	u := strings.TrimSpace(rh.Get("X-User-Id"))
	// 없으면 기본값
	if u == "" {
		return "anonymous"
	}
	return u
}

// request_id 생성
func newRequestID() string {
	// 8바이트 랜덤
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	// hex로 변환
	return hex.EncodeToString(b)
}

// 빈 에러는 NULL 처리
func nullableErr(s string) any {
	// 공백 제거
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// 로그 저장용 공통 시간
func now() time.Time {
	return time.Now()
}
