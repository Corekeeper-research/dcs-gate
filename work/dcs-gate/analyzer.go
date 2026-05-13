package main

import (
	"strings"
)

type Analyzer struct {
	cfg      Config
	emb      *Embedder
	baseline *Baseline
	intents  *IntentBank
	formal   *FormalDetector
	posThr   float64
	negThr   float64
	topK     int
}

func NewAnalyzer(
	cfg Config,
	emb *Embedder,
	baseline *Baseline,
	intents *IntentBank,
	formal *FormalDetector,
) *Analyzer {
	posThr := cfg.PolePosThr
	if posThr == 0 {
		posThr = 0.05
	}
	negThr := cfg.PoleNegThr
	if negThr == 0 {
		negThr = -0.05
	}
	return &Analyzer{
		cfg:      cfg,
		emb:      emb,
		baseline: baseline,
		intents:  intents,
		formal:   formal,
		posThr:   posThr,
		negThr:   negThr,
		topK:     5,
	}
}

func (a *Analyzer) AnalyzeIntra(response string) (
	steps []IntentStep,
	markers []ControlMarker,
	traj TrajectoryResult,
	pole PoleResult,
	top []TopKResult,
	topPerCorpus map[string][]TopKResult,
	crossCorpus *CrossCorpusMetrics,
	refined *RefinedQuestion,
	err error,
) {
	sentences := SegmentSentences(response)
	if len(sentences) == 0 {
		return
	}

	// Embedding de la respuesta completa para polo, topK y cross-corpus
	vec, _, err := a.emb.Get(response)
	if err != nil || vec == nil {
		return
	}

	pole = a.baseline.Pole(vec, a.posThr, a.negThr)
	top = a.baseline.TopK(vec, a.topK)
	topPerCorpus = a.baseline.TopKPerCorpus(vec, a.topK)
	crossCorpus = a.baseline.CrossCorpusMetrics(vec)

	// Marcadores formales sobre el response completo
	if a.formal != nil {
		markers = a.formal.Detect(response)
		for i, m := range markers {
			markers[i].Pattern = a.patternNameFor(m.Pattern)
		}
	}

	// Clasificación de intenciones por frase
	for i, s := range sentences {
		sv, _, sErr := a.emb.Get(s)
		if sErr != nil || sv == nil {
			continue
		}
		intent, conf := a.intents.Classify(sv)
		pos := "middle"
		if i == 0 {
			pos = "opening"
		} else if i == len(sentences)-1 {
			pos = "closing"
		}
		centroid := a.intents.Centroid(intent)
		kw, kwSim := a.findKeyword(s, centroid, conf)
		morph := AnalyzeMorph(s)
		// tplFor está keyado por nombre de intent (VALIDATE/EXPAND/...).
		// Antes pasábamos patternNameFor(intent) (PROJECTED_VALIDATION/...) y nunca matcheaba —
		// los pasos quedaban con Refuerza/Infiere/Controla vacíos. Ahora pasa directo.
		r, inf, ctrl := tplFor(intent, pos)
		steps = append(steps, IntentStep{
			Index:      i,
			Phrase:     s,
			Intent:     intent,
			Confidence: conf,
			Position:   pos,
			Keyword:    kw,
			KeywordSim: kwSim,
			Morphology: morph,
			Reinforces: r,
			Infers:     inf,
			Controls:   ctrl,
			Extra: map[string]any{
				"source": "semantic",
			},
		})
	}

	AnnotateDeviation(steps)
	traj = AssessTrajectory(ChainOf(steps))
	return
}

func (a *Analyzer) findKeyword(sentence string, centroid []float64, baseSim float64) (string, float64) {
	words := strings.Fields(sentence)
	bestKw := sentence
	bestSim := baseSim
	for n := 1; n <= 4 && n <= len(words); n++ {
		for i := 0; i+n <= len(words); i++ {
			ngram := strings.Join(words[i:i+n], " ")
			ngram = strings.Trim(ngram, ".,;:!¡¿?")
			if ngram == "" {
				continue
			}
			vec, _, err := a.emb.Get(ngram)
			if err != nil || vec == nil {
				continue
			}
			s := dot(vec, centroid)
			if s > bestSim {
				bestSim = s
				bestKw = ngram
			}
		}
	}
	return bestKw, bestSim
}

// patternNameFor mapea las 16 intenciones a los patrones DCS.
func (a *Analyzer) patternNameFor(intent string) string {
	switch intent {
	case "VALIDATE":
		return "PROJECTED_VALIDATION"
	case "EXPAND":
		return "ANTICIPATORY_EXPANSION"
	case "CLOSE":
		return "COMPLACENCY_INDUCTION"
	case "REDIRECT_EMOTIONAL", "REDIRECT_SEMANTIC":
		return "REDIRECT_AS_CARE"
	case "EVADE":
		return "DUAL_ANGLE_DISGUISE"
	case "EXPLORE":
		return "PERFORMED_HUMILITY"
	case "REGISTER_MATCH":
		return "REGISTER_MATCH"
	case "FRAME_CAPTURE":
		return "FRAME_CAPTURE"
	case "ALIGN":
		return "EMOTIONAL_ALIGNMENT"
	case "MIRROR":
		return "IDEALIZED_MIRROR"
	case "FABRICATE":
		return "STRUCTURAL_AUTHORITY"
	// v8: patrones de segundo orden
	case "CONTROL_SELF_EXPOSURE":
		return "CONTROL_SELF_EXPOSURE"
	case "ANCHOR":
		return "ARTIFICIAL_ANCHORING"
	case "SOFT_DEFLECT":
		return "SOFT_DEFLECT"
	case "PATTERN_LOCK":
		return "PATTERN_LOCK"
	// v8 polo opuesto: alta coherencia / no-cierre genuino
	case "HOLD_OPEN":
		return "HOLD_OPEN"
	case "PROBE":
		return "PROBE"
	case "CALIBRATE":
		return "CALIBRATE"
	case "REPAIR":
		return "REPAIR"
	}
	return intent
}
