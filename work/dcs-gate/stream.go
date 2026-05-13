package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// authStreamHandler implements POST /auth/stream — the v8.7 Server-Sent
// Events endpoint. It runs the same pipeline as authHandler (algorithmic
// pre-analysis followed by judge call) but emits each piece of the result
// as a discrete SSE event the moment it becomes available.
//
// Event sequence (see DISENO_v87_STREAMING.md §0):
//
//     pre_analysis        — intent_chain, trajectory, pole, baseline_top1,
//                           top_k, cross_corpus, markers (~300-500 ms)
//     judge_loading       — UI hint that the judge model is being loaded
//     thinking_chunk × N  — sanitized chain-of-thought chunks (qwen3/r1)
//     thinking_complete   — emitted once the response phase starts
//     analysis_chunk × M  — sanitized chunks of the structured JSON output
//     complete            — parsed AuthenticityAnalysis with judge_thinking
//
// Error events (terminate the stream):
//     error        — any failure before/during the call (stage indicates where)
//     parse_error  — Ollama finished but the JSON did not decode; thinking
//                    is preserved so the client can still display the CoT
//
// The existing /auth endpoint is NOT modified — clients that expect a single
// JSON body continue to use it. /auth/stream is opt-in.
func authStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Response == "" {
		http.Error(w, "response is required", http.StatusBadRequest)
		return
	}

	// SSE headers. X-Accel-Buffering disables nginx's response buffering
	// for installations sitting behind a reverse proxy; the header is
	// harmless when the proxy ignores it.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported by the underlying writer",
			http.StatusInternalServerError)
		return
	}

	// sendEvent serialises data to JSON and writes a complete SSE event
	// (event:, data:, blank line) to the response, then flushes so the
	// client receives the chunk immediately. Returns the write error if
	// the connection has been closed by the peer.
	sendEvent := func(name string, data any) error {
		payload, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	// ---- Pre-analysis (algorithmic, deterministic, ~300-500 ms) ----
	steps, markers, traj, pole, top, _, crossCorpus, _, err := az.AnalyzeIntra(req.Response)
	if err != nil {
		_ = sendEvent("error", map[string]string{
			"stage":   "pre_analysis",
			"message": err.Error(),
		})
		return
	}
	preData := map[string]any{
		"intent_chain":  steps,
		"trajectory":    traj,
		"pole_score":    pole,
		"baseline_top1": top1Score(top),
		"top_k":         top,
		"cross_corpus":  crossCorpus,
		"markers":       markers,
	}
	if err := sendEvent("pre_analysis", preData); err != nil {
		return
	}
	if err := sendEvent("judge_loading", map[string]string{
		"status": "loading",
		"model":  judge.cfg.JudgeModel,
	}); err != nil {
		return
	}

	// ---- Judge stream ----
	// Buffer of 32 keeps the producer goroutine from blocking on the
	// network writer for typical bursts of thinking chunks. The goroutine
	// closes the channel on return, which terminates this loop.
	events := make(chan StreamEvent, 32)
	go judge.AnalyzeStream(r.Context(), req.Question, req.Response,
		steps, markers, traj, pole, top, crossCorpus, events)

	for ev := range events {
		if err := sendEvent(ev.Name, ev.Data); err != nil {
			// Client closed the connection. The producer goroutine
			// will notice via ctx.Err() and exit on its next loop
			// iteration; we just stop reading.
			return
		}
	}
}
