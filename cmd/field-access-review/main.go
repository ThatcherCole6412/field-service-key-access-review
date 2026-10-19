package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/infrai-examples/field-access-review/internal/fieldaudit"
	"github.com/infrai-examples/field-access-review/internal/infrai"
)

type server struct {
	client *infrai.Client
	now    func() time.Time
}

type auditResult struct {
	Review fieldaudit.Review `json:"review"`
	PDF    infrai.PDF        `json:"pdf"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}

	s := &server{client: infrai.New(key), now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /audit", s.audit)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	log.Printf("field access review listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func (s *server) audit(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	keys, err := s.client.ListKeys(ctx)
	if err != nil {
		writeError(w, err)
		return
	}
	review := fieldaudit.Build(keys, s.now())
	pdf, err := s.client.GenerateReview(ctx, fieldaudit.Markdown(review))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, auditResult{Review: review, PDF: pdf})
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var apiErr *infrai.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		status = apiErr.HTTPStatus
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
