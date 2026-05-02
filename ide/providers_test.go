package ide

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ollama/ollama/api"
)

func TestLocalProviderListsInstalledAndRecommendedModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(api.ListResponse{
			Models: []api.ListModelResponse{{Model: "custom:latest"}},
		})
	}))
	defer server.Close()

	provider := localProviderForTest(t, server)
	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !containsModel(models, "custom:latest") {
		t.Fatalf("models = %v, want installed model", models)
	}
	if !containsModel(models, "qwen2.5-coder:1.5b") {
		t.Fatalf("models = %v, want recommended model", models)
	}
}

func TestLocalProviderPullsMissingModelBeforeGenerate(t *testing.T) {
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

	provider := localProviderForTest(t, server)
	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Model:  "qwen2.5-coder:1.5b",
		Prompt: "hello",
	}, AISettings{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "ok" {
		t.Fatalf("response text = %q, want ok", resp.Text)
	}
	if !pullCalled {
		t.Fatal("expected missing local model to be pulled")
	}
	if !generateCalled {
		t.Fatal("expected generation after pull")
	}
}

func TestLocalProviderDoesNotPullInstalledModel(t *testing.T) {
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

	provider := localProviderForTest(t, server)
	if _, err := provider.Complete(context.Background(), CompletionRequest{
		Model:  "llama3.2:3b",
		Prompt: "hello",
	}, AISettings{}); err != nil {
		t.Fatal(err)
	}
	if pullCalled {
		t.Fatal("did not expect pull for an installed model")
	}
}

func localProviderForTest(t *testing.T, server *httptest.Server) *LocalProvider {
	t.Helper()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &LocalProvider{client: api.NewClient(base, server.Client())}
}

func containsModel(models []string, want string) bool {
	for _, model := range models {
		if sameModelName(model, want) {
			return true
		}
	}
	return false
}
