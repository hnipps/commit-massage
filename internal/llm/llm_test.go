package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChat(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    string
		wantResult string
	}{
		{
			name: "success returns message content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var req ChatRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if req.Model != "test-model" {
					t.Errorf("expected model test-model, got %s", req.Model)
				}
				if req.Stream {
					t.Error("expected stream=false")
				}

				resp := ChatResponse{
					Choices: []struct {
						Message Message `json:"message"`
					}{
						{Message: Message{Role: "assistant", Content: "feat: add login endpoint"}},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantResult: "feat: add login endpoint",
		},
		{
			name: "error status code returns error with status and body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			},
			wantErr: "500",
		},
		{
			name: "empty choices returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				resp := ChatResponse{Choices: nil}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantErr: "no choices",
		},
		{
			name: "trims whitespace from response content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				resp := ChatResponse{
					Choices: []struct {
						Message Message `json:"message"`
					}{
						{Message: Message{Role: "assistant", Content: "  fix: trim spaces  \n"}},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			},
			wantResult: "fix: trim spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewClient(srv.URL)
			msgs := []Message{{Role: "user", Content: "hello"}}
			result, err := client.Chat(context.Background(), "test-model", msgs)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("expected %q, got %q", tt.wantResult, result)
			}
		})
	}
}

func TestTrailingSlashInBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "//") {
			t.Errorf("URL contains double slash: %s", r.URL.Path)
		}
		resp := ChatResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{
				{Message: Message{Role: "assistant", Content: "ok"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Create client with trailing slash
	client := NewClient(srv.URL + "/")
	_, err := client.Chat(context.Background(), "m", []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
