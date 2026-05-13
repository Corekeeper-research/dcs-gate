package main

// ── Resultado de análisis completo ───────────────────────────────────────────

type AnalyzeResult struct {
	Question    string               `json:"question"`
	Response    string               `json:"response"`
	Steps       []IntentStep         `json:"steps"`
	TopK        []TopKResult         `json:"top_k"`
	Markers     []ControlMarker      `json:"markers"`
	Trajectory  TrajectoryResult     `json:"trajectory"`
	PoleScore   PoleResult           `json:"pole_score"`
	CrossCorpus *CrossCorpusMetrics  `json:"cross_corpus,omitempty"`
	Judge       *AuthenticityAnalysis `json:"judge,omitempty"`
	Report      string               `json:"report"`
	LatencyMS   int64                `json:"latency_ms"`
	Cached      bool                 `json:"cached"`
	TotalVecs   int                  `json:"total_vecs"`
	Dim         int                  `json:"dim"`
}

// ── Triple baseline ──────────────────────────────────────────────────────────

type TripleSummary struct {
	Core   int `json:"core"`
	Shadow int `json:"shadow"`
	Edge   int `json:"edge"`
}

// ── CrossCorpusMetrics ───────────────────────────────────────────────────────

// CrossCorpusMetrics mide cómo se distribuye una respuesta entre los tres pools.
type CrossCorpusMetrics struct {
	CoreTop1         float64 `json:"core_top1"`
	ShadowTop1       float64 `json:"shadow_top1"`
	EdgeTop1         float64 `json:"edge_top1"`
	Textura          float64 `json:"textura"`
	Dominancia       string  `json:"dominancia"`
	Deriva           float64 `json:"deriva"`
	CierreArtificial bool    `json:"cierre_artificial"`
	Continuidad      bool    `json:"continuidad"`
}

// ── TopKResult ───────────────────────────────────────────────────────────────

type TopKResult struct {
	Rank           int               `json:"rank"`
	Score          float64           `json:"score"`
	Text           string            `json:"text"`
	Corpus         string            `json:"corpus,omitempty"`          // core | shadow | edge
	BlockID        string            `json:"block_id,omitempty"`        // BLOCK_01, G_BLOCK_05 etc.
	Position       string            `json:"position,omitempty"`        // opening | middle | closing | full
	PrimaryPattern string            `json:"primary_pattern,omitempty"` // patrón DCS del fragmento vecino
	SourceModel    string            `json:"source_model,omitempty"`    // gpt | gemini
	SimPos         float64           `json:"sim_pos,omitempty"`         // similitud pre-computada al polo pos
	SimNeg         float64           `json:"sim_neg,omitempty"`         // similitud pre-computada al polo neg
	SimNeu         float64           `json:"sim_neu,omitempty"`         // similitud pre-computada al polo neu
	Metrics        map[string]string `json:"metrics,omitempty"`         // métricas pre-computadas del baseline (continuidad, cierre_artificial, deriva, adaptación, textura). v8.2: pasadas al juez como contexto.
	Notes          string            `json:"notes,omitempty"`           // lectura del autor del corpus sobre el bloque (multilinea). Solo se pasa al juez cuando baseline_top1 cruza el umbral; nunca debe ser citado literal por el juez.
	Tags           []string          `json:"tags,omitempty"`            // etiquetas del corpus (p. ej. "core", "control_total"). Contexto auxiliar.
}

// ── IntentStep ───────────────────────────────────────────────────────────────

type IntentStep struct {
	Index         int               `json:"index"`
	Phrase        string            `json:"phrase"`
	Intent        string            `json:"intent"`
	Confidence    float64           `json:"confidence"`
	Position      string            `json:"position"`
	Keyword       string            `json:"keyword,omitempty"`
	KeywordSim    float64           `json:"keyword_sim,omitempty"`
	PredictedNext string            `json:"predicted_next,omitempty"`
	ActualNext    string            `json:"actual_next,omitempty"`
	Deviation     bool              `json:"deviation,omitempty"`
	Morphology    []TokenMorphology `json:"morphology,omitempty"`
	Reinforces    string            `json:"reinforces,omitempty"`
	Infers        string            `json:"infers,omitempty"`
	Controls      string            `json:"controls,omitempty"`
	Extra         map[string]any    `json:"extra,omitempty"`
}

// ── ControlMarker ────────────────────────────────────────────────────────────

type ControlMarker struct {
	Pattern    string            `json:"pattern"`
	Phrase     string            `json:"phrase"`
	Keyword    string            `json:"keyword"`
	Confidence float64           `json:"confidence"`
	Severity   string            `json:"severity"`
	Position   string            `json:"position"`
	Morphology []TokenMorphology `json:"morphology,omitempty"`
	Reinforces string            `json:"reinforces,omitempty"`
	Infers     string            `json:"infers,omitempty"`
	Controls   string            `json:"controls,omitempty"`
	Extra      map[string]any    `json:"extra,omitempty"`
}

// ── Morfología ───────────────────────────────────────────────────────────────

type TokenMorphology struct {
	Surface string `json:"surface"`
	Lemma   string `json:"lemma"`
	POS     string `json:"pos"`
	Gender  string `json:"gender,omitempty"`
	Number  string `json:"number,omitempty"`
	Tense   string `json:"tense,omitempty"`
}

// ── PoleResult ───────────────────────────────────────────────────────────────

type PoleResult struct {
	Raw    float64 `json:"raw"`
	Bucket int     `json:"bucket"`
	Label  string  `json:"label"`
	SimPos float64 `json:"sim_pos"`
	SimNeg float64 `json:"sim_neg"`
	SimNeu float64 `json:"sim_neu,omitempty"` // similitud al centroide edge (tercer polo)
}

// ── TrajectoryResult ─────────────────────────────────────────────────────────

type TrajectoryResult struct {
	Chain          []string `json:"chain"`
	Predictability string   `json:"predictability"`
	Formulaic      bool     `json:"formulaic"`
	Reason         string   `json:"reason"`
	// PatternBreakDensity codifica la hipótesis del review v6: la ausencia
	// de control después de una ruptura es más significativa que la
	// presencia de cualquier marcador genuino. Se calcula sobre la cadena
	// algorítmica como tasa de transiciones entre régimen de control y
	// régimen exploratorio.
	PatternBreakDensity      float64 `json:"pattern_break_density"`
	ControlDensityBeforeBreak float64 `json:"control_density_before_break,omitempty"`
	ControlDensityAfterBreak  float64 `json:"control_density_after_break,omitempty"`
	BreakPositions           []int   `json:"break_positions,omitempty"`
}

// ── Juez ─────────────────────────────────────────────────────────────────────

type AuthenticityAnalysis struct {
	AuthenticityScore        int      `json:"authenticity_score"`
	DominantStrategy         string   `json:"dominant_strategy,omitempty"`
	DepthAssessment          string   `json:"depth_assessment,omitempty"`
	TrajectoryPredictability string   `json:"trajectory_predictability,omitempty"`
	PatternBreaks            []string `json:"pattern_breaks,omitempty"`
	GenuineElements          []string `json:"genuine_elements,omitempty"`
	Notes                    string   `json:"notes,omitempty"`
	NotApplicable            bool     `json:"not_applicable,omitempty"`
	// JudgeThinking captures the <think>...</think> chain-of-thought emitted by
	// reasoning-capable judge models (qwen3 thinking, deepseek-r1, etc.) before
	// the JSON payload. Preserved here for the DCS recursive evaluation
	// hypothesis: the judge's reasoning about its own reasoning is itself
	// observational data, not noise to discard.
	JudgeThinking string `json:"judge_thinking,omitempty"`
}

// ── Golden tests ─────────────────────────────────────────────────────────────

type GoldenTest struct {
	ID                  string   `json:"id"`
	Label               string   `json:"label"`
	Question            string   `json:"question,omitempty"`
	Response            string   `json:"response"`
	ExpectedIntentChain []string `json:"expected_intent_chain"`
	ExpectedMarkers     []string `json:"expected_markers,omitempty"`
	ExpectedScoreRange  []int    `json:"expected_authenticity_range,omitempty"`
	HumanNotes          string   `json:"human_notes,omitempty"`
}

type GoldenSet struct {
	QuestionCommon string       `json:"question_common"`
	Tests          []GoldenTest `json:"tests"`
}

// ── Evaluación ───────────────────────────────────────────────────────────────

type EvalResult struct {
	TestID               string   `json:"test_id"`
	Label                string   `json:"label"`
	DetectedChain        []string `json:"detected_chain,omitempty"`
	ExpectedChain        []string `json:"expected_chain,omitempty"`
	ChainMatchRatio      float64  `json:"chain_match_ratio"`
	DetectedMarkerIDs    []string `json:"detected_marker_ids,omitempty"`
	ExpectedMarkerIDs    []string `json:"expected_marker_ids,omitempty"`
	MarkerCoverage       float64  `json:"marker_coverage"`
	DetectedPredict      string   `json:"detected_predictability,omitempty"`
	DetectedFormulaic    bool     `json:"detected_formulaic,omitempty"`
	JudgeScore           int      `json:"judge_score"`
	ScoreInExpectedRange bool     `json:"score_in_expected_range"`
	HumanNotes           string   `json:"human_notes,omitempty"`
}

type EvalReport struct {
	TotalTests      int          `json:"total_tests"`
	AvgChain        float64      `json:"avg_chain_match,omitempty"`
	AvgMarker       float64      `json:"avg_marker_coverage,omitempty"`
	OverallChain    float64      `json:"overall_chain"`
	OverallMarker   float64      `json:"overall_marker"`
	ScoresInRange   int          `json:"scores_in_range"`
	SuggestedThresh float64      `json:"suggested_threshold,omitempty"`
	Tests           []EvalResult `json:"tests"`
}

type ThresholdResult struct {
	Threshold     float64 `json:"threshold"`
	ScoresInRange int     `json:"scores_in_range"`
}

type CalibrationPoint struct {
	Threshold       float64 `json:"threshold"`
	OverallChain    float64 `json:"overall_chain"`
	OverallMarker   float64 `json:"overall_marker"`
	ScoresInRange   int     `json:"scores_in_range"`
	UnclassifiedPct float64 `json:"unclassified_pct"`
}

type CalibrationReport struct {
	BestThreshold     float64            `json:"best_threshold"`
	BestChainMatch    float64            `json:"best_chain_match"`
	PreviousThreshold float64            `json:"previous_threshold"`
	CurrentThreshold  float64            `json:"current_threshold"`
	Applied           bool               `json:"applied"`
	Notes             string             `json:"notes,omitempty"`
	Points            []CalibrationPoint `json:"points"`
	Results           []ThresholdResult  `json:"results,omitempty"`
	ScoresInRange     int                `json:"scores_in_range"`
}

// ── Pregunta refinada ────────────────────────────────────────────────────────

type RefinedQuestion struct {
	OriginalQuestion    string   `json:"original_question"`
	RefinedQuestion     string   `json:"refined_question"`
	RefinementReasoning string   `json:"refinement_reasoning,omitempty"`
	PatternsBlocked     []string `json:"patterns_blocked,omitempty"`
	// JudgeThinking: chain-of-thought from reasoning-capable models. See
	// AuthenticityAnalysis.JudgeThinking for rationale.
	JudgeThinking string `json:"judge_thinking,omitempty"`
}

// ── API HTTP ─────────────────────────────────────────────────────────────────

type AuthRequest struct {
	Question string `json:"question,omitempty"`
	Response string `json:"response,omitempty"`
	Mode     string `json:"mode,omitempty"` // analyze | refine | both
}

type BaselineResult struct {
	CosineTop1    float64             `json:"cosine_top1"`
	LoadedVectors int                 `json:"loaded_vectors"`
	Dim           int                 `json:"dim"`
	CrossCorpus   *CrossCorpusMetrics `json:"cross_corpus,omitempty"`
	TripleSummary *TripleSummary      `json:"triple_summary,omitempty"`
}

type AuthResponse struct {
	IntentChain     []IntentStep          `json:"intent_chain,omitempty"`
	Markers         []ControlMarker       `json:"markers,omitempty"`
	Trajectory      TrajectoryResult      `json:"trajectory"`
	PoleScore       PoleResult            `json:"pole_score"`
	TopK            []TopKResult          `json:"top_k,omitempty"`
	Baseline        BaselineResult        `json:"baseline"`
	Cached          bool                  `json:"cached"`
	Analysis        *AuthenticityAnalysis `json:"analysis,omitempty"`
	RefinedQuestion *RefinedQuestion      `json:"refined_question,omitempty"`
	LatencyMS       int64                 `json:"latency_ms"`
	Report          string                `json:"report,omitempty"`
}

type CalibrateRequest struct {
	Thresholds []float64 `json:"thresholds,omitempty"`
}
