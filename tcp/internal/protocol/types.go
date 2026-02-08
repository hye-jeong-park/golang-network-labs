package protocol

import (
	"encoding/json"
	"net"
)

// 요청 스키마
type Req struct {
	// 추적용 ID
	RequestID string `json:"request_id"`
	// 사용자 ID
	UserID string `json:"user_id"`
	// 작업 타입(cmd/file)
	Type string `json:"type"`

	// cmd 실행
	Cmd string `json:"cmd"`

	// 파일 읽기
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
}

// 응답 스키마
type Res struct {
	// 성공 여부
	Ok bool `json:"ok"`
	// 출력 텍스트
	Output string `json:"output"`
	// 에러 메시지
	Error string `json:"error"`

	// 추적용 ID
	RequestID string `json:"request_id"`
	// 사용자 ID
	UserID string `json:"user_id"`

	// 연결 정보
	TcpLocal  string `json:"tcp_local"`
	TcpRemote string `json:"tcp_remote"`

	// 파일 청크(Base64)
	FileB64 string `json:"file_b64"`
	// 다음 오프셋
	NextOffset int64 `json:"next_offset"`
	// EOF 여부
	EOF bool `json:"eof"`
}

// 응답 한 줄 전송
func (r Res) WriteLine(conn net.Conn) {
	// JSON 직렬화
	b, _ := json.Marshal(r)
	// 한 줄 프로토콜
	_, _ = conn.Write(append(b, '\n'))
}
