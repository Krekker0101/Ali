package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ollama/ollama/api"
)

func TestLocalAgentPullsMissingModelBeforeGenerate(t *testing.T) {
	installed := false
	pullCalled := false
	generateCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := api.ListResponse{}
			if installed {
				resp.Models = []api.ListModelResponse{{Model: "qwen2.5-coder:1.5b"}}
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/api/pull":
			pullCalled = true
			installed = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"success"}` + "\n"))
		case "/api/generate":
			generateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"response":"ok","done":true}` + "\n"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	agent := localAgentForTest(t, server)
	result, err := agent.callOllamaAPI(context.Background(), "qwen2.5-coder:1.5b", "hello", DefaultAgentConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Fatalf("result = %q, want ok", result)
	}
	if !pullCalled {
		t.Fatal("expected missing local model to be pulled")
	}
	if !generateCalled {
		t.Fatal("expected generation after pull")
	}
}

func TestLocalAgentDoesNotPullInstalledModel(t *testing.T) {
	pullCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(api.ListResponse{
				Models: []api.ListModelResponse{{Model: "llama3.2:3b"}},
			})
		case "/api/pull":
			pullCalled = true
			w.WriteHeader(http.StatusOK)
		case "/api/generate":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"response":"ok","done":true}` + "\n"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	agent := localAgentForTest(t, server)
	if _, err := agent.callOllamaAPI(context.Background(), "llama3.2:3b", "hello", DefaultAgentConfig()); err != nil {
		t.Fatal(err)
	}
	if pullCalled {
		t.Fatal("did not expect pull for an installed model")
	}
}

func localAgentForTest(t *testing.T, server *httptest.Server) *LocalAgentImpl {
	t.Helper()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return NewLocalAgent(api.NewClient(base, server.Client()), 1)
}
