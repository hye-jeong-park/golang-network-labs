package tcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"time"
)

// TCP 요청 스키마
type Req struct {
	// 추적용 ID
	RequestID string `json:"request_id,omitempty" yaml:"request_id,omitempty" form:"request_id"`
	// 사용자 ID
	UserID string `json:"user_id,omitempty" yaml:"user_id,omitempty" form:"user_id"`
	// 작업 타입(cmd/file)
	Type string `json:"type,omitempty" yaml:"type,omitempty" form:"type"`

	// cmd 실행
	Cmd string `json:"cmd,omitempty" yaml:"cmd,omitempty" form:"cmd"`

	// 파일 읽기
	Path   string `json:"path,omitempty" yaml:"path,omitempty" form:"path"`
	Offset int64  `json:"offset,omitempty" yaml:"offset,omitempty" form:"offset"`
	Limit  int64  `json:"limit,omitempty" yaml:"limit,omitempty" form:"limit"`
}

// TCP 응답 스키마
type Res struct {
	// 성공 여부
	Ok bool `json:"ok,omitempty" yaml:"ok,omitempty"`
	// 출력(텍스트)
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
	// 에러 메시지
	Error string `json:"error,omitempty" yaml:"error,omitempty"`

	// 추적용 ID
	RequestID string `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	// 사용자 ID
	UserID string `json:"user_id,omitempty" yaml:"user_id,omitempty"`

	// TCP 주소 로그
	TcpLocal  string `json:"tcp_local,omitempty" yaml:"tcp_local,omitempty"`
	TcpRemote string `json:"tcp_remote,omitempty" yaml:"tcp_remote,omitempty"`

	// 파일 청크(Base64)
	FileB64 string `json:"file_b64,omitempty" yaml:"file_b64,omitempty"`
	// 다음 오프셋
	NextOffset int64 `json:"next_offset,omitempty" yaml:"next_offset,omitempty"`
	// EOF 여부
	EOF bool `json:"eof,omitempty" yaml:"eof,omitempty"`
}

// TCP 클라이언트 설정
type Config struct {
	Host        string
	Port        string
	DialTimeout time.Duration
	IOTimeout   time.Duration
}

// TCP 클라이언트
type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// TCP 서버에 명령을 보내고 응답을 받는 함수
// - context를 통해 요청 취소/타임아웃 전파
func (c *Client) Call(ctx context.Context, req Req) Res {
	addr := c.cfg.Host + ":" + c.cfg.Port

	// TCP 연결 (Context + 연결 타임아웃 적용)
	dialer := net.Dialer{Timeout: c.cfg.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Res{Ok: false, Error: err.Error(), RequestID: req.RequestID, UserID: req.UserID}
	}
	defer conn.Close()

	// TCP 읽기/쓰기 전체 타임아웃 설정
	_ = conn.SetDeadline(time.Now().Add(c.cfg.IOTimeout))

	// 요청 JSON 생성
	b, _ := json.Marshal(req)

	// 한 줄(JSON + '\n') 프로토콜로 전송
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return Res{Ok: false, Error: err.Error(), RequestID: req.RequestID, UserID: req.UserID}
	}

	// 응답 한 줄 수신
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return Res{Ok: false, Error: err.Error(), RequestID: req.RequestID, UserID: req.UserID}
	}

	// JSON → Res 파싱
	var res Res
	if err := json.Unmarshal(line, &res); err != nil {
		return Res{Ok: false, Error: err.Error(), RequestID: req.RequestID, UserID: req.UserID}
	}

	return res
}
