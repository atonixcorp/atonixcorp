package runnerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type blockingRunner struct {
	release chan struct{}
}

func (runner blockingRunner) Run(ctx context.Context, workflow string) (string, error) {
	select {
	case <-runner.release:
		return "ok", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

type recordingRunner struct {
	done chan struct{}
}

func (runner recordingRunner) Run(_ context.Context, workflow string) (string, error) {
	close(runner.done)
	return "runner output for " + workflow, nil
}

func TestCreatePipelineCanBeRetrieved(t *testing.T) {
	mux := http.NewServeMux()
	runner := blockingRunner{release: make(chan struct{})}
	registerWithRunner(mux, runner)
	t.Cleanup(func() { close(runner.release) })

	payload := []byte(`{"project":"orcastack/platform","ref":"main","trigger":"manual"}`)
	request := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("create pipeline status = %d, want %d", response.Code, http.StatusAccepted)
	}

	var created Pipeline
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode created pipeline: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created pipeline ID is empty")
	}
	if created.Status != "queued" {
		t.Fatalf("created pipeline status = %q, want queued", created.Status)
	}

	request = httptest.NewRequest(http.MethodGet, "/pipelines/"+created.ID, nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("get pipeline status = %d, want %d", response.Code, http.StatusOK)
	}

	var retrieved Pipeline
	if err := json.NewDecoder(response.Body).Decode(&retrieved); err != nil {
		t.Fatalf("decode retrieved pipeline: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Fatalf("retrieved pipeline ID = %q, want %q", retrieved.ID, created.ID)
	}
}

func TestPipelineExecutionCompletesAndPublishesLogs(t *testing.T) {
	mux := http.NewServeMux()
	runner := recordingRunner{done: make(chan struct{})}
	registerWithRunner(mux, runner)

	payload := []byte(`{"project":"orcastack/platform","ref":"main","trigger":"manual","workflow":"api-test"}`)
	request := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("create pipeline status = %d, want %d", response.Code, http.StatusAccepted)
	}

	var created Pipeline
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode created pipeline: %v", err)
	}
	<-runner.done

	request = httptest.NewRequest(http.MethodGet, "/pipelines/"+created.ID, nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	var completed Pipeline
	if err := json.NewDecoder(response.Body).Decode(&completed); err != nil {
		t.Fatalf("decode completed pipeline: %v", err)
	}
	if completed.Status != "success" {
		t.Fatalf("completed pipeline status = %q, want success", completed.Status)
	}
	if len(completed.Jobs) != 1 || completed.Jobs[0].FinishedAt == "" {
		t.Fatalf("completed pipeline job = %#v, want one finished job", completed.Jobs)
	}

	request = httptest.NewRequest(http.MethodGet, completed.Jobs[0].LogEndpoint, nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("get logs status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "runner output for api-test" {
		t.Fatalf("logs = %q, want runner output", response.Body.String())
	}
}

func TestCreatePipelineRejectsUnsupportedWorkflow(t *testing.T) {
	mux := http.NewServeMux()
	registerWithRunner(mux, recordingRunner{done: make(chan struct{})})

	payload := []byte(`{"project":"orcastack/platform","ref":"main","workflow":"arbitrary-shell"}`)
	request := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("create unsupported workflow status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAgentProxyForwardsRegistryAndPredictionWithServerToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer server-secret" {
			t.Errorf("upstream Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/agent/registry":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[]}`))
		case "/v1/agent/predict":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode prediction request: %v", err)
			}
			if payload["model"] != "cluster-autoscaler" {
				t.Errorf("prediction model = %v", payload["model"])
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"mode":"advisory"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	agent := agentProxy{
		baseURL: upstream.URL,
		token:   "server-secret",
		client:  upstream.Client(),
	}

	registry := httptest.NewRecorder()
	agent.handleRegistry(registry, httptest.NewRequest(http.MethodGet, "/agent/registry", nil))
	if registry.Code != http.StatusOK || registry.Body.String() != `{"models":[]}` {
		t.Fatalf("registry response = %d %q", registry.Code, registry.Body.String())
	}

	prediction := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/agent/predict",
		bytes.NewBufferString(`{"model":"cluster-autoscaler","observations":[{"cpu_percent":80}]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	agent.handlePredict(prediction, request)
	if prediction.Code != http.StatusOK || prediction.Body.String() != `{"mode":"advisory"}` {
		t.Fatalf("prediction response = %d %q", prediction.Code, prediction.Body.String())
	}
}

func TestAgentProxyRequiresConfigurationAndJSON(t *testing.T) {
	unconfigured := httptest.NewRecorder()
	(agentProxy{}).handleRegistry(
		unconfigured,
		httptest.NewRequest(http.MethodGet, "/agent/registry", nil),
	)
	if unconfigured.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured proxy status = %d", unconfigured.Code)
	}

	agent := agentProxy{baseURL: "http://127.0.0.1:8000", token: "secret"}
	unsupported := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/predict", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "text/plain")
	agent.handlePredict(unsupported, request)
	if unsupported.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("non-JSON prediction status = %d", unsupported.Code)
	}
}

func TestRunnerRegistersAgentProxyRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer upstream.Close()

	mux := http.NewServeMux()
	registerWithRunnerAndAgent(
		mux,
		recordingRunner{done: make(chan struct{})},
		agentProxy{baseURL: upstream.URL, token: "server-secret", client: upstream.Client()},
	)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/agent/registry", nil))
	if response.Code != http.StatusOK || response.Body.String() != `{"models":[]}` {
		t.Fatalf("runner agent route = %d %q", response.Code, response.Body.String())
	}
}
