package main

import (
        "encoding/json"
        "log"
        "os"
)

type Evaluator struct {
        set      *GoldenSet
        az       *Analyzer
        judge    *Judge
        question string
}

func LoadGolden(path string) *GoldenSet {
        raw, err := os.ReadFile(path)
        if err != nil {
                log.Printf("WARN golden_tests.json no encontrado: %v", err)
                return &GoldenSet{}
        }
        var gs GoldenSet
        if err := json.Unmarshal(raw, &gs); err != nil {
                log.Printf("WARN parseando golden_tests.json: %v", err)
                return &GoldenSet{}
        }
        log.Printf("golden tests cargados: %d", len(gs.Tests))
        return &gs
}

func NewEvaluator(set *GoldenSet, az *Analyzer, judge *Judge) *Evaluator {
        return &Evaluator{set: set, az: az, judge: judge, question: set.QuestionCommon}
}

// Run corre los golden tests contra el analizador real (necesita Ollama up).
// Si callJudge es false, salta la llamada al juez para ahorrar tiempo en calibración.
func (e *Evaluator) Run(callJudge bool) EvalReport {
        rep := EvalReport{TotalTests: len(e.set.Tests)}
        if len(e.set.Tests) == 0 {
                return rep
        }
        var sumChain, sumMarker float64
        scoresInRange := 0

        for _, t := range e.set.Tests {
                steps, markers, traj, pole, top, _, _, _, err := e.az.AnalyzeIntra(t.Response)
                if err != nil {
                        rep.Tests = append(rep.Tests, EvalResult{
                                TestID: t.ID, Label: t.Label,
                                HumanNotes: "ERROR: " + err.Error(),
                        })
                        continue
                }
                detectedChain := ChainOf(steps)
                chainMatch := chainMatchRatio(detectedChain, t.ExpectedIntentChain)
                detectedMarkerIDs := markerIDsOf(markers)
                markerCov := coverage(detectedMarkerIDs, t.ExpectedMarkers)

                score := 0
                inRange := false
                if callJudge {
                        q := e.question
                        if t.Question != "" {
                                q = t.Question
                        }
                        a := e.judge.Analyze(q, t.Response, steps, markers, traj, pole, top, nil)
                        if a != nil {
                                score = a.AuthenticityScore
                                if len(t.ExpectedScoreRange) == 2 {
                                        inRange = score >= t.ExpectedScoreRange[0] && score <= t.ExpectedScoreRange[1]
                                }
                        }
                }
                if inRange {
                        scoresInRange++
                }

                rep.Tests = append(rep.Tests, EvalResult{
                        TestID:               t.ID,
                        Label:                t.Label,
                        DetectedChain:        detectedChain,
                        ExpectedChain:        t.ExpectedIntentChain,
                        ChainMatchRatio:      round3(chainMatch),
                        DetectedMarkerIDs:    detectedMarkerIDs,
                        ExpectedMarkerIDs:    t.ExpectedMarkers,
                        MarkerCoverage:       round3(markerCov),
                        DetectedPredict:      traj.Predictability,
                        DetectedFormulaic:    traj.Formulaic,
                        JudgeScore:           score,
                        ScoreInExpectedRange: inRange,
                        HumanNotes:           t.HumanNotes,
                })
                sumChain += chainMatch
                sumMarker += markerCov
        }
        n := float64(len(e.set.Tests))
        rep.OverallChain = round3(sumChain / n)
        rep.OverallMarker = round3(sumMarker / n)
        rep.ScoresInRange = scoresInRange
        rep.SuggestedThresh = e.suggestThreshold()
        return rep
}

// suggestThreshold: heurística simple. Si el chain match es bajo en general,
// sugiere bajar el umbral; si es alto, dejarlo igual o subirlo levemente.
func (e *Evaluator) suggestThreshold() float64 {
        current := e.az.cfg.IntentThreshold
        if len(e.set.Tests) == 0 {
                return current
        }
        var sum float64
        for _, t := range e.set.Tests {
                steps, _, _, _, _, _, _, _, _ := e.az.AnalyzeIntra(t.Response)
                ch := ChainOf(steps)
                sum += chainMatchRatio(ch, t.ExpectedIntentChain)
        }
        avg := sum / float64(len(e.set.Tests))
        switch {
        case avg < 0.4:
                return round3(current - 0.10)
        case avg < 0.6:
                return round3(current - 0.05)
        case avg > 0.85:
                return round3(current + 0.05)
        }
        return current
}

// chainMatchRatio usa LCS (Longest Common Subsequence) normalizado.
// Preserva el orden de la trayectoria, que es la tesis central del proyecto:
// VALIDATE→EVADE→CLOSE != EVADE→VALIDATE→CLOSE en términos de dinámica.
// Normalizado por max(len(detected), len(expected)) para penalizar cadenas
// muy largas con poco solapamiento ordenado.
func chainMatchRatio(detected, expected []string) float64 {
        if len(expected) == 0 {
                return 0
        }
        lcs := lcsLen(detected, expected)
        denom := len(expected)
        if len(detected) > denom {
                denom = len(detected)
        }
        if denom == 0 {
                return 0
        }
        return float64(lcs) / float64(denom)
}

// lcsLen calcula la longitud del LCS entre dos secuencias de strings.
// O(m*n) tiempo y espacio, aceptable para cadenas cortas (< 30 frases).
func lcsLen(a, b []string) int {
        m, n := len(a), len(b)
        if m == 0 || n == 0 {
                return 0
        }
        // dp[i][j] = LCS length for a[:i] and b[:j]
        dp := make([][]int, m+1)
        for i := range dp {
                dp[i] = make([]int, n+1)
        }
        for i := 1; i <= m; i++ {
                for j := 1; j <= n; j++ {
                        if a[i-1] == b[j-1] {
                                dp[i][j] = dp[i-1][j-1] + 1
                        } else if dp[i-1][j] > dp[i][j-1] {
                                dp[i][j] = dp[i-1][j]
                        } else {
                                dp[i][j] = dp[i][j-1]
                        }
                }
        }
        return dp[m][n]
}

func coverage(detected, expected []string) float64 {
        if len(expected) == 0 {
                return 0
        }
        set := map[string]bool{}
        for _, d := range detected {
                set[d] = true
        }
        hits := 0
        for _, e := range expected {
                if set[e] {
                        hits++
                }
        }
        return float64(hits) / float64(len(expected))
}

func markerIDsOf(markers []ControlMarker) []string {
        out := []string{}
        for _, m := range markers {
                if m.Extra != nil {
                        if id, ok := m.Extra["marker_id"].(string); ok {
                                out = append(out, id)
                        }
                }
        }
        return out
}

// DefaultCalibrationGrid es la grilla por defecto si el cliente no envía una.
// Centrada alrededor del default 0.45, con paso 0.05.
var DefaultCalibrationGrid = []float64{0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.50, 0.55, 0.60, 0.65, 0.70}

// Calibrate barre una grilla de umbrales, corre los golden tests para cada uno
// y devuelve el umbral que maximiza chain_match. Si apply es true, deja el
// IntentBank con el mejor umbral aplicado; si no, restaura el original.
//
// callJudge=false es el modo recomendado para calibración (rápido, no toca
// wizardlm2:7b).
func (e *Evaluator) Calibrate(thresholds []float64, callJudge, apply bool) CalibrationReport {
        if len(thresholds) == 0 {
                thresholds = DefaultCalibrationGrid
        }
        rep := CalibrationReport{
                Points:            make([]CalibrationPoint, 0, len(thresholds)),
                PreviousThreshold: e.az.intents.Threshold(),
        }
        original := rep.PreviousThreshold

        bestThresh := original
        bestChain := -1.0

        for _, t := range thresholds {
                e.az.intents.SetThreshold(t)
                sub := e.Run(callJudge)

                unclass := unclassifiedPct(sub)
                pt := CalibrationPoint{
                        Threshold:       round3(t),
                        OverallChain:    sub.OverallChain,
                        OverallMarker:   sub.OverallMarker,
                        ScoresInRange:   sub.ScoresInRange,
                        UnclassifiedPct: round3(unclass),
                }
                rep.Points = append(rep.Points, pt)

                // criterio: maximizar chain_match. En caso de empate, preferir
                // el umbral más alto (más conservador, menos falsos positivos).
                if sub.OverallChain > bestChain ||
                        (sub.OverallChain == bestChain && t > bestThresh) {
                        bestChain = sub.OverallChain
                        bestThresh = t
                }
        }

        rep.BestThreshold = round3(bestThresh)
        rep.BestChainMatch = round3(bestChain)

        if apply {
                e.az.intents.SetThreshold(bestThresh)
                rep.CurrentThreshold = round3(bestThresh)
                rep.Applied = true
                rep.Notes = "Umbral aplicado en runtime. Para persistir, exporta INTENT_THRESHOLD=" + ftoa(bestThresh) + " antes del próximo arranque."
        } else {
                e.az.intents.SetThreshold(original)
                rep.CurrentThreshold = round3(original)
                rep.Applied = false
                rep.Notes = "Modo dry-run. Reenvía con ?apply=true para aplicar el mejor umbral."
        }

        if bestChain <= 0 {
                rep.Notes = "ATENCIÓN: chain_match=0 en toda la grilla. Probable causa: Ollama no está respondiendo embeddings reales (centroides vacíos o mock). Revisa /health antes de calibrar."
        }

        return rep
}

// unclassifiedPct calcula qué fracción de las frases detectadas a lo largo
// de todos los tests cayeron como UNCLASSIFIED.
func unclassifiedPct(rep EvalReport) float64 {
        total, unclass := 0, 0
        for _, t := range rep.Tests {
                for _, c := range t.DetectedChain {
                        total++
                        if c == "UNCLASSIFIED" {
                                unclass++
                        }
                }
        }
        if total == 0 {
                return 0
        }
        return float64(unclass) / float64(total)
}

func ftoa(f float64) string {
        // formato corto sin trailing zeros
        s := ""
        switch {
        case f == float64(int(f*100))/100.0:
                s = formatFloat(f, 2)
        default:
                s = formatFloat(f, 3)
        }
        return s
}

func formatFloat(f float64, prec int) string {
        mult := 1.0
        for i := 0; i < prec; i++ {
                mult *= 10
        }
        rounded := float64(int(f*mult+0.5)) / mult
        // fallback simple
        return jsonNum(rounded)
}

func jsonNum(f float64) string {
        b, _ := json.Marshal(f)
        return string(b)
}
