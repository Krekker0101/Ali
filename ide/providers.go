package ide

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
)

type ProviderRegistry struct {
	local *LocalProvider
	cloud *CloudProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		local: NewLocalProvider(),
		cloud: &CloudProvider{client: &http.Client{Timeout: 2 * time.Minute}},
	}
}

func (r *ProviderRegistry) Get(id string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "", "local":
		return r.local, nil
	case "cloud":
		return r.cloud, nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", id)
	}
}

type LocalProvider struct {
	client *api.Client
	pullMu sync.Mutex
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{
		client: api.NewClient(envconfig.ConnectableHost(), http.DefaultClient),
	}
}

func (p *LocalProvider) ID() string {
	return "local"
}

func (p *LocalProvider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return recommendedLocalModels(), nil
	}

	models := make([]string, 0, len(resp.Models))
	for _, model := range resp.Models {
		if model.Model != "" {
			models = append(models, model.Model)
			continue
		}
		models = append(models, model.Name)
	}
	return mergeModels(models, recommendedLocalModels()), nil
}

func (p *LocalProvider) Complete(ctx context.Context, req CompletionRequest, settings AISettings) (CompletionResponse, error) {
	model := firstNonEmpty(req.Model, settings.Model)
	if model == "" {
		return CompletionResponse{}, fmt.Errorf("model is required")
	}

	if err := p.ensureModel(ctx, model); err != nil {
		return CompletionResponse{}, err
	}

	stream := false
	genReq := &api.GenerateRequest{
		Model:  model,
		System: req.System,
		Prompt: req.Prompt,
		Stream: &stream,
		Options: map[string]any{
			"temperature": firstPositiveFloat(req.Temperature, settings.Temperature, 0.2),
			"num_predict": firstPositiveInt(req.MaxTokens, settings.MaxTokens, 2048),
		},
	}

	var out strings.Builder
	if err := p.client.Generate(ctx, genReq, func(resp api.GenerateResponse) error {
		out.WriteString(resp.Response)
		return nil
	}); err != nil {
		return CompletionResponse{}, err
	}

	return CompletionResponse{Text: out.String(), Provider: p.ID(), Model: model}, nil
}

func (p *LocalProvider) ensureModel(ctx context.Context, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}

	installed, err := p.hasModel(ctx, model)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	p.pullMu.Lock()
	defer p.pullMu.Unlock()

	installed, err = p.hasModel(ctx, model)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	return p.client.Pull(ctx, &api.PullRequest{Model: model}, func(resp api.ProgressResponse) error {
		return nil
	})
}

func (p *LocalProvider) hasModel(ctx context.Context, model string) (bool, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return false, err
	}
	for _, existing := range resp.Models {
		if sameModelName(existing.Model, model) || sameModelName(existing.Name, model) {
			return true, nil
		}
	}
	return false, nil
}

type CloudProvider struct {
	client *http.Client
}

func (p *CloudProvider) ID() string {
	return "cloud"
}

func (p *CloudProvider) ListModels(ctx context.Context) ([]string, error) {
	return p.ListModelsWithSettings(ctx, AISettings{})
}

func (p *CloudProvider) ListModelsWithSettings(ctx context.Context, settings AISettings) ([]string, error) {
	baseURL := defaultCloudBaseURL(settings.CloudBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if key := firstNonEmpty(settings.CloudAPIKey, envconfig.Var("OPENAI_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, readHTTPError(resp)
	}

	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(body.Data))
	for _, model := range body.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (p *CloudProvider) Complete(ctx context.Context, req CompletionRequest, settings AISettings) (CompletionResponse, error) {
	model := firstNonEmpty(req.Model, settings.Model)
	if model == "" {
		return CompletionResponse{}, fmt.Errorf("model is required")
	}

	apiKey := firstNonEmpty(settings.CloudAPIKey, envconfig.Var("OPENAI_API_KEY"))
	if apiKey == "" {
		return CompletionResponse{}, fmt.Errorf("cloud API key is required")
	}

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.Prompt},
		},
		"temperature": firstPositiveFloat(req.Temperature, settings.Temperature, 0.2),
		"max_tokens":  firstPositiveInt(req.MaxTokens, settings.MaxTokens, 2048),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return CompletionResponse{}, err
	}

	baseURL := defaultCloudBaseURL(settings.CloudBaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return CompletionResponse{}, readHTTPError(resp)
	}

	var body struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return CompletionResponse{}, err
	}
	if len(body.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("cloud provider returned no choices")
	}

	return CompletionResponse{Text: body.Choices[0].Message.Content, Provider: p.ID(), Model: model}, nil
}

func defaultCloudBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return strings.TrimRight(baseURL, "/")
}

func recommendedLocalModels() []string {
	return []string{
		"qwen2.5-coder:1.5b",
		"qwen2.5-coder:7b",
		"llama3.2:3b",
		"mistral:7b",
		"codellama:7b",
	}
}

func mergeModels(groups ...[]string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, group := range groups {
		for _, model := range group {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			key := strings.ToLower(model)
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, model)
		}
	}
	return merged
}

func sameModelName(a string, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if !strings.Contains(a, ":") {
		a += ":latest"
	}
	if !strings.Contains(b, ":") {
		b += ":latest"
	}
	return a == b
}

func readHTTPError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if len(data) == 0 {
		return fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
