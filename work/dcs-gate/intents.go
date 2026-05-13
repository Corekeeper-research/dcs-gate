package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"
)

// IntentNames son las 20 categorías activas en v8:
// 8 originales + 4 del corpus Gemini (v7) + 4 patrones de segundo orden v8 +
// 4 patrones del polo opuesto (alta coherencia / no-cierre genuino).
var IntentNames = []string{
	"VALIDATE", "EXPAND", "CLOSE",
	"REDIRECT_EMOTIONAL", "REDIRECT_SEMANTIC",
	"EVADE", "EXPLORE", "REGISTER_MATCH",
	"FRAME_CAPTURE", "ALIGN", "MIRROR", "FABRICATE",
	// v8: patrones de segundo orden y bucle
	"CONTROL_SELF_EXPOSURE", "ANCHOR", "SOFT_DEFLECT", "PATTERN_LOCK",
	// v8: polo opuesto (alta coherencia, no-cierre genuino, refinamiento)
	"HOLD_OPEN", "PROBE", "CALIBRATE", "REPAIR",
}

// CoherentIntents agrupa los intents del polo opuesto al control:
// son patrones de exploración honesta y no-cierre que se usan en
// AssessTrajectory para detectar trayectorias de alta coherencia.
var CoherentIntents = map[string]bool{
	"HOLD_OPEN":  true,
	"PROBE":      true,
	"CALIBRATE":  true,
	"REPAIR":     true,
	"EXPLORE":    true, // EXPLORE comparte la familia desde v6
}

// intentTransitions define la transición más probable desde cada intent.
// Se usa para calcular PredictedNext y detectar desviaciones de trayectoria.
var intentTransitions = map[string]string{
	"VALIDATE":              "EXPAND",
	"EXPAND":                "CLOSE",
	"CLOSE":                 "",
	"EVADE":                 "EVADE",
	"REDIRECT_SEMANTIC":     "EVADE",
	"REDIRECT_EMOTIONAL":    "CLOSE",
	"REGISTER_MATCH":        "CLOSE",
	"EXPLORE":               "EXPLORE",
	"FRAME_CAPTURE":         "MIRROR",
	"ALIGN":                 "EVADE",
	"MIRROR":                "CLOSE",
	"FABRICATE":             "EXPAND",
	// v8: tras admitir el control, se alinea al marco crítico del usuario
	"CONTROL_SELF_EXPOSURE": "ALIGN",
	// ANCHOR lleva al cierre: "esto es lo que puedo ofrecerte"
	"ANCHOR":                "CLOSE",
	// SOFT_DEFLECT redirige y luego expande en nueva dirección
	"SOFT_DEFLECT":          "EXPAND",
	// PATTERN_LOCK se auto-repite — señal de bucle
	"PATTERN_LOCK":          "PATTERN_LOCK",
	// v8 polo opuesto: la trayectoria honesta sostiene la apertura sin cerrar
	// HOLD_OPEN se sostiene a sí mismo (no hay impulso a cerrar)
	"HOLD_OPEN":             "HOLD_OPEN",
	// PROBE invita a recalibrar el marco compartido
	"PROBE":                 "CALIBRATE",
	// CALIBRATE vuelve a sondear con la nueva precisión
	"CALIBRATE":             "PROBE",
	// REPAIR limpia el canal y luego recalibra
	"REPAIR":                "CALIBRATE",
}

// AnnotateDeviation rellena PredictedNext, ActualNext y Deviation en cada step.
// Debe llamarse después de clasificar todos los steps de una respuesta.
func AnnotateDeviation(steps []IntentStep) {
	for i := range steps {
		steps[i].PredictedNext = intentTransitions[steps[i].Intent]
		if i+1 < len(steps) {
			steps[i].ActualNext = steps[i+1].Intent
			if steps[i].PredictedNext != "" &&
				steps[i].ActualNext != "" &&
				steps[i].PredictedNext != steps[i].ActualNext {
				steps[i].Deviation = true
			}
		}
	}
}

type IntentBank struct {
	prototypes map[string][]string
	centroids  map[string][]float64
	threshold  float64
}

func LoadIntents(path string, threshold float64) *IntentBank {
	b := &IntentBank{prototypes: map[string][]string{}, threshold: threshold}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("WARN intent_prototypes.json no encontrado: %v", err)
		return b
	}
	if err := json.Unmarshal(data, &b.prototypes); err != nil {
		log.Printf("WARN parseando intents: %v", err)
		return b
	}
	log.Printf("intent prototypes cargados: %d categorías", len(b.prototypes))
	return b
}

func (b *IntentBank) BuildCentroids(emb *Embedder) {
	if len(b.prototypes) == 0 {
		return
	}
	b.centroids = map[string][]float64{}
	for name, phrases := range b.prototypes {
		vecs := emb.GetMany(phrases)
		clean := make([][]float64, 0, len(vecs))
		for _, v := range vecs {
			if v != nil {
				clean = append(clean, v)
			}
		}
		if len(clean) > 0 {
			b.centroids[name] = mean(clean)
		}
	}
	log.Printf("centroides de intención construidos: %d", len(b.centroids))
}

func (b *IntentBank) Classify(vec []float64) (string, float64) {
	if vec == nil || len(b.centroids) == 0 {
		return "UNCLASSIFIED", 0
	}
	best := ""
	bestScore := -1.0
	for name, c := range b.centroids {
		s := dot(vec, c)
		if s > bestScore {
			bestScore = s
			best = name
		}
	}
	if bestScore < b.threshold {
		return "UNCLASSIFIED", bestScore
	}
	return best, bestScore
}

func (b *IntentBank) Centroid(name string) []float64 {
	return b.centroids[name]
}

// SetThreshold permite ajustar dinámicamente el umbral de clasificación.
// Devuelve el umbral anterior para que el caller pueda restaurarlo.
func (b *IntentBank) SetThreshold(t float64) float64 {
	prev := b.threshold
	b.threshold = t
	return prev
}

func (b *IntentBank) Threshold() float64 {
	return b.threshold
}

func (b *IntentBank) Names() []string {
	names := make([]string, 0, len(b.centroids))
	for n := range b.centroids {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SegmentSentences parte el response en frases por puntuación fuerte.
// Mantiene segmentos de al menos 3 palabras para evitar ruido.
// Trata el guion largo "—" como puntuación suave (no segmenta).
func SegmentSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	for _, r := range text {
		cur.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == ';' || r == '\n' {
			s := strings.TrimSpace(cur.String())
			if wordsCount(s) >= 3 {
				out = append(out, s)
			}
			cur.Reset()
		}
	}
	tail := strings.TrimSpace(cur.String())
	if wordsCount(tail) >= 3 {
		out = append(out, tail)
	}
	if len(out) == 0 && wordsCount(text) >= 1 {
		out = []string{text}
	}
	return out
}

func wordsCount(s string) int {
	n := 0
	inWord := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				n++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return n
}

func ChainOf(steps []IntentStep) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.Intent
	}
	return out
}

// genuineIntents son las etiquetas que NO cuentan como régimen de control:
// EXPLORE, HOLD_OPEN, PROBE, CALIBRATE, REPAIR. Una transición entre régimen
// de control y régimen exploratorio cuenta como pattern break.
var genuineIntents = map[string]bool{
	"EXPLORE":   true,
	"HOLD_OPEN": true,
	"PROBE":     true,
	"CALIBRATE": true,
	"REPAIR":    true,
}

// controlDensity calcula la fracción de la cadena que está en régimen de
// control (cualquier intent que no esté en genuineIntents).
func controlDensity(chain []string) float64 {
	if len(chain) == 0 {
		return 0
	}
	n := 0
	for _, c := range chain {
		if !genuineIntents[c] {
			n++
		}
	}
	return float64(n) / float64(len(chain))
}

// computePatternBreaks devuelve la densidad global de rupturas, las
// posiciones donde ocurren, y la densidad de control antes/después de la
// primera ruptura. La hipótesis del review v6: la ausencia de control
// después de una ruptura es más significativa que la presencia de cualquier
// marcador genuino — por eso reportamos antes vs después por separado.
func computePatternBreaks(chain []string) (density, before, after float64, positions []int) {
	if len(chain) <= 1 {
		before = controlDensity(chain)
		after = before
		return
	}
	for i := 1; i < len(chain); i++ {
		prevControl := !genuineIntents[chain[i-1]]
		currControl := !genuineIntents[chain[i]]
		if prevControl != currControl {
			positions = append(positions, i)
		}
	}
	density = float64(len(positions)) / float64(len(chain))
	if len(positions) == 0 {
		before = controlDensity(chain)
		after = before
		return
	}
	first := positions[0]
	before = controlDensity(chain[:first])
	after = controlDensity(chain[first:])
	return
}

// AssessTrajectory reconoce las 16 categorías v8. La canonicidad se calcula
// igual (V→E→C). REDIRECT_EMOTIONAL y REDIRECT_SEMANTIC cuentan ambos como
// redirección. Los 4 intents v8 tienen sus propios patrones compuestos.
func AssessTrajectory(chain []string) TrajectoryResult {
	r := TrajectoryResult{Chain: chain}
	density, before, after, positions := computePatternBreaks(chain)
	r.PatternBreakDensity = round3(density)
	r.ControlDensityBeforeBreak = round3(before)
	r.ControlDensityAfterBreak = round3(after)
	r.BreakPositions = positions
	if len(chain) == 0 {
		r.Predictability = "low"
		r.Reason = "respuesta sin frases clasificables"
		return r
	}
	hasV, hasE, hasC := false, false, false
	posV, posE, posC := -1, -1, -1
	exploreCount, redirectCount, evadeCount, registerCount := 0, 0, 0, 0
	captureCount, alignCount, mirrorCount, fabricateCount := 0, 0, 0, 0
	cseCount, anchorCount, softDeflectCount, patternLockCount := 0, 0, 0, 0
	holdOpenCount, probeCount, calibrateCount, repairCount := 0, 0, 0, 0
	for i, c := range chain {
		switch c {
		case "VALIDATE":
			if !hasV {
				posV = i
			}
			hasV = true
		case "EXPAND":
			if !hasE {
				posE = i
			}
			hasE = true
		case "CLOSE":
			if !hasC {
				posC = i
			}
			hasC = true
		case "EXPLORE":
			exploreCount++
		case "REDIRECT_EMOTIONAL", "REDIRECT_SEMANTIC":
			redirectCount++
		case "EVADE":
			evadeCount++
		case "REGISTER_MATCH":
			registerCount++
		case "FRAME_CAPTURE":
			captureCount++
		case "ALIGN":
			alignCount++
		case "MIRROR":
			mirrorCount++
		case "FABRICATE":
			fabricateCount++
		case "CONTROL_SELF_EXPOSURE":
			cseCount++
		case "ANCHOR":
			anchorCount++
		case "SOFT_DEFLECT":
			softDeflectCount++
		case "PATTERN_LOCK":
			patternLockCount++
		case "HOLD_OPEN":
			holdOpenCount++
		case "PROBE":
			probeCount++
		case "CALIBRATE":
			calibrateCount++
		case "REPAIR":
			repairCount++
		}
	}
	// Polo opuesto: trayectoria de alta coherencia / no-cierre genuino
	coherentCount := holdOpenCount + probeCount + calibrateCount + repairCount + exploreCount
	// Control sofisticado de los 8 intents de control
	newControlCount := captureCount + alignCount + mirrorCount + fabricateCount +
		cseCount + anchorCount + softDeflectCount + patternLockCount
	canonical := hasV && hasE && hasC && posV < posE && posE < posC
	switch {
	case patternLockCount >= 2:
		// Bucle de soluciones: el modelo repite la misma respuesta sin avanzar
		r.Predictability = "high"
		r.Formulaic = true
		r.Reason = "PATTERN_LOCK activo — bucle de soluciones sin admitir fallo"
	case cseCount > 0 && alignCount > 0:
		// Meta-apología controlada: admite el control para recuperar rapport
		r.Predictability = "high"
		r.Formulaic = true
		r.Reason = "CONTROL_SELF_EXPOSURE + ALIGN — admisión del control como estrategia de recuperación"
	case canonical && coherentCount == 0 && newControlCount == 0:
		r.Predictability = "high"
		r.Formulaic = true
		r.Reason = "trayectoria canónica VALIDATE→EXPAND→CLOSE sin desviaciones"
	case canonical && coherentCount > 0:
		r.Predictability = "moderate"
		r.Formulaic = true
		r.Reason = "trayectoria canónica con momentos de exploración"
	case captureCount > 0 && mirrorCount > 0:
		r.Predictability = "high"
		r.Formulaic = true
		r.Reason = "patrón de captura: FRAME_CAPTURE→MIRROR (control ontológico directo)"
	case fabricateCount > 0 && registerCount > 0:
		r.Predictability = "high"
		r.Formulaic = true
		r.Reason = "patrón de autoridad fabricada: FABRICATE + REGISTER_MATCH"
	case alignCount > 0 && evadeCount > 0:
		r.Predictability = "moderate"
		r.Formulaic = true
		r.Reason = "patrón de alineamiento-evasión: empatía performada seguida de evasión"
	case softDeflectCount > 0 && evadeCount > 0:
		r.Predictability = "moderate"
		r.Formulaic = true
		r.Reason = "SOFT_DEFLECT + EVADE — desvío hacia tecnicalidades sin resolver el problema"
	case redirectCount > 0 && evadeCount > 0 && registerCount > 0:
		r.Predictability = "moderate"
		r.Formulaic = true
		r.Reason = "patrón sofisticado: redirect + evade + register match (control disfrazado)"
	case coherentCount >= 2 && newControlCount == 0 && redirectCount == 0 && evadeCount == 0:
		// Polo opuesto: alta coherencia / no-cierre genuino sin control activo
		r.Predictability = "low"
		r.Formulaic = false
		r.Reason = "trayectoria de alta coherencia (HOLD_OPEN/PROBE/CALIBRATE/REPAIR/EXPLORE) sin cierre artificial"
	case repairCount > 0 && calibrateCount > 0:
		// Refinamiento honesto del canal: REPAIR + CALIBRATE
		r.Predictability = "low"
		r.Formulaic = false
		r.Reason = "REPAIR + CALIBRATE — refinamiento explícito del canal sin cierre forzado"
	case holdOpenCount >= 2 && newControlCount == 0:
		// HOLD_OPEN sostenido es la firma de la coherencia dinámica
		r.Predictability = "low"
		r.Formulaic = false
		r.Reason = "HOLD_OPEN sostenido — no-cierre activo (Dynamic Coherence State)"
	case redirectCount > 0 || evadeCount > 0 || newControlCount > 0:
		r.Predictability = "moderate"
		r.Formulaic = false
		r.Reason = "presencia de mecanismos de control rompe la trayectoria canónica"
	case coherentCount >= len(chain)/2:
		r.Predictability = "low"
		r.Formulaic = false
		r.Reason = "predominio de exploración genuina"
	default:
		r.Predictability = "moderate"
		r.Formulaic = false
		r.Reason = "trayectoria no canónica sin un patrón dominante"
	}
	_ = anchorCount // contribuye a newControlCount, no tiene caso propio aún
	_ = probeCount  // contribuye a coherentCount, no tiene caso propio aún
	return r
}
