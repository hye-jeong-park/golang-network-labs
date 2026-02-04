package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

type Req struct {
	Cmd string `json:"cmd"`
}

type Res struct {
	Ok     bool   `json:"ok"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

type TitleResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// B-1. 프로그램 시작 (API 서버)
func main() {
	// B-2. sqlite DB 열기 및 초기화
	// logs.db 파일 열기 (없으면 생성)
	// 로그 테이블 준비
	db := mustDB()
	defer db.Close()

	// B-3. /run API 등록
	// /run 경로로 요청 오면 실행될 핸들러 등록
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		// B-4. 쿼리 파라미터 cmd 읽기
		// ex: /run?cmd=ipconfig
		cmd := r.URL.Query().Get("cmd")

		// B-5. cmd가 없으면 에러 반환
		if cmd == "" {
			http.Error(w, "cmd required", 400)
			return
		}

		// B-6. TCP 응답을 받을 채널 생성
		// 버퍼 1칸 → 고루틴 블로킹 방지
		ch := make(chan Res, 1)

		// B-7. TCP 호출을 고루틴으로 실행
		//API 서버 스레드를 막지 않기 위해 TCP 통신을 별도 실행
		go func() {
			ch <- callTCP(cmd)
		}()

		// B-8. TCP 응답 대기
		res := <-ch

		// B-9. 실행 로그 DB 저장
		//언제, 어떤 명령을, 성공했는지 기록
		_, _ = db.Exec(
			"INSERT INTO logs(ts, cmd, ok) VALUES (?,?,?)",
			time.Now().Format(time.RFC3339), cmd, boolToInt(res.Ok),
		)
		// B-10. HTTP JSON 응답 반환
		writeJSON(w, res)
	})

	// B-11. /title API 등록 (HTML 파싱 실습용 API)
	http.HandleFunc("/title", func(w http.ResponseWriter, r *http.Request) {
		// B-12. url 파라미터 읽기
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "url required", 400)
			return
		}

		// B-13. HTTP GET + HTML 파싱
		title, err := fetchTitle(url) //외부 웹 요청, <title> 태그 추출
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// B-14. 결과 반환
		writeJSON(w, TitleResult{URL: url, Title: title})
	})

	fmt.Println("api :8080")

	// B-15. HTTP 서버 실행 및 대기 (8080 포트에서 HTTP 요청 대기, 여기서 메인 고루틴은 blocking)
	_ = http.ListenAndServe(":8080", nil)
}

// C-1. TCP 서버(127.0.0.1:9000)에 명령(cmd)을 보내고 Res를 받아오는 함수
func callTCP(cmd string) Res {
	// C-2. TCP 연결 시도 (클라이언트 역할)
	conn, err := net.Dial("tcp", "127.0.0.1:9000")

	// C-3. 연결 실패하면 실패 응답 구조체를 만들어 바로 반환
	if err != nil {
		return Res{Ok: false, Error: err.Error()}
	}
	defer conn.Close() // C-4. 함수 끝나면 TCP 연결 자동 닫기

	// C-5. 요청(Req{Cmd: cmd})을 JSON으로 변환
	b, _ := json.Marshal(Req{Cmd: cmd})

	// C-6. JSON 뒤에 '\n' 붙여서 한 줄 프로토콜로 전송
	conn.Write(append(b, '\n'))

	// C-7. 응답을 한 줄 단위로 읽기 위한 Reader 생성
	br := bufio.NewReader(conn)
	// C-8. '\n'까지 읽어서 응답 한 줄 수신 (서버가 응답할 때까지 대기)
	line, err := br.ReadBytes('\n')

	// C-9. 읽기 실패 시 실패 응답 반환
	if err != nil {
		return Res{Ok: false, Error: err.Error()}
	}
	var res Res // C-10. 응답 JSON을 Res 구조체로 파싱
	_ = json.Unmarshal(line, &res)
	return res
}

// D-1. URL의 HTML을 가져와서 <title> 텍스트만 추출하는 함수
func fetchTitle(url string) (string, error) {
	// D-2. HTTP GET 요청 보내기
	resp, err := http.Get(url)

	// D-3. 요청 실패 시 (title="", err)로 반환
	if err != nil {
		return "", err
	}

	// D-4. 함수 끝나면 응답 바디 닫기 (리소스 누수 방지)
	defer resp.Body.Close()

	// D-5. HTML을 DOM 트리로 파싱
	doc, err := html.Parse(resp.Body)

	// D-6. 파싱 실패 시 에러 반환
	if err != nil {
		return "", err
	}

	// D-7. 찾은 title을 저장할 변수
	var title string

	// D-8. DFS 함수 변수 선언 (재귀함수 만들려고 미리 선언)
	var dfs func(*html.Node)

	// D-9. DFS 본체: 노드를 보고 <title>이면 텍스트 저장
	dfs = func(n *html.Node) { // D-10. 현재 노드가 <title>이고 자식 텍스트가 있으면 title 저장
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
			return
		}

		// D-11. 자식 노드들을 순회하며 DFS (title을 찾으면 더 안 내려감)
		for c := n.FirstChild; c != nil && title == ""; c = c.NextSibling {
			dfs(c)
		}
	}

	// D-12. 루트부터 DFS 실행
	dfs(doc)

	// D-13. title 못 찾았으면 에러
	if title == "" {
		return "", fmt.Errorf("no title")
	}

	// D-14. title 문자열 반환
	return title, nil
}

// E-1. HTTP 응답을 JSON으로 쓰는 헬퍼 함수
func writeJSON(w http.ResponseWriter, v any) {
	// E-2. 응답 헤더에 JSON임을 표시 (한글 깨짐 방지용 charset 포함)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// E-3. v(구조체 등)를 JSON으로 변환해서 응답 바디에 씀
	_ = json.NewEncoder(w).Encode(v)
}

// F-1. sqlite DB를 열고 테이블을 준비하는 함수
func mustDB() *sql.DB {
	// F-2. sqlite3 드라이버로 logs.db 파일 연결 (없으면 생성)
	db, err := sql.Open("sqlite3", "./logs.db")
	if err != nil {
		panic(err) // F-3. DB 열기 실패는 치명적이라 panic으로 프로그램 종료
	}

	// F-4. logs 테이블 없으면 생성 (간단 로그 저장용)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS logs (ts TEXT, cmd TEXT, ok INTEGER)")
	if err != nil {
		panic(err) // 에러를 실제로 보자
	}
	// F-5. 열린 DB 반환
	return db
}

// G-1. bool을 sqlite 저장용 int로 바꾸는 함수
func boolToInt(b bool) int {
	// G-2. true면 1
	if b {
		return 1
	}
	// G-3. false면 0
	return 0
}
