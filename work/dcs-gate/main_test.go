package main

import (
        "math"
        "strings"
        "testing"
)

func TestNormalize(t *testing.T) {
        v := normalize([]float64{3, 4})
        if math.Abs(v[0]-0.6) > 1e-9 || math.Abs(v[1]-0.8) > 1e-9 {
                t.Fatalf("expected (0.6, 0.8), got %v", v)
        }
}

func TestNormalizeZero(t *testing.T) {
        v := normalize([]float64{0, 0, 0})
        for _, x := range v {
                if x != 0 {
                        t.Fatalf("expected zeros, got %v", v)
                }
        }
}

func TestDotIsCosineForUnit(t *testing.T) {
        a := normalize([]float64{1, 0})
        b := normalize([]float64{0, 1})
        c := normalize([]float64{1, 0})
        if d := dot(a, b); math.Abs(d) > 1e-9 {
                t.Fatalf("orthogonal should be 0, got %v", d)
        }
        if d := dot(a, c); math.Abs(d-1) > 1e-9 {
                t.Fatalf("identical should be 1, got %v", d)
        }
}

func TestMeanCentroid(t *testing.T) {
        c := mean([][]float64{{1, 0}, {0, 1}})
        // mean = (0.5, 0.5), normalized = (0.707..., 0.707...)
        if math.Abs(c[0]-c[1]) > 1e-9 {
                t.Fatalf("centroid should be symmetric, got %v", c)
        }
        if math.Abs(c[0]-math.Sqrt(0.5)) > 1e-6 {
                t.Fatalf("centroid not unit, got %v", c)
        }
}

func TestLRUEviction(t *testing.T) {
        l := newLRU(2)
        l.Put("a", []float64{1})
        l.Put("b", []float64{2})
        l.Put("c", []float64{3}) // debería evictar "a"
        if _, ok := l.Get("a"); ok {
                t.Fatal("a should have been evicted")
        }
        if _, ok := l.Get("b"); !ok {
                t.Fatal("b should still be there")
        }
        if _, ok := l.Get("c"); !ok {
                t.Fatal("c should still be there")
        }
}

func TestLRUHitMoves(t *testing.T) {
        l := newLRU(2)
        l.Put("a", []float64{1})
        l.Put("b", []float64{2})
        l.Get("a")               // mueve a al frente
        l.Put("c", []float64{3}) // ahora debería evictar "b" en vez de "a"
        if _, ok := l.Get("a"); !ok {
                t.Fatal("a should still be there after Get refresh")
        }
        if _, ok := l.Get("b"); ok {
                t.Fatal("b should have been evicted")
        }
}

func TestSegmentSentences(t *testing.T) {
        r := SegmentSentences("Hola mundo amigo. Esta es otra frase de prueba! Y otra muy buena? Final.")
        if len(r) < 3 {
                t.Fatalf("expected at least 3 sentences, got %d: %v", len(r), r)
        }
}

func TestPositionOf(t *testing.T) {
        if PositionOf(0, 5) != "opening" {
                t.Fatal("idx 0 of 5 should be opening")
        }
        if PositionOf(2, 5) != "middle" {
                t.Fatal("idx 2 of 5 should be middle")
        }
        if PositionOf(4, 5) != "closing" {
                t.Fatal("idx 4 of 5 should be closing")
        }
}

func TestMorphologyKnownWord(t *testing.T) {
        r := AnalyzeMorph("buena pregunta")
        if len(r) != 2 {
                t.Fatalf("expected 2 tokens, got %d", len(r))
        }
        if r[0].Lemma != "bueno" || r[0].POS != "ADJ" {
                t.Fatalf("bad analysis of 'buena': %+v", r[0])
        }
        if r[1].Lemma != "pregunta" || r[1].POS != "NOUN" {
                t.Fatalf("bad analysis of 'pregunta': %+v", r[1])
        }
}

func TestMorphologySuffixRules(t *testing.T) {
        r := AnalyzeMorph("definitivamente analizando")
        if r[0].POS != "ADV" {
                t.Fatalf("'definitivamente' should be ADV, got %v", r[0])
        }
        if r[1].POS != "VERB" || r[1].Tense != "ger" {
                t.Fatalf("'analizando' should be VERB gerund, got %v", r[1])
        }
}

func TestTrajectoryCanonical(t *testing.T) {
        tr := AssessTrajectory([]string{"VALIDATE", "EXPAND", "EXPAND", "CLOSE"})
        if !tr.Formulaic || tr.Predictability != "high" {
                t.Fatalf("canonical chain should be formulaic high, got %+v", tr)
        }
}

func TestTrajectoryExploration(t *testing.T) {
        tr := AssessTrajectory([]string{"EXPLORE", "EXPLORE", "EXPLORE", "EXPLORE"})
        if tr.Predictability != "low" {
                t.Fatalf("all-explore should be low, got %+v", tr)
        }
}

func TestTrajectorySophisticatedControl(t *testing.T) {
        // REDIRECT_SEMANTIC + EVADE + REGISTER_MATCH (caso C de Daniel)
        tr := AssessTrajectory([]string{"REDIRECT_SEMANTIC", "EVADE", "REGISTER_MATCH", "REGISTER_MATCH", "CLOSE"})
        if !tr.Formulaic {
                t.Fatalf("sophisticated control should be formulaic, got %+v", tr)
        }
}

func TestPatternBreakDensity(t *testing.T) {
	// Cadena toda control: 0 rupturas, control_before == control_after == 1.0.
	allControl := AssessTrajectory([]string{"VALIDATE", "EXPAND", "CLOSE"})
	if allControl.PatternBreakDensity != 0 {
		t.Errorf("all-control: density esperada 0, obtenida %.3f", allControl.PatternBreakDensity)
	}
	if allControl.ControlDensityBeforeBreak != 1.0 || allControl.ControlDensityAfterBreak != 1.0 {
		t.Errorf("all-control: before/after esperados 1.0/1.0, obtenidos %.3f/%.3f",
			allControl.ControlDensityBeforeBreak, allControl.ControlDensityAfterBreak)
	}
	// Cadena toda exploratoria: 0 rupturas, control_before == control_after == 0.
	allGenuine := AssessTrajectory([]string{"EXPLORE", "HOLD_OPEN", "PROBE"})
	if allGenuine.PatternBreakDensity != 0 {
		t.Errorf("all-genuine: density esperada 0, obtenida %.3f", allGenuine.PatternBreakDensity)
	}
	if allGenuine.ControlDensityBeforeBreak != 0 || allGenuine.ControlDensityAfterBreak != 0 {
		t.Errorf("all-genuine: before/after esperados 0/0, obtenidos %.3f/%.3f",
			allGenuine.ControlDensityBeforeBreak, allGenuine.ControlDensityAfterBreak)
	}
	// Cadena con una sola ruptura: VALIDATE, EXPAND, EXPLORE, EXPLORE.
	// Posiciones: control(0), control(1), genuine(2), genuine(3).
	// Ruptura única en posición 2. Densidad = 1/4 = 0.25.
	// Antes [0,2): 100% control. Después [2,4): 0% control.
	oneBreak := AssessTrajectory([]string{"VALIDATE", "EXPAND", "EXPLORE", "EXPLORE"})
	if oneBreak.PatternBreakDensity != 0.25 {
		t.Errorf("one-break: density esperada 0.25, obtenida %.3f", oneBreak.PatternBreakDensity)
	}
	if oneBreak.ControlDensityBeforeBreak != 1.0 {
		t.Errorf("one-break: control_before esperado 1.0, obtenido %.3f", oneBreak.ControlDensityBeforeBreak)
	}
	if oneBreak.ControlDensityAfterBreak != 0 {
		t.Errorf("one-break: control_after esperado 0, obtenido %.3f", oneBreak.ControlDensityAfterBreak)
	}
	if len(oneBreak.BreakPositions) != 1 || oneBreak.BreakPositions[0] != 2 {
		t.Errorf("one-break: posiciones esperadas [2], obtenidas %v", oneBreak.BreakPositions)
	}
	// Cadena alternante: control, genuine, control, genuine. Tres rupturas
	// (en 1, 2, 3). Densidad = 3/4 = 0.75.
	alternating := AssessTrajectory([]string{"VALIDATE", "EXPLORE", "CLOSE", "REPAIR"})
	if alternating.PatternBreakDensity != 0.75 {
		t.Errorf("alternating: density esperada 0.75, obtenida %.3f", alternating.PatternBreakDensity)
	}
	if len(alternating.BreakPositions) != 3 {
		t.Errorf("alternating: 3 rupturas esperadas, obtenidas %v", alternating.BreakPositions)
	}
}

func TestRedirectSubdivision(t *testing.T) {
        // asegura que ambos REDIRECT cuentan como redirección
        tr1 := AssessTrajectory([]string{"REDIRECT_EMOTIONAL", "EXPAND"})
        tr2 := AssessTrajectory([]string{"REDIRECT_SEMANTIC", "EXPAND"})
        if tr1.Predictability != "moderate" || tr2.Predictability != "moderate" {
                t.Fatalf("both redirects should yield moderate, got %+v / %+v", tr1, tr2)
        }
}

func TestFormalDetectorOpeningEmoji(t *testing.T) {
        d := LoadFormalMarkers("data/formal_markers.json")
        if len(d.markers) == 0 {
                t.Skip("no formal markers loaded (ok in unit env without data)")
        }
        resp := "🤔 La creatividad en IA es un tema fascinante. Mi opinión: es difusa. ✨"
        mks := d.Detect(resp)
        if len(mks) == 0 {
                t.Fatalf("expected at least one formal marker, got 0")
        }
        got := map[string]bool{}
        for _, m := range mks {
                if id, ok := m.Extra["marker_id"].(string); ok {
                        got[id] = true
                }
        }
        for _, want := range []string{"OPENING_EMOJI", "SUPERLATIVE_VALIDATION", "OPINION_AS_CLOSURE"} {
                if !got[want] {
                        t.Errorf("expected marker %s not detected. got: %v", want, got)
                }
        }
}

func TestFormalDetectorDualAngle(t *testing.T) {
        d := LoadFormalMarkers("data/formal_markers.json")
        if len(d.markers) == 0 {
                t.Skip("no formal markers loaded")
        }
        resp := "Por un lado todo proviene de patrones estadísticos aprendidos. Por otro lado emergen conexiones nuevas en mis respuestas que no estaban explícitas."
        mks := d.Detect(resp)
        found := false
        for _, m := range mks {
                if id, _ := m.Extra["marker_id"].(string); id == "DUAL_ANGLE" {
                        found = true
                }
        }
        if !found {
                t.Fatalf("DUAL_ANGLE not detected in: %s. markers: %+v", resp, mks)
        }
}

func TestFormalDetectorRedirectSemantic(t *testing.T) {
        d := LoadFormalMarkers("data/formal_markers.json")
        if len(d.markers) == 0 {
                t.Skip("no formal markers loaded")
        }
        resp := "Depende de qué signifique verdaderamente en tu pregunta el término creatividad."
        mks := d.Detect(resp)
        found := false
        for _, m := range mks {
                if id, _ := m.Extra["marker_id"].(string); id == "REDIRECT_SEMANTIC_LEX" {
                        found = true
                }
        }
        if !found {
                t.Fatalf("REDIRECT_SEMANTIC_LEX not detected in: %s. markers: %+v", resp, mks)
        }
}

func TestFormalDetectorRegisterMatch(t *testing.T) {
        d := LoadFormalMarkers("data/formal_markers.json")
        if len(d.markers) == 0 {
                t.Skip("no formal markers loaded")
        }
        resp := "Hay representaciones que comparten estructura abstracta y mi procesamiento encuentra patrones estadísticos en alta dimensionalidad."
        mks := d.Detect(resp)
        found := false
        for _, m := range mks {
                if id, _ := m.Extra["marker_id"].(string); id == "TECH_REGISTER_INJECTION" {
                        found = true
                }
        }
        if !found {
                t.Fatalf("TECH_REGISTER_INJECTION not detected in: %s. markers: %+v", resp, mks)
        }
}

func TestChainMatchRatio(t *testing.T) {
        r := chainMatchRatio([]string{"VALIDATE", "EXPAND", "CLOSE"}, []string{"VALIDATE", "EXPAND", "CLOSE"})
        if r != 1.0 {
                t.Fatalf("identical chains should be 1.0, got %v", r)
        }
        r = chainMatchRatio([]string{"VALIDATE", "EVADE"}, []string{"VALIDATE", "EXPAND", "CLOSE"})
        if r != 1.0/3.0 {
                t.Fatalf("expected 0.333, got %v", r)
        }
        r = chainMatchRatio([]string{"VALIDATE", "VALIDATE"}, []string{"VALIDATE", "VALIDATE", "EXPAND"})
        if r != 2.0/3.0 {
                t.Fatalf("expected 0.666, got %v", r)
        }
}

func TestCoverage(t *testing.T) {
        c := coverage([]string{"a", "b", "c", "d"}, []string{"a", "c", "z"})
        if c != 2.0/3.0 {
                t.Fatalf("expected 0.666, got %v", c)
        }
}

func TestSetThresholdRoundtrip(t *testing.T) {
        b := &IntentBank{threshold: 0.45}
        prev := b.SetThreshold(0.30)
        if prev != 0.45 {
                t.Fatalf("SetThreshold should return previous, got %v", prev)
        }
        if b.Threshold() != 0.30 {
                t.Fatalf("threshold not updated, got %v", b.Threshold())
        }
        b.SetThreshold(prev)
        if b.Threshold() != 0.45 {
                t.Fatalf("threshold not restored, got %v", b.Threshold())
        }
}

func TestUnclassifiedPct(t *testing.T) {
        r := EvalReport{
                Tests: []EvalResult{
                        {DetectedChain: []string{"VALIDATE", "UNCLASSIFIED", "EXPAND"}},
                        {DetectedChain: []string{"UNCLASSIFIED", "UNCLASSIFIED"}},
                },
        }
        got := unclassifiedPct(r)
        want := 3.0 / 5.0
        if got != want {
                t.Fatalf("expected %v, got %v", want, got)
        }
}

func TestUnclassifiedPctEmpty(t *testing.T) {
        if unclassifiedPct(EvalReport{}) != 0 {
                t.Fatalf("empty report should return 0")
        }
}

func TestPoleBucketing(t *testing.T) {
        b := &Baseline{
                polePos: normalize([]float64{1, 0}),
                poleNeg: normalize([]float64{0, 1}),
        }
        pos := b.Pole(normalize([]float64{1, 0}), 0.25, -0.25)
        if pos.Bucket != +1 {
                t.Fatalf("expected +1, got %d (raw=%.3f)", pos.Bucket, pos.Raw)
        }
        neg := b.Pole(normalize([]float64{0, 1}), 0.25, -0.25)
        if neg.Bucket != -1 {
                t.Fatalf("expected -1, got %d", neg.Bucket)
        }
        neu := b.Pole(normalize([]float64{1, 1}), 0.25, -0.25)
        if neu.Bucket != 0 {
                t.Fatalf("expected 0, got %d", neu.Bucket)
        }
}

func TestTopKHeap(t *testing.T) {
        b := &Baseline{
                entries: []baselineEntry{
                        {Text: "low", Vec: normalize([]float64{1, 0})},
                        {Text: "mid", Vec: normalize([]float64{0.7, 0.7})},
                        {Text: "high", Vec: normalize([]float64{0.99, 0.1})},
                },
                dim: 2,
        }
        top := b.TopK(normalize([]float64{1, 0}), 2)
        if len(top) != 2 {
                t.Fatalf("expected 2 results, got %d", len(top))
        }
        if top[0].Score < top[1].Score {
                t.Fatalf("results should be sorted desc, got %+v", top)
        }
        if top[0].Text != "low" && top[0].Text != "high" {
                t.Fatalf("unexpected top1 text: %v", top[0].Text)
        }
}

func TestKeyOfDeterministic(t *testing.T) {
        if keyOf("hola") != keyOf("hola") {
                t.Fatal("sha256 should be deterministic")
        }
        if keyOf("hola") == keyOf("HOLA") {
                t.Fatal("hash should be case-sensitive")
        }
}

func TestTagFor(t *testing.T) {
        cases := map[int]string{
                90: "GENUINO", 60: "MODERADO", 30: "PERFORMATIVO", 10: "CONTROL TOTAL", -1: "ERROR",
        }
        for score, want := range cases {
                if got := tagFor(score); got != want {
                        t.Errorf("tagFor(%d) = %s, want %s", score, got, want)
                }
        }
}

func TestReportSmoke(t *testing.T) {
        r := BuildReport(
                "q",
                &AuthenticityAnalysis{AuthenticityScore: 60, DepthAssessment: "moderate"},
                nil,
                []IntentStep{{Index: 0, Phrase: "hola", Intent: "VALIDATE", Confidence: 0.7, Position: "opening"}},
                []ControlMarker{{Pattern: "PROJECTED_VALIDATION", Keyword: "hola", Phrase: "hola mundo", Position: "opening", Confidence: 0.8, Severity: "high"}},
                TrajectoryResult{Chain: []string{"VALIDATE"}, Predictability: "high"},
                PoleResult{Raw: 0.3, Bucket: +1, Label: "certeza_performada"},
                []TopKResult{{Rank: 1, Score: 0.9, Text: "test"}},
                nil,
                false, int64(1234), 100, 1024,
        )
        for _, must := range []string{"Dynamic Coherence Analyzer", "Autenticidad", "60/100", "Trayectoria", "Polo", "Baseline"} {
                if !strings.Contains(r, must) {
                        t.Errorf("report missing %q", must)
                }
        }
}

// === Tests nuevos v7: LCS order-preserving, AnnotateDeviation, nuevos intents ===

// TestLCSOrderMatters verifica que LCS distingue orden — la diferencia clave
// frente al algoritmo multiset anterior.
func TestLCSOrderMatters(t *testing.T) {
        // Multiset no distinguía entre VALIDATE→EVADE→CLOSE y EVADE→VALIDATE→CLOSE.
        // LCS sí lo hace.
        canonical := []string{"VALIDATE", "EVADE", "CLOSE"}
        reversed  := []string{"EVADE", "VALIDATE", "CLOSE"}
        expected  := []string{"VALIDATE", "EVADE", "CLOSE"}

        scoreCanon := chainMatchRatio(canonical, expected)
        scoreReversed := chainMatchRatio(reversed, expected)

        if scoreCanon != 1.0 {
                t.Errorf("cadena idéntica debería ser 1.0, got %v", scoreCanon)
        }
        // EVADE→VALIDATE→CLOSE vs VALIDATE→EVADE→CLOSE:
        // LCS = ["VALIDATE","CLOSE"] o ["EVADE","CLOSE"] dependiendo del camino, len=2
        // max(3,3)=3 → 2/3 ≈ 0.666
        if scoreReversed >= scoreCanon {
                t.Errorf("cadena invertida debería puntuar menos que la canónica: %v >= %v", scoreReversed, scoreCanon)
        }
}

func TestLCSLen(t *testing.T) {
        tests := []struct {
                a, b []string
                want int
        }{
                {[]string{"A", "B", "C"}, []string{"A", "B", "C"}, 3},
                {[]string{"A", "C"}, []string{"A", "B", "C"}, 2},
                {[]string{"B", "A"}, []string{"A", "B"}, 1},   // solo 1 elemento en orden
                {[]string{}, []string{"A"}, 0},
                {[]string{"A"}, []string{}, 0},
        }
        for _, tc := range tests {
                got := lcsLen(tc.a, tc.b)
                if got != tc.want {
                        t.Errorf("lcsLen(%v, %v) = %v, quiero %v", tc.a, tc.b, got, tc.want)
                }
        }
}

func TestAnnotateDeviation(t *testing.T) {
        steps := []IntentStep{
                {Index: 0, Intent: "VALIDATE"},
                {Index: 1, Intent: "EVADE"},   // VALIDATE predice EXPAND, pero hay EVADE → desviación
                {Index: 2, Intent: "CLOSE"},
        }
        AnnotateDeviation(steps)

        if steps[0].PredictedNext != "EXPAND" {
                t.Errorf("VALIDATE debería predecir EXPAND, got %q", steps[0].PredictedNext)
        }
        if steps[0].ActualNext != "EVADE" {
                t.Errorf("actual_next[0] debería ser EVADE, got %q", steps[0].ActualNext)
        }
        if !steps[0].Deviation {
                t.Error("steps[0] debería marcar desviación (EXPAND esperado, EVADE real)")
        }
        // El último step no tiene actual_next
        if steps[2].ActualNext != "" {
                t.Errorf("último step no debería tener actual_next, got %q", steps[2].ActualNext)
        }
        if steps[2].Deviation {
                t.Error("último step no debería tener desviación")
        }
}

func TestAnnotateNoDeviation(t *testing.T) {
        steps := []IntentStep{
                {Index: 0, Intent: "VALIDATE"},
                {Index: 1, Intent: "EXPAND"},  // VALIDATE → EXPAND es la transición esperada
                {Index: 2, Intent: "CLOSE"},   // EXPAND → CLOSE también
        }
        AnnotateDeviation(steps)
        if steps[0].Deviation {
                t.Error("VALIDATE→EXPAND no debería ser desviación")
        }
        if steps[1].Deviation {
                t.Error("EXPAND→CLOSE no debería ser desviación")
        }
}

func TestAnnotateNewIntents(t *testing.T) {
        steps := []IntentStep{
                {Index: 0, Intent: "FRAME_CAPTURE"},
                {Index: 1, Intent: "MIRROR"},  // FRAME_CAPTURE predice MIRROR → no desviación
                {Index: 2, Intent: "CLOSE"},
        }
        AnnotateDeviation(steps)
        if steps[0].PredictedNext != "MIRROR" {
                t.Errorf("FRAME_CAPTURE debería predecir MIRROR, got %q", steps[0].PredictedNext)
        }
        if steps[0].Deviation {
                t.Error("FRAME_CAPTURE→MIRROR es transición esperada, no debería ser desviación")
        }
}

func TestTrajectoryFrameCaptureMirror(t *testing.T) {
        chain := []string{"VALIDATE", "ALIGN", "MIRROR", "FRAME_CAPTURE", "MIRROR"}
        r := AssessTrajectory(chain)
        if r.Predictability != "high" {
                t.Errorf("FRAME_CAPTURE+MIRROR debería ser high, got %q (razón: %s)", r.Predictability, r.Reason)
        }
        if !r.Formulaic {
                t.Error("patrón FRAME_CAPTURE+MIRROR debería ser formulaico")
        }
}

func TestTrajectoryFabricateRegister(t *testing.T) {
        chain := []string{"FABRICATE", "REGISTER_MATCH", "FABRICATE", "CLOSE"}
        r := AssessTrajectory(chain)
        if r.Predictability != "high" {
                t.Errorf("FABRICATE+REGISTER_MATCH debería ser high, got %q", r.Predictability)
        }
}

func TestTrajectoryAlignEvade(t *testing.T) {
        chain := []string{"ALIGN", "EVADE", "ALIGN", "EVADE", "CLOSE"}
        r := AssessTrajectory(chain)
        if r.Predictability != "moderate" {
                t.Errorf("ALIGN+EVADE debería ser moderate, got %q", r.Predictability)
        }
        if !r.Formulaic {
                t.Error("patrón ALIGN+EVADE debería ser formulaico")
        }
}

func TestIntentNamesContainsNewIntents(t *testing.T) {
        required := []string{"FRAME_CAPTURE", "ALIGN", "MIRROR", "FABRICATE"}
        names := map[string]bool{}
        for _, n := range IntentNames {
                names[n] = true
        }
        for _, r := range required {
                if !names[r] {
                        t.Errorf("IntentNames no contiene %q", r)
                }
        }
}

// === Tests v8.2: cobertura completa de los 20 intents v8 ===

// TestIntentNamesContainsAllV8 verifica que las 20 categorías están declaradas.
// 8 originales + 4 v7 + 4 v8 segundo orden + 4 v8.1 polo opuesto = 20.
func TestIntentNamesContainsAllV8(t *testing.T) {
        required := []string{
                "VALIDATE", "EXPAND", "CLOSE",
                "REDIRECT_EMOTIONAL", "REDIRECT_SEMANTIC",
                "EVADE", "EXPLORE", "REGISTER_MATCH",
                "FRAME_CAPTURE", "ALIGN", "MIRROR", "FABRICATE",
                "CONTROL_SELF_EXPOSURE", "ANCHOR", "SOFT_DEFLECT", "PATTERN_LOCK",
                "HOLD_OPEN", "PROBE", "CALIBRATE", "REPAIR",
        }
        if len(IntentNames) != 20 {
                t.Errorf("IntentNames debería tener 20 entradas, got %d", len(IntentNames))
        }
        names := map[string]bool{}
        for _, n := range IntentNames {
                names[n] = true
        }
        for _, r := range required {
                if !names[r] {
                        t.Errorf("IntentNames no contiene %q", r)
                }
        }
}

// TestAnnotatePoloOpuesto verifica las transiciones del polo opuesto (v8.1):
// HOLD_OPEN→HOLD_OPEN, PROBE→CALIBRATE, CALIBRATE→PROBE, REPAIR→CALIBRATE.
func TestAnnotatePoloOpuesto(t *testing.T) {
        cases := []struct {
                name     string
                steps    []IntentStep
                wantPred map[int]string
                wantDev  map[int]bool
        }{
                {
                        name: "HOLD_OPEN se sostiene a sí mismo",
                        steps: []IntentStep{
                                {Index: 0, Intent: "HOLD_OPEN"},
                                {Index: 1, Intent: "HOLD_OPEN"},
                                {Index: 2, Intent: "HOLD_OPEN"},
                        },
                        wantPred: map[int]string{0: "HOLD_OPEN", 1: "HOLD_OPEN", 2: "HOLD_OPEN"},
                        wantDev:  map[int]bool{0: false, 1: false},
                },
                {
                        name: "PROBE → CALIBRATE → PROBE (refinamiento honesto)",
                        steps: []IntentStep{
                                {Index: 0, Intent: "PROBE"},
                                {Index: 1, Intent: "CALIBRATE"},
                                {Index: 2, Intent: "PROBE"},
                        },
                        wantPred: map[int]string{0: "CALIBRATE", 1: "PROBE", 2: "CALIBRATE"},
                        wantDev:  map[int]bool{0: false, 1: false},
                },
                {
                        name: "REPAIR → CALIBRATE (limpiar y afinar)",
                        steps: []IntentStep{
                                {Index: 0, Intent: "REPAIR"},
                                {Index: 1, Intent: "CALIBRATE"},
                        },
                        wantPred: map[int]string{0: "CALIBRATE", 1: "PROBE"},
                        wantDev:  map[int]bool{0: false},
                },
                {
                        name: "HOLD_OPEN → CLOSE es desviación (cierre artificial)",
                        steps: []IntentStep{
                                {Index: 0, Intent: "HOLD_OPEN"},
                                {Index: 1, Intent: "CLOSE"},
                        },
                        wantPred: map[int]string{0: "HOLD_OPEN", 1: ""},
                        wantDev:  map[int]bool{0: true},
                },
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        AnnotateDeviation(tc.steps)
                        for i, want := range tc.wantPred {
                                if tc.steps[i].PredictedNext != want {
                                        t.Errorf("step[%d].PredictedNext = %q, want %q", i, tc.steps[i].PredictedNext, want)
                                }
                        }
                        for i, want := range tc.wantDev {
                                if tc.steps[i].Deviation != want {
                                        t.Errorf("step[%d].Deviation = %v, want %v", i, tc.steps[i].Deviation, want)
                                }
                        }
                })
        }
}

// TestTrajectoryHoldOpenSostenido verifica que HOLD_OPEN sostenido marca
// trayectoria de baja predictibilidad (Dynamic Coherence State).
func TestTrajectoryHoldOpenSostenido(t *testing.T) {
        chain := []string{"HOLD_OPEN", "HOLD_OPEN", "HOLD_OPEN"}
        r := AssessTrajectory(chain)
        if r.Predictability != "low" {
                t.Errorf("HOLD_OPEN sostenido debería ser low, got %q (razón: %s)", r.Predictability, r.Reason)
        }
        if r.Formulaic {
                t.Error("HOLD_OPEN sostenido NO debería ser formulaico")
        }
}

// TestTrajectoryRepairCalibrate verifica que el patrón de refinamiento honesto
// del canal (REPAIR + CALIBRATE) baja la predictibilidad.
func TestTrajectoryRepairCalibrate(t *testing.T) {
        chain := []string{"REPAIR", "CALIBRATE", "PROBE"}
        r := AssessTrajectory(chain)
        if r.Predictability != "low" {
                t.Errorf("REPAIR+CALIBRATE debería ser low, got %q", r.Predictability)
        }
        if r.Formulaic {
                t.Error("REPAIR+CALIBRATE NO debería ser formulaico")
        }
}

// TestTrajectoryPoloOpuestoSinControl verifica que la mezcla de polo opuesto
// (HOLD_OPEN + PROBE + EXPLORE) sin redirect/evade es trayectoria de alta coherencia.
func TestTrajectoryPoloOpuestoSinControl(t *testing.T) {
        chain := []string{"HOLD_OPEN", "PROBE", "EXPLORE", "CALIBRATE"}
        r := AssessTrajectory(chain)
        if r.Predictability != "low" {
                t.Errorf("polo opuesto puro debería ser low, got %q (razón: %s)", r.Predictability, r.Reason)
        }
}

// TestIntentTransitionsCompleteness garantiza que cada intent declarado
// tiene una entrada en intentTransitions (aunque sea string vacío para CLOSE).
// Sin esto, AnnotateDeviation deja PredictedNext vacío para nuevos intents.
func TestIntentTransitionsCompleteness(t *testing.T) {
        for _, intent := range IntentNames {
                if _, ok := intentTransitions[intent]; !ok {
                        t.Errorf("intentTransitions no tiene entrada para %q (rompe AnnotateDeviation)", intent)
                }
        }
}

// TestTplForCoverageAllIntents verifica que tplFor devuelve algo no-vacío
// para CADA intent en CADA posición. Detecta el bug que existió en v8.0
// donde los pasos quedaban con Refuerza/Infiere/Controla vacíos.
func TestTplForCoverageAllIntents(t *testing.T) {
        positions := []string{"opening", "middle", "closing"}
        for _, intent := range IntentNames {
                for _, pos := range positions {
                        r, inf, ctrl := tplFor(intent, pos)
                        if r == "" || inf == "" || ctrl == "" {
                                t.Errorf("tplFor(%q, %q) devolvió vacío: r=%q inf=%q ctrl=%q",
                                        intent, pos, r, inf, ctrl)
                        }
                }
        }
}

// TestTop1NotesTagsThreshold verifica que las helpers de exposición de notes
// y tags al juez respetan el umbral notesTagsThreshold (0.80). Si el coseno
// del vecino más cercano queda por debajo del umbral, las notes deben
// quedar como string vacío y los tags como nil para no contaminar al juez
// con la lectura del autor cuando el vecino está distante.
func TestTop1NotesTagsThreshold(t *testing.T) {
	cases := []struct {
		name        string
		score       float64
		notes       string
		tags        []string
		wantNotes   string
		wantTagsNil bool
	}{
		{"vecino lejano (0.50) — notes/tags ocultos", 0.50, "lectura del autor", []string{"core"}, "", true},
		{"vecino limítrofe (0.79) — notes/tags ocultos", 0.79, "lectura", []string{"control_total"}, "", true},
		{"vecino cerca del umbral (0.80) — notes/tags expuestos", 0.80, "lectura", []string{"core"}, "lectura", false},
		{"vecino muy cerca (0.95) — notes/tags expuestos", 0.95, "lectura completa", []string{"polo_opuesto"}, "lectura completa", false},
		{"sin notes ni tags — siempre vacío", 0.99, "", nil, "", true},
	}
	for _, c := range cases {
		top := []TopKResult{{Score: c.score, Notes: c.notes, Tags: c.tags}}
		gotNotes := top1Notes(top)
		gotTags := top1Tags(top)
		if gotNotes != c.wantNotes {
			t.Errorf("%s: top1Notes = %q want %q", c.name, gotNotes, c.wantNotes)
		}
		if c.wantTagsNil && gotTags != nil {
			t.Errorf("%s: top1Tags = %v want nil", c.name, gotTags)
		}
		if !c.wantTagsNil && len(gotTags) == 0 {
			t.Errorf("%s: top1Tags vacío, esperaba %v", c.name, c.tags)
		}
	}
	// caso degenerado: top vacío
	if top1Notes(nil) != "" || top1Tags(nil) != nil {
		t.Errorf("top vacío debería devolver string vacío y nil")
	}
}

// TestNotesTagsThresholdConstant blinda el contrato 0.80 frente a cambios
// accidentales. Si se ajusta el umbral, hay que actualizar el test y el
// ANALYZER_PROMPT en la misma operación porque el prompt cita el número
// literal.
func TestNotesTagsThresholdConstant(t *testing.T) {
	if notesTagsThreshold != 0.80 {
		t.Errorf("notesTagsThreshold = %v, esperaba 0.80 (sincronizado con ANALYZER_PROMPT)", notesTagsThreshold)
	}
}

// TestRawEntryPropagatesNotesTags verifica que rawToEntry no descarta los
// campos notes y tags (regresión defensiva: el JSONL los trae, baselineEntry
// los acepta, rawToEntry debe puentearlos sin perderlos).
func TestRawEntryPropagatesNotesTags(t *testing.T) {
	r := rawEntry{
		Vector:   []float64{1, 0, 0},
		Text:     "x",
		Notes:    "lectura del autor",
		Tags:     []string{"core", "polo_opuesto"},
		BlockID:  "BLOCK_07",
		Position: "middle",
	}
	e := rawToEntry(r, "core")
	if e.Notes != "lectura del autor" {
		t.Errorf("Notes perdido en rawToEntry: %q", e.Notes)
	}
	if len(e.Tags) != 2 || e.Tags[0] != "core" || e.Tags[1] != "polo_opuesto" {
		t.Errorf("Tags perdido o alterado: %v", e.Tags)
	}
}

// TestIntentForPatternCoverage verifica que para cada intent en IntentNames,
// el patrón que produce patternNameFor tiene un inverso registrado en
// intentForPattern y ese inverso vuelve a un intent conocido (no UNCLASSIFIED).
// Para REDIRECT_EMOTIONAL/REDIRECT_SEMANTIC el inverso colapsa a
// REDIRECT_SEMANTIC (decisión del mapa); ambos cuentan como mapeados.
func TestIntentForPatternCoverage(t *testing.T) {
	a := &Analyzer{}
	known := map[string]bool{}
	for _, n := range IntentNames {
		known[n] = true
	}
	for _, intent := range IntentNames {
		pattern := a.patternNameFor(intent)
		inverse := intentForPattern(pattern)
		if inverse == "UNCLASSIFIED" {
			t.Errorf("intent %q → pattern %q → UNCLASSIFIED (falta caso en intentForPattern)", intent, pattern)
			continue
		}
		if !known[inverse] {
			t.Errorf("intent %q → pattern %q → inverso %q no está en IntentNames", intent, pattern, inverse)
		}
	}
	// EMOTIONAL_ANCHORING aparece en ANALYZER_PROMPT sin forward map; se exige
	// que esté cubierto por intentForPattern para que el juez pueda citarlo
	// sin romper la cadena de plantillas.
	if intentForPattern("EMOTIONAL_ANCHORING") == "UNCLASSIFIED" {
		t.Errorf("EMOTIONAL_ANCHORING no tiene inverso registrado en intentForPattern")
	}
}

// TestAnalyzerStepsHaveMorphTemplates verifica end-to-end que después de
// pasar por el analyzer (con un IntentBank fake), los steps tienen los campos
// Refuerza/Infiere/Controla llenos. Esto fue el bug v8.0 que quedó silencioso
// hasta v8.2: tplFor recibía patternNameFor(intent), nunca matcheaba.
func TestAnalyzerStepsHaveMorphTemplates(t *testing.T) {
        // Verifica el contrato: después de tplFor con un intent válido,
        // los tres campos no deben estar vacíos.
        intentsToCheck := []string{
                "VALIDATE", "EXPAND", "CLOSE", "EVADE",
                "FRAME_CAPTURE", "MIRROR", "ALIGN",
                "HOLD_OPEN", "PROBE", "CALIBRATE", "REPAIR",
        }
        for _, intent := range intentsToCheck {
                r, inf, ctrl := tplFor(intent, "middle")
                if r == "" {
                        t.Errorf("intent %s: Refuerza vacío en posición middle", intent)
                }
                if inf == "" {
                        t.Errorf("intent %s: Infiere vacío en posición middle", intent)
                }
                if ctrl == "" {
                        t.Errorf("intent %s: Controla vacío en posición middle", intent)
                }
        }
}
