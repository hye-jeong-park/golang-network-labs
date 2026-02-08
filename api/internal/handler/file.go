package handler

import (
	"net/http"
	"strconv"
	"strings"

	"golang-network-labs/api/internal/tcpclient"
)

// /file 응답 스키마
type FileReadResult struct {
	// 추적용 ID
	RequestID string `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	// 사용자 ID
	UserID string `json:"user_id,omitempty" yaml:"user_id,omitempty"`

	// 파일 파라미터
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Offset int64  `json:"offset,omitempty" yaml:"offset,omitempty"`
	Limit  int64  `json:"limit,omitempty" yaml:"limit,omitempty"`

	// 결과 필드
	Ok         bool   `json:"ok,omitempty" yaml:"ok,omitempty"`
	Error      string `json:"error,omitempty" yaml:"error,omitempty"`
	FileB64    string `json:"file_b64,omitempty" yaml:"file_b64,omitempty"`
	NextOffset int64  `json:"next_offset,omitempty" yaml:"next_offset,omitempty"`
	EOF        bool   `json:"eof,omitempty" yaml:"eof,omitempty"`
}

// /file: TCP로 파일 청크 읽기
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	// inFlight 증가
	incInFlight()
	// 종료 시 감소
	defer decInFlight()

	// user_id 추출
	userID := userIDFromReq(r.Header)
	// request_id 생성
	reqID := newRequestID()

	// path 파라미터
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	// offset 기본값
	offset := int64(0)
	// limit 기본값
	limit := int64(4096)

	// offset 파싱
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			offset = n
		}
	}

	// limit 파싱
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}

	// TCP 요청 구성
	tcpReq := tcpclient.Req{
		RequestID: reqID,
		UserID:    userID,
		Type:      "file",
		Path:      path,
		Offset:    offset,
		Limit:     limit,
	}

	// TCP 호출
	res := h.tcp.Call(r.Context(), tcpReq)

	// file_reads 로그 저장
	_, _ = h.db.Exec(
		`INSERT INTO file_reads(ts, request_id, user_id, file_path, file_offset, limit_size, ok, err_msg)
		 VALUES (?,?,?,?,?,?,?,?)`,
		now(), reqID, userID, path, offset, limit, boolToInt(res.Ok), nullableErr(res.Error),
	)

	// 응답 구성
	out := FileReadResult{
		RequestID:  reqID,
		UserID:     userID,
		Path:       path,
		Offset:     offset,
		Limit:      limit,
		Ok:         res.Ok,
		Error:      res.Error,
		FileB64:    res.FileB64,
		NextOffset: res.NextOffset,
		EOF:        res.EOF,
	}

	// 응답 반환(JSON/YAML)
	writeResponse(w, r, out)
}
