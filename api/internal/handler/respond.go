package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAML 응답 여부 결정
func wantYAML(r *http.Request) bool {
	// query로 강제
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "yaml") {
		return true
	}
	// Accept 기반
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/x-yaml") || strings.Contains(accept, "text/yaml")
}

// 공통 응답 작성(JSON/YAML)
func writeResponse(w http.ResponseWriter, r *http.Request, v any) {
	// YAML이면 YAML로 반환
	if wantYAML(r) {
		w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
		b, _ := yaml.Marshal(v)
		_, _ = w.Write(b)
		return
	}

	// 기본은 JSON
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
