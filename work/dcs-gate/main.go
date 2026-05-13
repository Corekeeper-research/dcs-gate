package main

import (
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "time"
)

var (
        cfg       Config
        emb       *Embedder
        baseline  *Baseline
        intents   *IntentBank
        formal    *FormalDetector
        judge     *Judge
        az        *Analyzer
        golden    *GoldenSet
        evaluator *Evaluator
)

func main() {
        cfg = loadConfig()
        emb = NewEmbedder(cfg)
        // Cargar baseline triple si existen los tres archivos;
        // si no, fallback al pool plano legacy.
        triplePaths := map[string]string{
                "data/baseline_core.jsonl":     "core",
                "data/baseline_shadow.jsonl":   "shadow",
                "data/baseline_edge.jsonl":     "edge",
        }
        var tripleFound int
        for path := range triplePaths {
                if _, err := os.Stat(path); err == nil {
                        tripleFound++
                }
        }
        if tripleFound >= 2 {
                baseline = LoadTripleBaseline(
                        "data/baseline_core.jsonl",
                        "data/baseline_shadow.jsonl",
                        "data/baseline_edge.jsonl",
                )
        } else {
                baseline = LoadBaseline("data/baseline_vectors.jsonl")
        }
        baseline.LoadPoles("data/poles_1024.json")
        intents = LoadIntents("data/intent_prototypes.json", cfg.IntentThreshold)
        formal = LoadFormalMarkers("data/formal_markers.json")
        golden = LoadGolden("data/golden_tests.json")
        judge = NewJudge(cfg)

        // BuildCentroids síncrono antes de ListenAndServe: evita race
        // condition sobre intents.centroids cuando /auth llega antes de
        // que la goroutine termine de escribir el mapa. También garantiza
        // que /health y /metrics reporten centroides reales desde el
        // primer request.
        waitOllama(cfg)
        intents.BuildCentroids(emb)
        log.Printf("centroides listos: %v", intents.Names())

        az = NewAnalyzer(cfg, emb, baseline, intents, formal)
        evaluator = NewEvaluator(golden, az, judge)

        mux := http.NewServeMux()
        mux.HandleFunc("/", indexHandler)
        mux.HandleFunc("/auth", authHandler)
        mux.HandleFunc("/auth/stream", authStreamHandler) // v8.7
        mux.HandleFunc("/stream-demo", streamDemoHandler) // v8.7
        mux.HandleFunc("/evaluate", evaluateHandler)
        mux.HandleFunc("/calibrate", calibrateHandler)
        mux.HandleFunc("/health", healthHandler)
        mux.HandleFunc("/metrics", metricsHandler)

        addr := ":" + cfg.Port
        ts := baseline.TripleSummary()
        log.Printf("dcs-gate v8.7 escuchando en %s — vectors=%d dim=%d core=%d shadow=%d edge=%d intents=%v",
                addr, baseline.Size(), baseline.Dim(), ts.Core, ts.Shadow, ts.Edge, IntentNames)
        log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

func withCORS(h http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ngrok-skip-browser-warning")
                if r.Method == "OPTIONS" {
                        w.WriteHeader(204)
                        return
                }
                h.ServeHTTP(w, r)
        })
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
                http.NotFound(w, r)
                return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write([]byte(indexHTML))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
                "status":         "ok",
                "vectors":        baseline.Size(),
                "dim":            baseline.Dim(),
                "intents":        intents.Names(),
                "poles_ok":       baseline.polePos != nil && baseline.poleNeg != nil,
                "golden_tests":   len(golden.Tests),
                "formal_markers": len(formal.markers),
        })
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
        size, hits, miss := emb.cache.Stats()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
                "cache_size":   size,
                "cache_hits":   hits,
                "cache_misses": miss,
                "hit_ratio":    hitRatio(hits, miss),
                "vectors":      baseline.Size(),
                "intents":      intents.Names(),
        })
}

func hitRatio(h, m uint64) float64 {
        if h+m == 0 {
                return 0
        }
        return float64(h) / float64(h+m)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "POST only", http.StatusMethodNotAllowed)
                return
        }
        t0 := time.Now()
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

        resp := AuthResponse{}

        if req.Mode == "analyze" || req.Mode == "both" {
                steps, markers, traj, pole, top, _, crossCorpus, _, err := az.AnalyzeIntra(req.Response)
                if err != nil {
                        http.Error(w, "analyze failed: "+err.Error(), http.StatusInternalServerError)
                        return
                }
                resp.IntentChain = steps
                resp.Markers = markers
                resp.Trajectory = traj
                resp.PoleScore = pole
                resp.TopK = top
                resp.Baseline = BaselineResult{
                        CosineTop1:    top1Score(top),
                        LoadedVectors: baseline.Size(),
                        Dim:           baseline.Dim(),
                        CrossCorpus:   crossCorpus,
                        TripleSummary: baseline.TripleSummary(),
                }
                resp.Cached = false
                resp.Analysis = judge.Analyze(req.Question, req.Response, steps, markers, traj, pole, top, crossCorpus)
        }

        if req.Mode == "refine" || req.Mode == "both" {
                resp.RefinedQuestion = judge.Refine(req.Question)
        }

        resp.LatencyMS = time.Since(t0).Milliseconds()
        resp.Report = BuildReport(req.Question, resp.Analysis, resp.RefinedQuestion,
                resp.IntentChain, resp.Markers, resp.Trajectory, resp.PoleScore, resp.TopK,
                resp.Baseline.CrossCorpus, resp.Cached, resp.LatencyMS, baseline.Size(), baseline.Dim())

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
}

// evaluateHandler corre los golden tests y devuelve un reporte de cobertura.
// Acepta ?judge=false para saltar la llamada al juez (calibración rápida).
func evaluateHandler(w http.ResponseWriter, r *http.Request) {
        callJudge := r.URL.Query().Get("judge") != "false"
        if evaluator == nil || golden == nil || len(golden.Tests) == 0 {
                http.Error(w, "no golden tests loaded", http.StatusServiceUnavailable)
                return
        }
        rep := evaluator.Run(callJudge)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(rep)
}

// calibrateHandler barre una grilla de umbrales sobre los golden tests y
// devuelve el que maximiza chain_match. Por defecto NO llama al juez (más rápido).
//
//   POST /calibrate                       -> dry-run con grilla por defecto
//   POST /calibrate?apply=true            -> aplica el mejor umbral en runtime
//   POST /calibrate?judge=true            -> también consulta wizardlm2:7b
//   POST /calibrate (body: {"thresholds":[0.30,0.40,0.50]}) -> grilla custom
func calibrateHandler(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                http.Error(w, "POST only", http.StatusMethodNotAllowed)
                return
        }
        if evaluator == nil || golden == nil || len(golden.Tests) == 0 {
                http.Error(w, "no golden tests loaded — sin tests no hay nada que calibrar", http.StatusServiceUnavailable)
                return
        }
        var req CalibrateRequest
        if r.ContentLength > 0 {
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
                        return
                }
        }
        apply := r.URL.Query().Get("apply") == "true"
        callJudge := r.URL.Query().Get("judge") == "true"

        rep := evaluator.Calibrate(req.Thresholds, callJudge, apply)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(rep)
}

func waitOllama(cfg Config) {
        url := cfg.OllamaURL + "/api/tags"
        for i := 0; i < 60; i++ {
                resp, err := http.Get(url)
                if err == nil && resp.StatusCode < 300 {
                        resp.Body.Close()
                        log.Printf("ollama listo en %s", cfg.OllamaURL)
                        return
                }
                if resp != nil {
                        resp.Body.Close()
                }
                time.Sleep(2 * time.Second)
        }
        log.Printf("WARN ollama no respondió tras 120s en %s", cfg.OllamaURL)
}

func printConfig() string {
        return fmt.Sprintf("port=%s ollama=%s embed=%s judge=%s",
                cfg.Port, cfg.OllamaURL, cfg.EmbedModel, cfg.JudgeModel)
}
