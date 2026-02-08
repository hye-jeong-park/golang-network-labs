package handler

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"

	"golang-network-labs/tcp/internal/execx"
	"golang-network-labs/tcp/internal/filex"
	"golang-network-labs/tcp/internal/protocol"
)

// 핸들러 본체
type Handler struct{}

// 핸들러 생성
func New() *Handler {
	// 단순 생성
	return &Handler{}
}

// 연결 처리
func (h *Handler) Handle(conn net.Conn) {

	defer conn.Close()

	// 연결 주소 확보
	local := conn.LocalAddr().String()
	remote := conn.RemoteAddr().String()

	// 한 줄 읽기 준비
	br := bufio.NewReader(conn)

	// 한 줄 수신
	line, err := br.ReadBytes('\n')
	if err != nil {
		return
	}

	// 요청 파싱
	var req protocol.Req
	if err := json.Unmarshal(line, &req); err != nil {
		// 파싱 실패 응답
		protocol.Res{
			Ok:        false,
			Error:     "bad json",
			TcpLocal:  local,
			TcpRemote: remote,
		}.WriteLine(conn)
		return
	}

	// user 기본값
	if strings.TrimSpace(req.UserID) == "" {
		req.UserID = "anonymous"
	}

	// request_id 기본값
	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = "no-request-id"
	}

	// type 기본값
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "cmd"
	}

	// 공통 응답 필드
	base := protocol.Res{
		RequestID: req.RequestID,
		UserID:    req.UserID,
		TcpLocal:  local,
		TcpRemote: remote,
	}

	// 타입 분기
	switch req.Type {
	case "cmd":
		// cmd 실행 처리
		res := execx.Run(req, base)
		res.WriteLine(conn)
		return

	case "file":
		// 파일 읽기 처리
		res := filex.ReadChunk(req, base)
		res.WriteLine(conn)
		return

	default:
		// 미지원 타입 처리
		base.Ok = false
		base.Error = "unsupported type"
		base.WriteLine(conn)
		return
	}
}
