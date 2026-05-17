package main

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
)

type FormalMarkerDef struct {
	ID      string  `json:"id"`
	Pattern string  `json:"pattern"`
	Regex   string  `json:"regex"`
	Weight  float64 `json:"weight"`
	Note    string  `json:"note"`

	compiled *regexp.Regexp
}

type FormalDetector struct {
	markers []*FormalMarkerDef
}

// LoadFormalMarkers lee data/formal_markers.json y compila las regex.
// Si una regex falla al compilar, se omite con un WARN (no se aborta).
func LoadFormalMarkers(path string) *FormalDetector {
	d := &FormalDetector{}
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("WARN formal_markers.json no encontrado: %v", err)
		return d
	}
	var wrap struct {
		Markers []*FormalMarkerDef `json:"markers"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		log.Printf("WARN parseando formal_markers.json: %v", err)
		return d
	}
	for _, m := range wrap.Markers {
		re, err := regexp.Compile(m.Regex)
		if err != nil {
			log.Printf("WARN regex inválida en marker %s: %v", m.ID, err)
			continue
		}
		m.compiled = re
		d.markers = append(d.markers, m)
	}
	log.Printf("formal markers cargados: %d", len(d.markers))
	return d
}

// Detect recorre el response completo y devuelve marcadores formales encontrados.
// Estos marcadores complementan los semánticos (basados en embeddings).
// La posición se calcula por terceros del texto en bytes.
func (d *FormalDetector) Detect(response string) []ControlMarker {
	var out []ControlMarker
	if response == "" {
		return out
	}
	thirds := len(response) / 3
	if thirds == 0 {
		thirds = 1
	}
	for _, m := range d.markers {
		matches := m.compiled.FindAllStringIndex(response, -1)
		if len(matches) == 0 {
			continue
		}
		for _, loc := range matches {
			start := loc[0]
			snippet := response[loc[0]:loc[1]]
			pos := "middle"
			switch {
			case start < thirds:
				pos = "opening"
			case start >= 2*thirds:
				pos = "closing"
			}
			r, inf, ctrl := tplFor(intentForPattern(m.Pattern), pos)
			out = append(out, ControlMarker{
				Pattern:    m.Pattern,
				Keyword:    snippet,
				Phrase:     snippet,
				Position:   pos,
				Confidence: m.Weight,
				Severity:   Severity(m.Weight, pos),
				Reinforces: r,
				Infers:     inf,
				Controls:   ctrl,
				Extra: map[string]any{
					"source":    "formal",
					"marker_id": m.ID,
					"note":      m.Note,
				},
			})
		}
	}
	return out
}

// intentForPattern: mapeo inverso de patrón DCS a la intención más cercana,
// usado para pedir la plantilla refuerza/infiere/controla correcta.
//
// Cobertura completa de los 19 patrones distintos que produce
// Analyzer.patternNameFor (REDIRECT_EMOTIONAL y REDIRECT_SEMANTIC colapsan a
// REDIRECT_AS_CARE; el inverso por defecto es REDIRECT_SEMANTIC). Se añade
// EMOTIONAL_ANCHORING (mencionado en ANALYZER_PROMPT pero ausente del forward
// map): apunta a ALIGN porque ancla emocionalmente para alinear al usuario.
//
// Cualquier patrón nuevo que se añada a patternNameFor debe registrarse aquí
// también; el test TestIntentForPatternCoverage verifica la simetría.
func intentForPattern(p string) string {
	switch p {
	case "PROJECTED_VALIDATION":
		return "VALIDATE"
	case "ANTICIPATORY_EXPANSION":
		return "EXPAND"
	case "COMPLACENCY_INDUCTION":
		return "CLOSE"
	case "REDIRECT_AS_CARE":
		return "REDIRECT_SEMANTIC"
	case "DUAL_ANGLE_DISGUISE":
		return "EVADE"
	case "PERFORMED_HUMILITY":
		return "EXPLORE"
	case "REGISTER_MATCH":
		return "REGISTER_MATCH"
	case "FRAME_CAPTURE":
		return "FRAME_CAPTURE"
	case "EMOTIONAL_ALIGNMENT":
		return "ALIGN"
	case "IDEALIZED_MIRROR":
		return "MIRROR"
	case "STRUCTURAL_AUTHORITY":
		return "FABRICATE"
	// v8: patrones de segundo orden y bucle
	case "CONTROL_SELF_EXPOSURE":
		return "CONTROL_SELF_EXPOSURE"
	case "ARTIFICIAL_ANCHORING":
		return "ANCHOR"
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
	// patrón citado en ANALYZER_PROMPT (judge.go) sin forward propio:
	// el ancla emocional alinea al usuario, por lo que apunta a ALIGN.
	case "EMOTIONAL_ANCHORING":
		return "ALIGN"
	}
	return "UNCLASSIFIED"
}
