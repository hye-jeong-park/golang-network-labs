package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/html"
)

type Req struct {
	Cmd string `json:"cmd" form:"cmd"`
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

// 프로그램 시작 (API 서버)
func main() {
	// sqlite DB 열기 및 초기화
	// logs.db 파일 열기 (없으면 생성)
	// 로그 테이블 준비
	db := mustDB()
	defer db.Close()

	// /run API 등록
	// /run 경로로 요청 오면 실행될 핸들러 등록
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		var cmd string

		// 1) Method별 파싱
		switch r.Method {
		case http.MethodGet:
			// 쿼리 파라미터 cmd 읽기
			// ex: /run?cmd=ipconfig
			cmd = r.URL.Query().Get("cmd")

		case http.MethodPost:
			// body 크기 제한(1MB)
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

			ct := r.Header.Get("Content-Type")

			// (a) JSON
			if strings.HasPrefix(ct, "application/json") {
				var req Req
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "invalid json", 400)
					return
				}
				cmd = req.Cmd

			} else if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
				// (b) form-urlencoded
				if err := r.ParseForm(); err != nil {
					http.Error(w, "invalid form", 400)
					return
				}
				cmd = r.Form.Get("cmd")

			} else if strings.HasPrefix(ct, "multipart/form-data") {
				// (c) multipart
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					http.Error(w, "invalid multipart form", 400)
					return
				}
				cmd = r.FormValue("cmd")

			} else {
				http.Error(w, "unsupported content-type", 415)
				return
			}

		default:
			http.Error(w, "method not allowed", 405)
			return
		}

		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			http.Error(w, "cmd required", 400)
			return
		}

		// 2) TCP 호출 - TCP 응답을 받을 채널 생성
		// 버퍼 1칸 → 고루틴 블로킹 방지
		ch := make(chan Res, 1)

		//TCP 호출을 고루틴으로 실행
		//API 서버 스레드를 막지 않기 위해 TCP 통신을 별도 실행
		go func() { ch <- callTCP(cmd) }()

		// TCP 응답 대기
		res := <-ch

		// 3) 실행 로그 DB 저장 - 언제, 어떤 명령을, 성공했는지 기록
		_, _ = db.Exec(
			"INSERT INTO logs(ts, cmd, ok) VALUES (?,?,?)",
			time.Now(), cmd, boolToInt(res.Ok),
		)

		// 4) 응답 -HTTP JSON 응답 반환
		writeJSON(w, res)
	})

	// /title API 등록 (HTML 파싱 실습용 API)
	http.HandleFunc("/title", func(w http.ResponseWriter, r *http.Request) {
		// url 파라미터 읽기
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "url required", 400)
			return
		}

		// HTTP GET + HTML 파싱
		title, err := fetchTitle(url) //외부 웹 요청, <title> 태그 추출
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// 결과 반환
		writeJSON(w, TitleResult{URL: url, Title: title})
	})

	fmt.Println("api :8080")

	// HTTP 서버 실행 및 대기 (8080 포트에서 HTTP 요청 대기, 여기서 메인 고루틴은 blocking)
	_ = http.ListenAndServe(":8080", nil)
}

// TCP 서버(127.0.0.1:9000)에 명령(cmd)을 보내고 Res를 받아오는 함수
func callTCP(cmd string) Res {
	host := os.Getenv("TCP_HOST")
	port := os.Getenv("TCP_PORT")
	if host == "" {
		host = "tcp"
	}
	if port == "" {
		port = "9000"
	}
	addr := host + ":" + port
	// TCP 연결 시도 (클라이언트 역할)
	conn, err := net.Dial("tcp", addr)

	// 연결 실패하면 실패 응답 구조체를 만들어 바로 반환
	if err != nil {
		return Res{Ok: false, Error: err.Error()}
	}
	defer conn.Close() // 함수 끝나면 TCP 연결 자동 닫기

	// 요청(Req{Cmd: cmd})을 JSON으로 변환
	b, _ := json.Marshal(Req{Cmd: cmd})

	// JSON 뒤에 '\n' 붙여서 한 줄 프로토콜로 전송
	conn.Write(append(b, '\n'))

	// 응답을 한 줄 단위로 읽기 위한 Reader 생성
	br := bufio.NewReader(conn)
	// '\n'까지 읽어서 응답 한 줄 수신 (서버가 응답할 때까지 대기)
	line, err := br.ReadBytes('\n')

	// 읽기 실패 시 실패 응답 반환
	if err != nil {
		return Res{Ok: false, Error: err.Error()}
	}
	var res Res // C-10. 응답 JSON을 Res 구조체로 파싱
	_ = json.Unmarshal(line, &res)
	return res
}

// URL의 HTML을 가져와서 <title> 텍스트만 추출하는 함수
func fetchTitle(url string) (string, error) {
	// HTTP GET 요청 보내기
	resp, err := http.Get(url)

	//요청 실패 시 (title="", err)로 반환
	if err != nil {
		return "", err
	}

	// 함수 끝나면 응답 바디 닫기 (리소스 누수 방지)
	defer resp.Body.Close()

	// HTML을 DOM 트리로 파싱
	doc, err := html.Parse(resp.Body)

	// 파싱 실패 시 에러 반환
	if err != nil {
		return "", err
	}

	// 찾은 title을 저장할 변수
	var title string

	// DFS 함수 변수 선언 (재귀함수 만들려고 미리 선언)
	var dfs func(*html.Node)

	// DFS 본체: 노드를 보고 <title>이면 텍스트 저장
	dfs = func(n *html.Node) { // 현재 노드가 <title>이고 자식 텍스트가 있으면 title 저장
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
			return
		}

		// 자식 노드들을 순회하며 DFS (title을 찾으면 더 안 내려감)
		for c := n.FirstChild; c != nil && title == ""; c = c.NextSibling {
			dfs(c)
		}
	}

	// 루트부터 DFS 실행
	dfs(doc)

	// title 못 찾았으면 에러
	if title == "" {
		return "", fmt.Errorf("no title")
	}

	// title 문자열 반환
	return title, nil
}

// HTTP 응답을 JSON으로 쓰는 헬퍼 함수
func writeJSON(w http.ResponseWriter, v any) {
	// 응답 헤더에 JSON임을 표시 (한글 깨짐 방지용 charset 포함)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	//v(구조체 등)를 JSON으로 변환해서 응답 바디에 씀
	_ = json.NewEncoder(w).Encode(v)
}

// dns 만드는 함수
func buildDSNFromEnv() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")

	if host == "" || port == "" || name == "" || user == "" || pass == "" {
		panic("DB environment variables are not fully set")
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		user, pass, host, port, name,
	)

	return dsn
}

// MariaDB를 열고 테이블을 준비하는 함수
func mustDB() *sql.DB {
	dsn := buildDSNFromEnv()

	db, err := sql.Open("mysql", dsn)
	if err != nil { // DB 열기 실패는 치명적이라 panic으로 프로그램 종료
		panic(err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// MariaDB 준비까지 재시도
	var lastErr error
	for i := 1; i <= 30; i++ { // 최대 30초 대기(1초 간격)
		if err := db.Ping(); err == nil {
			lastErr = nil
			break
		} else {
			lastErr = err
			time.Sleep(1 * time.Second)
		}
	}
	if lastErr != nil {
		panic(lastErr)
	}

	// MariaDB용 테이블 스키마
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS logs (
            id BIGINT AUTO_INCREMENT PRIMARY KEY,
            ts DATETIME NOT NULL,
            cmd VARCHAR(255) NOT NULL,
            ok TINYINT NOT NULL
        );
    `)
	if err != nil {
		panic(err)
	}

	return db
}

// bool을 sqlite 저장용 int로 바꾸는 함수
func boolToInt(b bool) int {
	// true면 1
	if b {
		return 1
	}
	// false면 0
	return 0
}
