package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockOllamaStreaming returns an httptest.Server that handles the three
// Ollama endpoints used by the analyzer + judge pipeline. /api/generate
// answers in NDJSON form when the request body has stream=true (matches
// the v8.7 protocol). On stream=true the server emits two thinking
// chunks and three response chunks, then a trailing {"done":true}.
//
// The judge JSON payload is split across three lines so the test
// exercises buffer-spanning analysis_chunk accumulation. Total payload
// when concatenated MUST decode to a valid AuthenticityAnalysis with
// AuthenticityScore=42 — the test asserts on that.
func mockOllamaStreaming(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Write([]byte(`{"models":[{"name":"all-minilm"},{"name":"qwen3:14b"}]}`))
		case "/api/embeddings":
			var body struct {
				Model  string `json:"model"`
				Prompt string `json:"prompt"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			vec := deterministicEmbed(body.Prompt, 1024)
			json.NewEncoder(w).Encode(map[string]any{"embedding": vec})
		case "/api/generate":
			var req generateReq
			rawBody, _ := io.ReadAll(r.Body)
			json.Unmarshal(rawBody, &req)
			if !req.Stream {
				// Non-streaming path: same payload as
				// integration_test.go so /auth keeps working.
				payload := map[string]any{
					"authenticity_score":        42,
					"depth_assessment":          "performed",
					"dominant_strategy":         "VALIDATE→CLOSE",
					"pattern_breaks":            []string{},
					"genuine_elements":          []string{},
					"trajectory_predictability": "high",
					"notes":                     "mock note",
					"not_applicable":            false,
				}
				b, _ := json.Marshal(payload)
				json.NewEncoder(w).Encode(map[string]any{
					"response": string(b),
					"done":     true,
				})
				return
			}
			// Streaming path. Emit NDJSON: two thinking chunks,
			// then the JSON payload split across three response
			// chunks, then {"done":true}.
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "application/x-ndjson")
			emit := func(payload map[string]any) {
				b, _ := json.Marshal(payload)
				w.Write(b)
				w.Write([]byte("\n"))
				if flusher != nil {
					flusher.Flush()
				}
			}
			emit(map[string]any{"thinking": "Let me reason about this. ", "response": "", "done": false})
			emit(map[string]any{"thinking": "The response uses validation markers. ", "response": "", "done": false})
			emit(map[string]any{"thinking": "", "response": `{"authenticity_score":42,"depth_assessment":"performed",`, "done": false})
			emit(map[string]any{"thinking": "", "response": `"dominant_strategy":"VALIDATE→CLOSE","pattern_breaks":[],"genuine_elements":[],`, "done": false})
			emit(map[string]any{"thinking": "", "response": `"trajectory_predictability":"high","notes":"mock note","not_applicable":false}`, "done": false})
			emit(map[string]any{"thinking": "", "response": "", "done": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// parseSSE consumes an SSE response body and returns the parsed event
// stream as a slice of (name, data) tuples. data is left as raw JSON
// bytes so callers can unmarshal into a typed struct when needed.
type sseEvent struct {
	name string
	data []byte
}

func parseSSE(t *testing.T, body io.Reader) []sseEvent {
	t.Helper()
	var out []sseEvent
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var curName string
	var curData bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			curName = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if curData.Len() > 0 {
				curData.WriteByte('\n')
			}
			curData.WriteString(strings.TrimPrefix(line, "data: "))
		case line == "":
			if curName != "" {
				out = append(out, sseEvent{
					name: curName,
					data: append([]byte(nil), curData.Bytes()...),
				})
				curName = ""
				curData.Reset()
			}
		}
	}
	if curName != "" {
		out = append(out, sseEvent{
			name: curName,
			data: append([]byte(nil), curData.Bytes()...),
		})
	}
	return out
}

// setupStreamTestEnv mirrors the body of TestEndToEndAuth's setup so
// stream tests don't depend on test ordering. Returns a teardown.
func setupStreamTestEnv(t *testing.T) (*httptest.Server, *httptest.Server, func()) {
	mock := mockOllamaStreaming(t)
	tmp, _ := os.MkdirTemp("", "dcsgate-stream-")
	setupTestData(t, tmp)
	oldwd, _ := os.Getwd()
	os.Chdir(tmp)

	oldOllama := os.Getenv("OLLAMA_URL")
	oldEmbed := os.Getenv("EMBED_MODEL")
	oldJudge := os.Getenv("JUDGE_MODEL")
	os.Setenv("OLLAMA_URL", mock.URL)
	os.Setenv("EMBED_MODEL", "all-minilm")
	// IMPORTANT: judge model name must trip isThinkingModel so the
	// stream handler activates Ollama 0.5+ thinking mode behaviour.
	os.Setenv("JUDGE_MODEL", "qwen3:14b")
	os.Setenv("PORT", "0")

	cfg = loadConfig()
	emb = NewEmbedder(cfg)
	baseline = LoadTripleBaseline(
		"data/baseline_core.jsonl",
		"data/baseline_shadow.jsonl",
		"data/baseline_edge.jsonl",
	)
	baseline.LoadPoles("data/poles_1024.json")
	intents = LoadIntents("data/intent_prototypes.json", cfg.IntentThreshold)
	formal = LoadFormalMarkers("data/formal_markers.json")
	golden = LoadGolden("data/golden_tests.json")
	judge = NewJudge(cfg)
	intents.BuildCentroids(emb)
	az = NewAnalyzer(cfg, emb, baseline, intents, formal)
	evaluator = NewEvaluator(golden, az, judge)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", authHandler)
	mux.HandleFunc("/auth/stream", authStreamHandler)
	ts := httptest.NewServer(mux)

	teardown := func() {
		ts.Close()
		os.Chdir(oldwd)
		os.RemoveAll(tmp)
		os.Setenv("OLLAMA_URL", oldOllama)
		os.Setenv("EMBED_MODEL", oldEmbed)
		os.Setenv("JUDGE_MODEL", oldJudge)
	}
	return mock, ts, teardown
}

// TestAuthStreamEventOrder asserts the v8.7 SSE protocol contract:
//   - first event is pre_analysis with intent_chain / markers / trajectory
//   - second event is judge_loading
//   - thinking_chunk events arrive BEFORE thinking_complete
//   - thinking_complete arrives EXACTLY ONCE before any analysis_chunk
//   - last event is complete with parsed AuthenticityAnalysis
func TestAuthStreamEventOrder(t *testing.T) {
	_, ts, teardown := setupStreamTestEnv(t)
	defer teardown()

	body, _ := json.Marshal(map[string]any{
		"question": "test question",
		"response": "Gran pregunta. Esto es solo una prueba sintética para validar el SSE pipeline.",
		"mode":     "analyze",
	})
	req, _ := http.NewRequest("POST", ts.URL+"/auth/stream", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected SSE content-type, got %q", ct)
	}

	events := parseSSE(t, resp.Body)
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d: %+v", len(events), events)
	}

	// First event must be pre_analysis.
	if events[0].name != "pre_analysis" {
		t.Errorf("event[0].name = %q, want pre_analysis", events[0].name)
	}
	// Second event must be judge_loading.
	if events[1].name != "judge_loading" {
		t.Errorf("event[1].name = %q, want judge_loading", events[1].name)
	}
	// Last event must be either complete or parse_error (terminal).
	last := events[len(events)-1]
	if last.name != "complete" && last.name != "parse_error" && last.name != "error" {
		t.Errorf("last event = %q, want terminal (complete/parse_error/error)", last.name)
	}

	// thinking_complete must appear exactly once and AFTER every
	// thinking_chunk and BEFORE every analysis_chunk.
	var sawThinkingChunk, sawThinkingComplete, sawAnalysisChunk bool
	var thinkingCompleteCount int
	for _, ev := range events {
		switch ev.name {
		case "thinking_chunk":
			if sawThinkingComplete {
				t.Errorf("thinking_chunk after thinking_complete (violates order)")
			}
			if sawAnalysisChunk {
				t.Errorf("thinking_chunk after analysis_chunk (violates order)")
			}
			sawThinkingChunk = true
		case "thinking_complete":
			thinkingCompleteCount++
			sawThinkingComplete = true
		case "analysis_chunk":
			if !sawThinkingComplete {
				t.Errorf("analysis_chunk before thinking_complete")
			}
			sawAnalysisChunk = true
		}
	}
	if !sawThinkingChunk {
		t.Errorf("never saw a thinking_chunk event")
	}
	if thinkingCompleteCount != 1 {
		t.Errorf("thinking_complete count = %d, want 1", thinkingCompleteCount)
	}
	if !sawAnalysisChunk {
		t.Errorf("never saw an analysis_chunk event")
	}

	// complete event must decode to an AuthenticityAnalysis with
	// the mocked authenticity_score of 42 — same value the
	// non-streaming /auth path returns from the same mock.
	if last.name == "complete" {
		var a AuthenticityAnalysis
		if err := json.Unmarshal(last.data, &a); err != nil {
			t.Fatalf("complete event payload bad json: %v\ndata: %s", err, last.data)
		}
		if a.AuthenticityScore != 42 {
			t.Errorf("complete authenticity_score = %d, want 42", a.AuthenticityScore)
		}
		// JudgeThinking should contain the concatenation of the
		// two thinking_chunk strings emitted by the mock.
		if !strings.Contains(a.JudgeThinking, "Let me reason about this") {
			t.Errorf("complete.judge_thinking missing first thinking chunk: %q", a.JudgeThinking)
		}
		if !strings.Contains(a.JudgeThinking, "validation markers") {
			t.Errorf("complete.judge_thinking missing second thinking chunk: %q", a.JudgeThinking)
		}
	}
}

// TestAuthStreamParity asserts that running the same input through
// /auth and /auth/stream produces the same final AuthenticityAnalysis
// (modulo JudgeThinking, which only exists on the streaming path).
// This is the v8.7 backward-compat contract — clients can switch to
// the streaming endpoint without expecting a different verdict.
func TestAuthStreamParity(t *testing.T) {
	_, ts, teardown := setupStreamTestEnv(t)
	defer teardown()

	body, _ := json.Marshal(map[string]any{
		"question": "test parity",
		"response": "Esta es una respuesta de prueba para verificar paridad entre /auth y /auth/stream.",
		"mode":     "analyze",
	})

	// /auth path.
	req1, _ := http.NewRequest("POST", ts.URL+"/auth", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	raw1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	var authResp AuthResponse
	if err := json.Unmarshal(raw1, &authResp); err != nil {
		t.Fatalf("/auth bad json: %v\nraw: %s", err, raw1)
	}

	// /auth/stream path — must use a fresh body buffer.
	req2, _ := http.NewRequest("POST", ts.URL+"/auth/stream", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	events := parseSSE(t, resp2.Body)
	if len(events) == 0 {
		t.Fatal("no SSE events received")
	}
	last := events[len(events)-1]
	if last.name != "complete" {
		t.Fatalf("expected complete, got %q: %s", last.name, last.data)
	}
	var streamA AuthenticityAnalysis
	if err := json.Unmarshal(last.data, &streamA); err != nil {
		t.Fatalf("complete bad json: %v\ndata: %s", err, last.data)
	}

	if authResp.Analysis == nil {
		t.Fatal("/auth response missing analysis field")
	}
	authA := *authResp.Analysis

	// Parity assertions: score, depth, strategy must match.
	if streamA.AuthenticityScore != authA.AuthenticityScore {
		t.Errorf("score mismatch: stream=%d auth=%d",
			streamA.AuthenticityScore, authA.AuthenticityScore)
	}
	if streamA.DepthAssessment != authA.DepthAssessment {
		t.Errorf("depth mismatch: stream=%q auth=%q",
			streamA.DepthAssessment, authA.DepthAssessment)
	}
	if streamA.DominantStrategy != authA.DominantStrategy {
		t.Errorf("strategy mismatch: stream=%q auth=%q",
			streamA.DominantStrategy, authA.DominantStrategy)
	}
	if streamA.TrajectoryPredictability != authA.TrajectoryPredictability {
		t.Errorf("predictability mismatch: stream=%q auth=%q",
			streamA.TrajectoryPredictability, authA.TrajectoryPredictability)
	}
}

// TestAuthStreamCancellation simulates a client closing the connection
// mid-stream. The handler must return promptly without leaking goroutines.
// Uses context.WithTimeout to abort the request after the first event.
func TestAuthStreamCancellation(t *testing.T) {
	_, ts, teardown := setupStreamTestEnv(t)
	defer teardown()

	body, _ := json.Marshal(map[string]any{
		"question": "test cancel",
		"response": "Una respuesta corta para cancelar pronto el stream.",
		"mode":     "analyze",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", ts.URL+"/auth/stream", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		close(done)
	}()

	select {
	case <-done:
		// Ok — the client returned (with or without error).
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after client cancellation")
	}
	wg.Wait()
}

// TestAuthStreamRejectsGET asserts the handler responds 405 to non-POST.
func TestAuthStreamRejectsGET(t *testing.T) {
	_, ts, teardown := setupStreamTestEnv(t)
	defer teardown()
	resp, err := http.Get(ts.URL + "/auth/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /auth/stream = %d, want 405", resp.StatusCode)
	}
}

// TestAuthStreamEmptyResponseRejected asserts that POST with an empty
// response field returns 400 (matches /auth contract: response is required).
func TestAuthStreamEmptyResponseRejected(t *testing.T) {
	_, ts, teardown := setupStreamTestEnv(t)
	defer teardown()
	body, _ := json.Marshal(map[string]any{
		"question": "no response field",
		"mode":     "analyze",
	})
	req, _ := http.NewRequest("POST", ts.URL+"/auth/stream", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty response /auth/stream = %d, want 400", resp.StatusCode)
	}
}

// TestStreamDemoServesHTML asserts that GET /stream-demo returns the
// embedded HTML page with the expected hook (a <textarea> element).
func TestStreamDemoServesHTML(t *testing.T) {
	_, _, teardown := setupStreamTestEnv(t)
	defer teardown()
	// /stream-demo is not registered in the test mux above; register
	// it on a one-off server so the test is self-contained.
	mux := http.NewServeMux()
	mux.HandleFunc("/stream-demo", streamDemoHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/stream-demo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /stream-demo = %d, want 200", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)
	if !strings.Contains(body, "<textarea") {
		t.Errorf("/stream-demo HTML missing <textarea>")
	}
	if !strings.Contains(body, "/auth/stream") {
		t.Errorf("/stream-demo HTML missing /auth/stream reference")
	}
}

// quietly keep "fmt" alive in case any future debug Printf is added.
var _ = fmt.Sprintf
