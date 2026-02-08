package handler

import (
	"net/http"
)

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	//200 반환
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
