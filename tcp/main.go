package main

import (
	"bufio"
	"encoding/json"
	"net"
	"os/exec"
	"runtime"
)

type Req struct {
	Cmd string `json:"cmd"`
}

type Res struct {
	Ok     bool   `json:"ok"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

func (r Res) WriteLine(conn net.Conn) {
	b, _ := json.Marshal(r)
	conn.Write(append(b, '\n'))
}

// A-1. 프로그램 시작
func main() {
	//net.Listen("tcp", ":9000") 호출
	//OS에 “9000 포트 TCP 서버 열어줘” 요청
	//성공하면 ln(listener) 생성
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		panic(err)
	}
	//A-2. 연결을 계속 받는 루프
	for {
		//클라이언트가 TCP로 접속할 때마다 Accept()가 연결을 하나 반환
		conn, _ := ln.Accept()
		//연결마다 handle(conn)을 고루틴으로 실행 → 여러 요청이 동시에 와도 각각 병렬 처리 가능
		go handle(conn)
	}
}

// A-3. 연결 처리(handle)
func handle(conn net.Conn) {
	//defer conn.Close()로 끝날 때 연결 닫도록 예약
	defer conn.Close()
	//“한 줄 읽기” 준비
	br := bufio.NewReader(conn)

	//TCP에서 '\n' 나올 때까지 읽어서 한 줄 확보
	//즉 TCP 프로토콜이 “JSON 한 줄”인 이유가 여기서 생김
	line, err := br.ReadBytes('\n')
	if err != nil {
		return
	}

	var req Req
	//JSON → Req{Cmd: ...} 구조체로 변환
	if err := json.Unmarshal(line, &req); err != nil {
		//실패하면 {ok:false, error:"bad json"} 응답하고 종료
		Res{Ok: false, Error: "bad json"}.WriteLine(conn)
		return
	}

	if runtime.GOOS != "windows" {
		//Windows 명령을 실행할 수 없으니 에러 응답하고 종료
		Res{Ok: false, Error: "run tcp server on Windows"}.WriteLine(conn)
		return
	}

	//Windows 명령 실행
	out, err := exec.Command("cmd", "/C", req.Cmd).CombinedOutput()
	if err != nil {
		//실패면 {ok:false, output:"...", error:"..."}를 JSON 한 줄로 write
		Res{Ok: false, Output: string(out), Error: err.Error()}.WriteLine(conn)
		return
	}
	//성공이면 {ok:true, output:"..."}를 JSON 한 줄로 write
	Res{Ok: true, Output: string(out)}.WriteLine(conn)

	//handle() 끝 → defer conn.Close() 실행 → 연결 닫힘
}
