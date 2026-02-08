package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang-network-labs/api/internal/tcpclient"

	"gopkg.in/yaml.v3"
)

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	// inFlight 증가(메트릭)
	incInFlight()
	// 종료 시 감소
	defer decInFlight()

	// user_id 추출
	userID := userIDFromReq(r.Header)
	// request_id 생성
	reqID := newRequestID()

	var cmd string

	// 1) cmd 파싱 (GET + POST)
	switch r.Method {
	case http.MethodGet:
		// GET 쿼리에서 cmd
		cmd = r.URL.Query().Get("cmd")

	case http.MethodPost:
		// body 크기 제한(1MB)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		// Content-Type 확인
		ct := r.Header.Get("Content-Type")

		// JSON 요청 처리
		if strings.HasPrefix(ct, "application/json") {
			var req tcpclient.Req
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			cmd = req.Cmd
		} else if strings.HasPrefix(ct, "application/x-yaml") || strings.HasPrefix(ct, "text/yaml") {
			// YAML 요청 처리
			var req tcpclient.Req
			if err := yaml.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid yaml", http.StatusBadRequest)
				return
			}
			cmd = req.Cmd
		} else if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			// form 파싱
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			cmd = r.Form.Get("cmd")
		} else if strings.HasPrefix(ct, "multipart/form-data") {
			// multipart 파싱
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				http.Error(w, "invalid multipart form", http.StatusBadRequest)
				return
			}
			cmd = r.FormValue("cmd")
		} else {
			// 지원하지 않는 타입
			http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
			return
		}

	default:
		// 지원하지 않는 메서드
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// cmd 공백 제거
	cmd = strings.TrimSpace(cmd)
	// 빈 cmd 거절
	if cmd == "" {
		http.Error(w, "cmd required", http.StatusBadRequest)
		return
	}

	// TCP 요청 구성
	tcpReq := tcpclient.Req{
		RequestID: reqID,
		UserID:    userID,
		Type:      "cmd",
		Cmd:       cmd,
	}

	// TCP 호출(컨텍스트 포함)
	res := h.tcp.Call(r.Context(), tcpReq)

	// 실행 로그 저장
	_, _ = h.db.Exec(
		`INSERT INTO logs(ts, request_id, user_id, cmd, ok, tcp_local, tcp_remote, err_msg)
		 VALUES (?,?,?,?,?,?,?,?)`,
		now(), reqID, userID, cmd, boolToInt(res.Ok), res.TcpLocal, res.TcpRemote, nullableErr(res.Error),
	)

	// 응답 반환(JSON/YAML)
	writeResponse(w, r, res)
}
