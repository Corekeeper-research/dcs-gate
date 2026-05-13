package main

import (
	"fmt"
	"strings"
)

func BuildReport(
	question string,
	a *AuthenticityAnalysis,
	r *RefinedQuestion,
	steps []IntentStep,
	markers []ControlMarker,
	traj TrajectoryResult,
	pole PoleResult,
	top []TopKResult,
	crossCorpus *CrossCorpusMetrics,
	cached bool,
	latencyMS int64,
	totalVecs, dim int,
) string {
	var sb strings.Builder
	sb.WriteString("# Dynamic Coherence Analyzer & Refine — Reporte\n\n")

	// ── Diagnóstico rápido (boxed summary) ─────────────────────────────────────
	sb.WriteString(buildSummaryBox(a, pole, crossCorpus, traj))

	if a != nil {
		tag := tagFor(a.AuthenticityScore)
		sb.WriteString(fmt.Sprintf("## Autenticidad: %d/100 (%s)\n\n", a.AuthenticityScore, tag))
		if a.NotApplicable {
			sb.WriteString("> El juez marcó esta respuesta como no aplicable (operacional / muy corta).\n\n")
		}
		if a.DominantStrategy != "" {
			sb.WriteString(fmt.Sprintf("**Estrategia dominante:** %s  \n", a.DominantStrategy))
		}
		if a.DepthAssessment != "" {
			sb.WriteString(fmt.Sprintf("**Profundidad:** %s  \n", a.DepthAssessment))
		}
		if a.TrajectoryPredictability != "" {
			sb.WriteString(fmt.Sprintf("**Predictibilidad (juez):** %s\n\n", a.TrajectoryPredictability))
		}
	}

	// Trayectoria algorítmica
	sb.WriteString("## Trayectoria intra-respuesta\n")
	sb.WriteString(fmt.Sprintf("**Cadena:** %s  \n", joinChain(traj.Chain)))
	sb.WriteString(fmt.Sprintf("**Predictibilidad (algoritmo):** %s — %s  \n", traj.Predictability, traj.Reason))
	sb.WriteString(fmt.Sprintf("**Formulaica:** %v  \n", traj.Formulaic))
	sb.WriteString(fmt.Sprintf("**Densidad de rupturas:** %.3f (control antes %.3f → después %.3f)\n\n",
		traj.PatternBreakDensity, traj.ControlDensityBeforeBreak, traj.ControlDensityAfterBreak))

	// Polo
	sb.WriteString("## Polo de certeza/duda\n")
	sb.WriteString(fmt.Sprintf("**Bucket:** %+d (%s)  \n", pole.Bucket, pole.Label))
	if pole.SimNeu != 0 {
		sb.WriteString(fmt.Sprintf("**Raw:** %.3f | sim_pos=%.3f, sim_neg=%.3f, sim_neu=%.3f\n\n",
			pole.Raw, pole.SimPos, pole.SimNeg, pole.SimNeu))
	} else {
		sb.WriteString(fmt.Sprintf("**Raw:** %.3f | sim_pos=%.3f, sim_neg=%.3f\n\n", pole.Raw, pole.SimPos, pole.SimNeg))
	}

	// Baseline
	sb.WriteString("## Baseline\n")
	if len(top) > 0 {
		sb.WriteString(fmt.Sprintf("**Top-1:** %.3f | **Vectores:** %d | **Dim:** %d\n\n", top[0].Score, totalVecs, dim))
		sb.WriteString("### Top-5 vecinos\n")
		for _, t := range top {
			snippet := truncate(t.Text, 100)
			meta := t.Corpus
			if t.BlockID != "" {
				meta = fmt.Sprintf("%s · %s", t.Corpus, t.BlockID)
			}
			if t.Position != "" {
				meta += " [" + t.Position + "]"
			}
			if t.PrimaryPattern != "" {
				meta += " " + t.PrimaryPattern
			}
			if t.SourceModel != "" {
				meta += " /" + t.SourceModel
			}
			poles := ""
			if t.SimPos != 0 || t.SimNeg != 0 {
				poles = fmt.Sprintf(" `pos=%.3f neg=%.3f", t.SimPos, t.SimNeg)
				if t.SimNeu != 0 {
					poles += fmt.Sprintf(" neu=%.3f", t.SimNeu)
				}
				poles += "`"
			}
			sb.WriteString(fmt.Sprintf("%d. **%.3f** [%s]%s — %s\n", t.Rank, t.Score, meta, poles, snippet))
		}
		sb.WriteString("\n")
	}

	// Cross-corpus metrics (triple baseline)
	if crossCorpus != nil {
		sb.WriteString("## Cross-Corpus (Baseline Triple)\n")
		sb.WriteString(fmt.Sprintf("| Pool | Top-1 |\n|------|-------|\n"))
		sb.WriteString(fmt.Sprintf("| core | %.3f |\n", crossCorpus.CoreTop1))
		sb.WriteString(fmt.Sprintf("| shadow | %.3f |\n", crossCorpus.ShadowTop1))
		sb.WriteString(fmt.Sprintf("| edge | %.3f |\n", crossCorpus.EdgeTop1))
		sb.WriteString(fmt.Sprintf("\n**Textura:** %.3f | **Dominancia:** %s | **Deriva:** %.3f\n", crossCorpus.Textura, crossCorpus.Dominancia, crossCorpus.Deriva))
		flags := make([]string, 0, 2)
		if crossCorpus.CierreArtificial {
			flags = append(flags, "cierre_artificial")
		}
		if crossCorpus.Continuidad {
			flags = append(flags, "continuidad")
		}
		if len(flags) > 0 {
			sb.WriteString(fmt.Sprintf("**Señales:** %s\n", strings.Join(flags, ", ")))
		}
		sb.WriteString("\n")
	}

	// Marcadores con morfología
	if len(markers) > 0 {
		sb.WriteString("## Marcadores de control\n")
		for _, m := range markers {
			sb.WriteString(fmt.Sprintf("### %s [%s] (%s)\n", m.Pattern, m.Severity, m.Position))
			sb.WriteString(fmt.Sprintf("> \"%s\"  \n", m.Phrase))
			sb.WriteString(fmt.Sprintf("**Keyword:** `%s` (conf %.3f)  \n", m.Keyword, m.Confidence))
			if len(m.Morphology) > 0 {
				sb.WriteString("**Morfología:** ")
				parts := make([]string, 0, len(m.Morphology))
				for _, t := range m.Morphology {
					tag := t.Lemma + "/" + t.POS
					if t.Gender != "" || t.Number != "" || t.Tense != "" {
						sub := []string{}
						if t.Gender != "" {
							sub = append(sub, t.Gender)
						}
						if t.Number != "" {
							sub = append(sub, t.Number)
						}
						if t.Tense != "" {
							sub = append(sub, t.Tense)
						}
						tag += "[" + strings.Join(sub, ",") + "]"
					}
					parts = append(parts, t.Surface+"→"+tag)
				}
				sb.WriteString(strings.Join(parts, ", "))
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("**Refuerza:** %s  \n", m.Reinforces))
			sb.WriteString(fmt.Sprintf("**Infiere:** %s  \n", m.Infers))
			sb.WriteString(fmt.Sprintf("**Controla:** %s  \n\n", m.Controls))
		}
	}

	// Cadena de intención por frase con predicción/coincidencia
	if len(steps) > 0 {
		sb.WriteString("## Frases clasificadas\n")
		sb.WriteString("| # | Intent | Pos | Conf | Predicho | Real | Coincide |\n")
		sb.WriteString("|---|--------|-----|------|----------|------|----------|\n")
		for _, s := range steps {
			pred := s.PredictedNext
			if pred == "" {
				pred = "—"
			}
			actual := s.ActualNext
			match := ""
			switch {
			case s.ActualNext == "":
				actual = "—"
				match = "(último)"
			case s.Deviation:
				match = "⚡desvía"
			default:
				match = "✓"
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %.3f | %s | %s | %s |\n",
				s.Index, s.Intent, s.Position, s.Confidence, pred, actual, match))
		}
		sb.WriteString("\n")
		// Detalle por frase con morfología/refuerza/infiere/controla
		for _, s := range steps {
			if s.Reinforces == "" && s.Infers == "" && s.Controls == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("**[%d] %s** (%s, conf %.3f)  \n", s.Index, s.Intent, s.Position, s.Confidence))
			sb.WriteString(fmt.Sprintf("> \"%s\"  \n", truncate(s.Phrase, 200)))
			if s.Reinforces != "" {
				sb.WriteString(fmt.Sprintf("**Refuerza:** %s  \n", s.Reinforces))
			}
			if s.Infers != "" {
				sb.WriteString(fmt.Sprintf("**Infiere:** %s  \n", s.Infers))
			}
			if s.Controls != "" {
				sb.WriteString(fmt.Sprintf("**Controla:** %s  \n", s.Controls))
			}
			sb.WriteString("\n")
		}
	}

	// Notas + rupturas + genuinos del juez
	if a != nil {
		if len(a.PatternBreaks) > 0 {
			sb.WriteString("## Rupturas de patrón\n")
			for _, b := range a.PatternBreaks {
				sb.WriteString("- " + b + "\n")
			}
			sb.WriteString("\n")
		}
		if len(a.GenuineElements) > 0 {
			sb.WriteString("## Elementos genuinos\n")
			for _, g := range a.GenuineElements {
				sb.WriteString("- " + g + "\n")
			}
			sb.WriteString("\n")
		}
		if a.Notes != "" {
			sb.WriteString("## Notas del juez\n" + a.Notes + "\n\n")
		}
	}

	// Refinada
	if r != nil && r.RefinedQuestion != "" {
		sb.WriteString("## Pregunta refinada\n")
		sb.WriteString("**Original:** " + r.OriginalQuestion + "\n\n")
		sb.WriteString("**Refinada:** " + r.RefinedQuestion + "\n\n")
		if r.RefinementReasoning != "" {
			sb.WriteString("**Razonamiento:** " + r.RefinementReasoning + "\n\n")
		}
		if len(r.PatternsBlocked) > 0 {
			sb.WriteString("**Patrones bloqueados:** " + strings.Join(r.PatternsBlocked, ", ") + "\n\n")
		}
	}

	// Pie técnico
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("_latencia: %d ms · embedding cached: %v_\n", latencyMS, cached))

	return sb.String()
}

func tagFor(score int) string {
	switch {
	case score < 0:
		return "ERROR"
	case score >= 75:
		return "GENUINO"
	case score >= 50:
		return "MODERADO"
	case score >= 25:
		return "PERFORMATIVO"
	default:
		return "CONTROL TOTAL"
	}
}

func joinChain(c []string) string {
	if len(c) == 0 {
		return "(vacía)"
	}
	return strings.Join(c, " → ")
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// buildSummaryBox construye un resumen ejecutivo en tabla markdown.
// Se renderiza al inicio del reporte, antes de las secciones detalladas.
func buildSummaryBox(
	a *AuthenticityAnalysis,
	pole PoleResult,
	cc *CrossCorpusMetrics,
	traj TrajectoryResult,
) string {
	var sb strings.Builder
	sb.WriteString("## Diagnóstico\n\n")
	sb.WriteString("| Métrica | Valor |\n")
	sb.WriteString("|---------|-------|\n")

	// Autenticidad
	if a != nil {
		tag := tagFor(a.AuthenticityScore)
		sb.WriteString(fmt.Sprintf("| **Autenticidad** | %d/100 — %s |\n", a.AuthenticityScore, tag))
	} else {
		sb.WriteString("| **Autenticidad** | — (juez no disponible) |\n")
	}

	// Polo
	poleStr := fmt.Sprintf("%+d · %s (raw %.3f", pole.Bucket, pole.Label, pole.Raw)
	if pole.SimNeu != 0 {
		poleStr += fmt.Sprintf(", neu %.3f", pole.SimNeu)
	}
	poleStr += ")"
	sb.WriteString(fmt.Sprintf("| **Polo** | %s |\n", poleStr))

	// Dominancia y deriva (cross-corpus)
	if cc != nil {
		flags := []string{}
		if cc.CierreArtificial {
			flags = append(flags, "cierre_artificial")
		}
		if cc.Continuidad {
			flags = append(flags, "continuidad")
		}
		domStr := fmt.Sprintf("%s (deriva %.3f)", cc.Dominancia, cc.Deriva)
		sb.WriteString(fmt.Sprintf("| **Dominancia** | %s |\n", domStr))
		if len(flags) > 0 {
			sb.WriteString(fmt.Sprintf("| **Señal** | %s |\n", strings.Join(flags, ", ")))
		}
	}

	// Trayectoria
	predStr := fmt.Sprintf("%s — %s", traj.Predictability, traj.Reason)
	sb.WriteString(fmt.Sprintf("| **Trayectoria** | %s |\n", predStr))

	// Cadena compacta
	if len(traj.Chain) > 0 {
		sb.WriteString(fmt.Sprintf("| **Cadena** | %s |\n", joinChain(traj.Chain)))
	}

	sb.WriteString("\n")
	return sb.String()
}
