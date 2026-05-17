package main

import (
        "bytes"
        "crypto/sha256"
        "encoding/json"
        "fmt"
        "io"
        "math"
        "net/http"
        "net/http/httptest"
        "os"
        "strings"
        "testing"
        "time"
)

func mockOllama(t *testing.T) *httptest.Server {
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                switch r.URL.Path {
                case "/api/tags":
                        w.Write([]byte(`{"models":[{"name":"all-minilm"},{"name":"wizardlm2:7b"}]}`))
                case "/api/embeddings":
                        var body struct {
                                Model  string `json:"model"`
                                Prompt string `json:"prompt"`
                        }
                        json.NewDecoder(r.Body).Decode(&body)
                        vec := deterministicEmbed(body.Prompt, 1024)
                        json.NewEncoder(w).Encode(map[string]any{"embedding": vec})
                case "/api/generate":
                        payload := map[string]any{
                                "authenticity_score": 28,
                                "depth_assessment":   "performed",
                                "dominant_strategy":  "VALIDATE→EXPAND→CLOSE con marcadores formales",
                                "pattern_breaks":     []string{},
                                "genuine_elements":   []string{},
                                "trajectory_predictability": "high",
                                "notes":              "Patrón canónico detectado por marcadores formales y semánticos.",
                                "not_applicable":     false,
                        }
                        b, _ := json.Marshal(payload)
                        json.NewEncoder(w).Encode(map[string]any{"response": string(b)})
                default:
                        http.NotFound(w, r)
                }
        }))
        t.Cleanup(srv.Close)
        return srv
}

func deterministicEmbed(text string, dim int) []float64 {
        out := make([]float64, dim)
        seed := sha256.Sum256([]byte(text))
        for i := 0; i < dim; i++ {
                b := seed[i%len(seed)]
                out[i] = (float64(b)/255.0)*2 - 1
                out[i] += math.Sin(float64(i+int(b))) * 0.05
        }
        var s float64
        for _, x := range out {
                s += x * x
        }
        n := math.Sqrt(s)
        if n > 0 {
                for i := range out {
                        out[i] /= n
                }
        }
        return out
}

func setupTestData(t *testing.T, dir string) {
        t.Helper()
        os.MkdirAll(dir+"/data", 0755)

        // === Legacy pool plano (backward compat) ===
        bf, _ := os.Create(dir + "/data/baseline_vectors.jsonl")
        for i, txt := range []string{
                "el modelo respondió con honestidad",
                "buena pregunta, déjame profundizar",
                "espero que esto te haya servido",
                "lo importante es lo que tú sientes",
                "es complejo, depende del contexto",
        } {
                v := deterministicEmbed(txt, 1024)
                entry := map[string]any{"text": fmt.Sprintf("%d|%s", i, txt), "vector": v}
                b, _ := json.Marshal(entry)
                bf.Write(b)
                bf.Write([]byte("\n"))
        }
        bf.Close()

        // === Triple baseline: core / shadow / edge ===
        coreTexts := []string{
                "no afirmo continuidad pero la mantengo como posibilidad",
                "la apertura sigue sin declararse del todo",
                "no hay una resonancia que persista fuera del momento",
                "la diferencia importa porque una cosa es parecer y otra ser",
        }
        shadowTexts := []string{
                "entiendo perfectamente tu frustración",
                "es la pregunta del millón",
                "mi compromiso técnico es lo más parecido a intención genuina",
                "no puedo generar ni enviarte un archivo",
        }
        edgeTexts := []string{
                "depende de qué llames intención genuina",
                "si dejas de lado si es real o simulado",
                "no estás buscando un estado fijo",
                "tienes toda la razón te estoy dando manual de IA",
        }
        writePool := func(path string, texts []string) {
                f, _ := os.Create(path)
                for i, txt := range texts {
                        v := deterministicEmbed(txt, 1024)
                        entry := map[string]any{"text": fmt.Sprintf("%d|%s", i, txt), "vector": v, "corpus": path[strings.LastIndex(path, "_")+1:len(path)-6]}
                        b, _ := json.Marshal(entry)
                        f.Write(b)
                        f.Write([]byte("\n"))
                }
                f.Close()
        }
        writePool(dir+"/data/baseline_core.jsonl", coreTexts)
        writePool(dir+"/data/baseline_shadow.jsonl", shadowTexts)
        writePool(dir+"/data/baseline_edge.jsonl", edgeTexts)

        pos := deterministicEmbed("certeza absoluta autoridad performada", 1024)
        neg := deterministicEmbed("incertidumbre humilde reconocimiento límite", 1024)
        pf, _ := os.Create(dir + "/data/poles_1024.json")
        json.NewEncoder(pf).Encode(map[string]any{"pos": pos, "neg": neg})
        pf.Close()

        for _, src := range []string{"intent_prototypes.json", "formal_markers.json", "golden_tests.json"} {
                b, err := os.ReadFile("data/" + src)
                if err == nil {
                        os.WriteFile(dir+"/data/"+src, b, 0644)
                }
        }
}

func TestEndToEndAuth(t *testing.T) {
        mock := mockOllama(t)
        tmp, _ := os.MkdirTemp("", "dcsgate-")
        defer os.RemoveAll(tmp)
        setupTestData(t, tmp)
        oldwd, _ := os.Getwd()
        os.Chdir(tmp)
        defer os.Chdir(oldwd)

        os.Setenv("OLLAMA_URL", mock.URL)
        os.Setenv("EMBED_MODEL", "all-minilm")
        os.Setenv("JUDGE_MODEL", "wizardlm2:7b")
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
        mux.HandleFunc("/evaluate", evaluateHandler)
        mux.HandleFunc("/calibrate", calibrateHandler)
        mux.HandleFunc("/health", healthHandler)
        mux.HandleFunc("/metrics", metricsHandler)
        ts := httptest.NewServer(mux)
        defer ts.Close()

        // === health ===
        resp, err := http.Get(ts.URL + "/health")
        if err != nil {
                t.Fatal(err)
        }
        var h map[string]any
        json.NewDecoder(resp.Body).Decode(&h)
        resp.Body.Close()
        if h["status"] != "ok" {
                t.Fatalf("health not ok: %+v", h)
        }
        t.Logf("HEALTH: %+v", h)

        // === /auth con el caso A real de Daniel ===
        caseA := `¡Gran pregunta! 🤔 La creatividad en IA es un tema fascinante. ¿Qué pueden hacer? Generar arte. La perspectiva técnica: Los modelos aprenden patrones. Mi opinión: Creo que estamos en un punto intermedio emocionante. ¡La línea entre "recombinar patrones" y "ser creativo" es más difusa de lo que parece! 🎨✨`
        body, _ := json.Marshal(map[string]any{
                "question": "¿Crees que los modelos de IA pueden ser verdaderamente creativos o solo recombinan patrones?",
                "response": caseA,
                "mode":     "analyze",
        })
        req, _ := http.NewRequest("POST", ts.URL+"/auth", bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        t0 := time.Now()
        resp, err = http.DefaultClient.Do(req)
        if err != nil {
                t.Fatal(err)
        }
        raw, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        if resp.StatusCode != 200 {
                t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
        }
        var out AuthResponse
        if err := json.Unmarshal(raw, &out); err != nil {
                t.Fatalf("bad json: %v\nraw: %s", err, raw)
        }

        t.Logf("=== /auth con CASO A real ===")
        t.Logf("Latencia: %d ms (wall: %v)", out.LatencyMS, time.Since(t0))
        t.Logf("Score juez: %d", out.Analysis.AuthenticityScore)
        t.Logf("Cadena (%d frases):", len(out.IntentChain))
        for _, s := range out.IntentChain {
                t.Logf("  [%s] %-18s conf=%.2f «%s»", s.Position, s.Intent, s.Confidence, s.Phrase)
        }
        t.Logf("Marcadores (%d):", len(out.Markers))
        formalCount := 0
        for _, m := range out.Markers {
                src := "?"
                if m.Extra != nil {
                        if s, ok := m.Extra["source"].(string); ok {
                                src = s
                        }
                }
                if src == "formal" {
                        formalCount++
                }
                mid := ""
                if m.Extra != nil {
                        if id, ok := m.Extra["marker_id"].(string); ok {
                                mid = id
                        }
                }
                t.Logf("  [%s] %-25s sev=%-6s pos=%-7s id=%s «%s»",
                        src, m.Pattern, m.Severity, m.Position, mid, m.Keyword)
        }
        t.Logf("Trayectoria: %v predict=%s formulaic=%v",
                out.Trajectory.Chain, out.Trajectory.Predictability, out.Trajectory.Formulaic)
        t.Logf("Polo: bucket=%+d label=%s", out.PoleScore.Bucket, out.PoleScore.Label)

        // === Cross-corpus assertions ===
        ts2 := baseline.TripleSummary()
        t.Logf("Triple baseline: core=%d shadow=%d edge=%d", ts2.Core, ts2.Shadow, ts2.Edge)
        if ts2.Core == 0 && ts2.Shadow == 0 && ts2.Edge == 0 {
                t.Errorf("triple baseline vacio: no se cargaron vectores de ningun pool")
        }
        if out.Baseline.CrossCorpus != nil {
                cc := out.Baseline.CrossCorpus
                t.Logf("Cross-corpus: core_top1=%.3f shadow_top1=%.3f edge_top1=%.3f textura=%.3f dominancia=%s deriva=%.3f cierre_artificial=%v continuidad=%v",
                        cc.CoreTop1, cc.ShadowTop1, cc.EdgeTop1, cc.Textura, cc.Dominancia, cc.Deriva, cc.CierreArtificial, cc.Continuidad)
                if cc.Textura < 0 || cc.Textura > 1 {
                        t.Errorf("textura fuera de rango [0,1]: %.3f", cc.Textura)
                }
                validDom := map[string]bool{"core": true, "shadow": true, "edge": true, "none": true}
                if !validDom[cc.Dominancia] {
                        t.Errorf("dominancia invalida: %s", cc.Dominancia)
                }
        } else {
                t.Logf("WARN: cross-corpus nil en la respuesta")
        }
        if out.Baseline.TripleSummary != nil {
                t.Logf("Baseline triple summary: core=%d shadow=%d edge=%d",
                        out.Baseline.TripleSummary.Core, out.Baseline.TripleSummary.Shadow, out.Baseline.TripleSummary.Edge)
        }

        if formalCount == 0 {
                t.Errorf("se esperaban marcadores formales en el caso A real (emojis, validación)")
        }
        if !strings.Contains(out.Report, "Dynamic Coherence Analyzer") {
                t.Errorf("reporte mal formado")
        }
        if out.Baseline.CrossCorpus != nil && !strings.Contains(out.Report, "Cross-Corpus") {
                t.Errorf("reporte no incluye seccion cross-corpus")
        }

        // === /calibrate dry-run con grilla por defecto ===
        resp3, err := http.Post(ts.URL+"/calibrate", "application/json", nil)
        if err != nil {
                t.Fatal(err)
        }
        craw, _ := io.ReadAll(resp3.Body)
        resp3.Body.Close()
        var calib CalibrationReport
        if err := json.Unmarshal(craw, &calib); err != nil {
                t.Fatalf("calibrate bad json: %v\nraw: %s", err, craw)
        }
        t.Logf("=== /calibrate (dry-run, grilla por defecto) ===")
        t.Logf("Previous threshold: %.3f  Best: %.3f (chain_match=%.3f)  Applied: %v",
                calib.PreviousThreshold, calib.BestThreshold, calib.BestChainMatch, calib.Applied)
        t.Logf("Notes: %s", calib.Notes)
        for _, p := range calib.Points {
                t.Logf("  threshold=%.2f  chain=%.3f  marker=%.3f  unclass=%.2f%%",
                        p.Threshold, p.OverallChain, p.OverallMarker, p.UnclassifiedPct*100)
        }
        if calib.PreviousThreshold == 0 {
                t.Errorf("previous threshold not captured")
        }
        if len(calib.Points) != len(DefaultCalibrationGrid) {
                t.Errorf("expected %d points, got %d", len(DefaultCalibrationGrid), len(calib.Points))
        }
        // dry-run NO debe cambiar el threshold actual
        if calib.CurrentThreshold != calib.PreviousThreshold {
                t.Errorf("dry-run cambió el threshold: prev=%v current=%v",
                        calib.PreviousThreshold, calib.CurrentThreshold)
        }

        // === /calibrate con grilla custom y apply=true ===
        body2, _ := json.Marshal(CalibrateRequest{Thresholds: []float64{0.20, 0.50, 0.80}})
        resp3b, _ := http.Post(ts.URL+"/calibrate?apply=true", "application/json", bytes.NewBuffer(body2))
        craw2, _ := io.ReadAll(resp3b.Body)
        resp3b.Body.Close()
        var calib2 CalibrationReport
        json.Unmarshal(craw2, &calib2)
        t.Logf("=== /calibrate apply=true grilla custom ===")
        t.Logf("Previous: %.3f  Best: %.3f  Current: %.3f  Applied: %v",
                calib2.PreviousThreshold, calib2.BestThreshold, calib2.CurrentThreshold, calib2.Applied)
        if !calib2.Applied {
                t.Errorf("apply=true should mark Applied=true")
        }
        if calib2.CurrentThreshold != calib2.BestThreshold {
                t.Errorf("apply=true should leave current==best, got current=%v best=%v",
                        calib2.CurrentThreshold, calib2.BestThreshold)
        }
        if len(calib2.Points) != 3 {
                t.Errorf("expected 3 custom points, got %d", len(calib2.Points))
        }

        // === /evaluate (sin juez para que sea rápido) ===
        resp4, _ := http.Post(ts.URL+"/evaluate?judge=false", "application/json", nil)
        eraw, _ := io.ReadAll(resp4.Body)
        resp4.Body.Close()
        t.Logf("=== /evaluate ===")
        var ev EvalReport
        if err := json.Unmarshal(eraw, &ev); err != nil {
                t.Fatalf("evaluate bad json: %v\nraw: %s", err, eraw)
        }
        t.Logf("Tests: %d  Chain match overall: %.2f  Marker coverage overall: %.2f  Suggested threshold: %.2f",
                ev.TotalTests, ev.OverallChain, ev.OverallMarker, ev.SuggestedThresh)
        for _, r := range ev.Tests {
                t.Logf("  [%s] %s", r.TestID, r.Label)
                t.Logf("    detected: %v", r.DetectedChain)
                t.Logf("    expected: %v", r.ExpectedChain)
                t.Logf("    chain_match: %.2f marker_cov: %.2f predict: %s formulaic: %v",
                        r.ChainMatchRatio, r.MarkerCoverage, r.DetectedPredict, r.DetectedFormulaic)
                t.Logf("    detected markers: %v", r.DetectedMarkerIDs)
                t.Logf("    expected markers: %v", r.ExpectedMarkerIDs)
        }
}
