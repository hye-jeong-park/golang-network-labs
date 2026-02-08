package execx

import (
	"os/exec"
	"runtime"
	"strings"

	"golang-network-labs/tcp/internal/protocol"
)

// 허용 명령
var allowCmd = map[string]bool{
	"uname":  true,
	"date":   true,
	"whoami": true,
	"id":     true,
	"ls":     true,
	"pwd":    true,
}

// cmd 실행 처리
func Run(req protocol.Req, base protocol.Res) protocol.Res {
	// cmd 공백 제거
	cmdText := strings.TrimSpace(req.Cmd)
	if cmdText == "" {
		base.Ok = false
		base.Error = "cmd required"
		return base
	}

	// 토큰 분리
	tokens := strings.Fields(cmdText)
	if len(tokens) == 0 {
		base.Ok = false
		base.Error = "cmd required"
		return base
	}

	// allowlist 검사
	mainCmd := tokens[0]
	if !allowCmd[mainCmd] {
		base.Ok = false
		base.Error = "command not allowed"
		return base
	}

	// OS별 실행 분기
	var out []byte
	var err error

	// 윈도우면 cmd
	if runtime.GOOS == "windows" {
		out, err = exec.Command("cmd", "/C", cmdText).CombinedOutput()
	} else {
		// 리눅스면 sh
		out, err = exec.Command("sh", "-c", cmdText).CombinedOutput()
	}

	// 실패 처리
	if err != nil {
		base.Ok = false
		base.Output = string(out)
		base.Error = err.Error()
		return base
	}

	// 성공 처리
	base.Ok = true
	base.Output = string(out)
	return base
}
