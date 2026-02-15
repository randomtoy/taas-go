package openrouter_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/randomtoy/taas-go/internal/adapters/llm/openrouter"
	"github.com/randomtoy/taas-go/internal/ports"
)

func testInput() ports.InterpretInput {
	return ports.InterpretInput{
		DeckID:   "major_arcana",
		Spread:   "three_card",
		Question: "What lies ahead?",
		Cards: []ports.CardInput{
			{Name: "The Fool", Position: 1, Orientation: "upright", Keywords: []string{"beginnings"}, Short: "A fresh start."},
			{Name: "The Magician", Position: 2, Orientation: "reversed", Keywords: []string{"willpower"}, Short: "Personal power."},
			{Name: "The Star", Position: 3, Orientation: "upright", Keywords: []string{"hope"}, Short: "Renewed faith."},
		},
	}
}

func TestClient_Interpret_Success(t *testing.T) {
	llmResp := ports.InterpretOutput{
		Text:       "A thoughtful interpretation.",
		Style:      "neutral",
		Disclaimer: "For reflection/entertainment; not medical/legal/financial advice.",
	}
	llmJSON, _ := json.Marshal(llmResp)

	var gotReq map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		// Verify headers.
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("bad content-type: %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": string(llmJSON)}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := openrouter.NewClient(
		srv.Client(),
		"test-key",
		srv.URL,
		"test-model",
		slog.Default(),
	)

	out, err := client.Interpret(context.Background(), testInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Text != "A thoughtful interpretation." {
		t.Errorf("unexpected text: %s", out.Text)
	}
	if out.Style != "neutral" {
		t.Errorf("unexpected style: %s", out.Style)
	}

	// Verify the request body contains our model.
	if gotReq["model"] != "test-model" {
		t.Errorf("request model: %v", gotReq["model"])
	}
}

func TestClient_Interpret_BadJSON_Retry_Success(t *testing.T) {
	llmResp := ports.InterpretOutput{
		Text:       "Retried interpretation.",
		Style:      "neutral",
		Disclaimer: "For reflection/entertainment; not medical/legal/financial advice.",
	}
	llmJSON, _ := json.Marshal(llmResp)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		var content string
		if callCount == 1 {
			content = "this is not json at all"
		} else {
			content = string(llmJSON)
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": content}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := openrouter.NewClient(srv.Client(), "key", srv.URL, "model", slog.Default())

	out, err := client.Interpret(context.Background(), testInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (original + retry), got %d", callCount)
	}
	if out.Text != "Retried interpretation." {
		t.Errorf("unexpected text: %s", out.Text)
	}
}

func TestClient_Interpret_BadJSON_Retry_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "still not json"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := openrouter.NewClient(srv.Client(), "key", srv.URL, "model", slog.Default())

	_, err := client.Interpret(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for double-bad JSON, got nil")
	}
}

func TestClient_Interpret_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	client := openrouter.NewClient(srv.Client(), "key", srv.URL, "model", slog.Default())

	_, err := client.Interpret(context.Background(), testInput())
	if err == nil {
		t.Fatal("expected error for upstream 500, got nil")
	}
}
