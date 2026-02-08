package filex

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"golang-network-labs/tcp/internal/protocol"
)

// 파일 루트 제한
var fileRoot = "/data"

// 파일 청크 읽기
func ReadChunk(req protocol.Req, base protocol.Res) protocol.Res {
	// limit 기본값
	limit := req.Limit
	if limit <= 0 {
		limit = 4096
	}

	// limit 제한(1MB)
	if limit > 1<<20 {
		limit = 1 << 20
	}

	// offset 기본값
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// path 공백 제거
	p := strings.TrimSpace(req.Path)
	if p == "" {
		base.Ok = false
		base.Error = "path required"
		return base
	}

	// 경로 정규화
	clean := filepath.Clean(p)

	// 루트 하위로 강제
	abs := filepath.Join(fileRoot, clean)

	// 루트 탈출 방지
	if !strings.HasPrefix(abs, fileRoot) {
		base.Ok = false
		base.Error = "invalid path"
		return base
	}

	// 파일 오픈
	f, err := os.Open(abs)
	if err != nil {
		base.Ok = false
		base.Error = err.Error()
		return base
	}
	defer f.Close()

	// 오프셋 이동
	if _, err := f.Seek(offset, 0); err != nil {
		base.Ok = false
		base.Error = err.Error()
		return base
	}

	// 버퍼 준비
	buf := make([]byte, limit)

	// 읽기 수행
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		base.Ok = false
		base.Error = err.Error()
		return base
	}

	// 실제 청크
	chunk := buf[:n]

	// Base64 인코딩
	base.FileB64 = base64.StdEncoding.EncodeToString(chunk)

	// 다음 오프셋 계산
	base.NextOffset = offset + int64(n)

	// EOF 추정
	base.EOF = n < int(limit)

	// 성공 처리
	base.Ok = true
	base.Output = "file chunk read"
	base.Error = ""
	return base
}
