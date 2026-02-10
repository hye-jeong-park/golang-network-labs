package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang-network-labs/api/internal/tcpclient"

	"golang.org/x/net/html"
)

// /title 응답 스키마
type TitleResult struct {
	// 추적용 ID
	RequestID string `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	// 사용자 ID
	UserID string `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	// 대상 URL
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
	// 타이틀
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	// 링크 목록
	Links []string `json:"links,omitempty" yaml:"links,omitempty"`
}

// /title: title + 링크 수집 + DB 저장
func (h *Handler) Title(w http.ResponseWriter, r *http.Request) {
	// inFlight 증가
	incInFlight()
	// 종료 시 감소
	defer decInFlight()

	// user_id 추출
	userID := userIDFromReq(r.Header)
	// request_id 생성
	reqID := newRequestID()

	// url 파라미터 읽기
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	// 없으면 거절
	if raw == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}

	// title + links 수집
	title, links, err := fetchTitleAndLinks(raw)
	// 실패면 에러
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// url_results 저장
	resultID, err := insertURLResult(h.db, reqID, userID, raw, title)
	if err != nil {
		http.Error(w, "db insert url_results failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(links) > 0 {
		if err := insertURLLinks(h.db, resultID, links); err != nil {
			http.Error(w, "db insert url_links failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// url_links 저장
	if resultID > 0 && len(links) > 0 {
		_ = insertURLLinks(h.db, resultID, links)
	}

	// 응답 구성
	resp := TitleResult{
		RequestID: reqID,
		UserID:    userID,
		URL:       raw,
		Title:     title,
	}

	// 링크가 있으면 포함
	if len(links) > 0 {
		resp.Links = links
	}

	// 응답 반환(JSON/YAML)
	writeResponse(w, r, resp)

	// tcpclient import unused 방지(실수 방지용)
	_ = tcpclient.Req{}
}

// url_results insert
func insertURLResult(db *sql.DB, reqID, userID, u, title string) (int64, error) {
	// insert 수행
	res, err := db.Exec(
		`INSERT INTO url_results(ts, request_id, user_id, url, title) VALUES (?,?,?,?,?)`,
		now(), reqID, userID, u, title,
	)
	// 실패 반환
	if err != nil {
		return 0, err
	}
	// id 반환
	return res.LastInsertId()
}

// url_links insert
func insertURLLinks(db *sql.DB, resultID int64, links []string) error {
	// 링크 단건씩 저장
	for _, l := range links {
		_, err := db.Exec(`INSERT INTO url_links(result_id, link_url) VALUES (?,?)`, resultID, l)
		if err != nil {
			return err
		}
	}
	return nil
}

// title + links 추출
func fetchTitleAndLinks(raw string) (string, []string, error) {
	// URL 요청
	resp, err := http.Get(raw)
	if err != nil {
		return "", nil, err
	}
	// 바디 닫기
	defer resp.Body.Close()

	// DOM 파싱
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, err
	}

	// base url 파싱
	baseURL, _ := url.Parse(raw)

	// title 저장
	var title string
	// 중복 제거용 map
	linksMap := make(map[string]bool)

	// DFS 탐색
	var dfs func(*html.Node)
	dfs = func(n *html.Node) {
		// 엘리먼트 노드만 처리
		if n.Type == html.ElementNode {
			// title 추출
			if n.Data == "title" && n.FirstChild != nil && title == "" {
				title = n.FirstChild.Data
			}
			// a 태그 href 수집
			if n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						href := strings.TrimSpace(a.Val)
						if href == "" {
							continue
						}
						abs := normalizeLink(baseURL, href)
						if abs != "" {
							linksMap[abs] = true
						}
					}
				}
			}
		}

		// 자식 순회
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			dfs(c)
		}
	}

	// 루트부터 실행
	dfs(doc)

	// 타이틀 없으면 에러
	if title == "" {
		return "", nil, fmt.Errorf("no title")
	}

	// map → slice
	links := make([]string, 0, len(linksMap))
	for k := range linksMap {
		links = append(links, k)
	}

	// 결과 반환
	return title, links, nil
}

// 링크 정규화
func normalizeLink(base *url.URL, href string) string {
	// href 파싱
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	// 상대경로면 base로 보정
	if base != nil {
		u = base.ResolveReference(u)
	}
	// 위험 스킴 제외
	if href == "#" || strings.HasPrefix(href, "#") {
		return ""
	}
	if u.Scheme == "javascript" || u.Scheme == "mailto" || u.Scheme == "tel" {
		return ""
	}
	return u.String()
}
