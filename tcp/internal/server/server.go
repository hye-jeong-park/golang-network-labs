package server

import (
	"net"

	"golang-network-labs/tcp/internal/handler"
)

// 서버 설정
type Config struct {
	// 리슨 주소
	Addr string
}

// 서버 본체
type Server struct {
	// 설정 보관
	cfg Config
	// 핸들러 보관
	h *handler.Handler
}

// 서버 생성
func New(cfg Config) *Server {
	// 핸들러 생성
	h := handler.New()
	// 서버 반환
	return &Server{cfg: cfg, h: h}
}

// TCP 리슨 + Accept 루프
func (s *Server) ListenAndServe() error {
	// 리슨 시작
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	// 계속 연결 처리
	for {
		// 연결 수락
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		// 연결은 고루틴 처리
		go s.h.Handle(conn)
	}
}
