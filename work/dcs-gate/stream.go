package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// authStreamHandler implements POST /auth/stream — SSE endpoint with mode support.
// mode=analyze: stream only analysis.
// mode=refine: stream only question refinement.
// mode=both: stream analysis first and refinement afterwards.
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
	if req.Mode == "" {
		req.Mode = "both"
	}
	if req.Response == "" && req.Mode != "refine" {
		http.Error(w, "response is required for analyze/both", http.StatusBadRequest)
		return
	}
	if req.Question == "" && req.Mode == "refine" {
		http.Error(w, "question is required for refine", http.StatusBadRequest)
		return
	}

	flusher, sendEvent, ok := prepareSSE(w)
	if !ok {
		http.Error(w, "streaming unsupported by the underlying writer", http.StatusInternalServerError)
		return
	}
	_ = flusher

	if req.Mode == "analyze" || req.Mode == "both" {
		steps, markers, traj, pole, top, _, crossCorpus, _, err := az.AnalyzeIntra(req.Response)
		if err != nil {
			_ = sendEvent("error", map[string]string{"stage": "pre_analysis", "message": err.Error()})
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
		if err := sendEvent("judge_loading", map[string]string{"status": "loading", "model": judge.cfg.JudgeModel}); err != nil {
			return
		}

		events := make(chan StreamEvent, 32)
		go judge.AnalyzeStream(r.Context(), req.Question, req.Response, steps, markers, traj, pole, top, crossCorpus, events)
		for ev := range events {
			if err := sendEvent(ev.Name, ev.Data); err != nil {
				return
			}
		}
	}

	if req.Mode == "refine" || req.Mode == "both" {
		if err := sendEvent("refine_loading", map[string]string{"status": "loading", "model": judge.cfg.JudgeModel}); err != nil {
			return
		}
		refined := judge.Refine(req.Question)
		if err := sendEvent("refine_complete", refined); err != nil {
			return
		}
	}
}

func prepareSSE(w http.ResponseWriter) (http.Flusher, func(string, any) error, bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, nil, false
	}
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
	return flusher, sendEvent, true
}

func analyzeStreamHandler(w http.ResponseWriter, r *http.Request) {
	streamHandlerWithMode(w, r, "analyze")
}

func refineStreamHandler(w http.ResponseWriter, r *http.Request) {
	streamHandlerWithMode(w, r, "refine")
}

func streamHandlerWithMode(w http.ResponseWriter, r *http.Request, mode string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Mode = mode
	b, _ := json.Marshal(req)
	r.Body = io.NopCloser(bytes.NewReader(b))
	authStreamHandler(w, r)
}
