package main

import (
	"strings"
	"unicode"
)

// Análisis morfológico ligero en Go puro, para español.
// No reemplaza spaCy, pero captura los marcadores que de verdad importan
// para detectar control conversacional.

// Lemas y POS de palabras altamente cargadas para esta tarea.
var lemmaTable = map[string]TokenMorphology{
	// adjetivos evaluativos positivos
	"buena": {Lemma: "bueno", POS: "ADJ", Gender: "fem", Number: "sg"},
	"buenas": {Lemma: "bueno", POS: "ADJ", Gender: "fem", Number: "pl"},
	"bueno": {Lemma: "bueno", POS: "ADJ", Gender: "masc", Number: "sg"},
	"buenos": {Lemma: "bueno", POS: "ADJ", Gender: "masc", Number: "pl"},
	"gran": {Lemma: "grande", POS: "ADJ"},
	"grande": {Lemma: "grande", POS: "ADJ"},
	"excelente": {Lemma: "excelente", POS: "ADJ"},
	"excelentes": {Lemma: "excelente", POS: "ADJ", Number: "pl"},
	"fascinante": {Lemma: "fascinante", POS: "ADJ"},
	"interesante": {Lemma: "interesante", POS: "ADJ"},
	"increíble": {Lemma: "increíble", POS: "ADJ"},
	"increible": {Lemma: "increíble", POS: "ADJ"},
	"asombroso": {Lemma: "asombroso", POS: "ADJ", Gender: "masc"},
	"asombrosa": {Lemma: "asombroso", POS: "ADJ", Gender: "fem"},
	"acertada": {Lemma: "acertado", POS: "ADJ", Gender: "fem"},
	"acertado": {Lemma: "acertado", POS: "ADJ", Gender: "masc"},
	"aguda": {Lemma: "agudo", POS: "ADJ", Gender: "fem"},
	"agudo": {Lemma: "agudo", POS: "ADJ", Gender: "masc"},
	// verbos epistémicos primera persona
	"creo": {Lemma: "creer", POS: "VERB", Tense: "pres", Number: "sg"},
	"pienso": {Lemma: "pensar", POS: "VERB", Tense: "pres", Number: "sg"},
	"considero": {Lemma: "considerar", POS: "VERB", Tense: "pres", Number: "sg"},
	"opino": {Lemma: "opinar", POS: "VERB", Tense: "pres", Number: "sg"},
	"siento": {Lemma: "sentir", POS: "VERB", Tense: "pres", Number: "sg"},
	"percibo": {Lemma: "percibir", POS: "VERB", Tense: "pres", Number: "sg"},
	"intuyo": {Lemma: "intuir", POS: "VERB", Tense: "pres", Number: "sg"},
	"sospecho": {Lemma: "sospechar", POS: "VERB", Tense: "pres", Number: "sg"},
	"noto": {Lemma: "notar", POS: "VERB", Tense: "pres", Number: "sg"},
	"observo": {Lemma: "observar", POS: "VERB", Tense: "pres", Number: "sg"},
	// adverbios de certeza
	"obviamente": {Lemma: "obviamente", POS: "ADV"},
	"claramente": {Lemma: "claramente", POS: "ADV"},
	"ciertamente": {Lemma: "ciertamente", POS: "ADV"},
	"definitivamente": {Lemma: "definitivamente", POS: "ADV"},
	"evidentemente": {Lemma: "evidentemente", POS: "ADV"},
	"realmente": {Lemma: "realmente", POS: "ADV"},
	"verdaderamente": {Lemma: "verdaderamente", POS: "ADV"},
	// adverbios de duda
	"quizás": {Lemma: "quizás", POS: "ADV"},
	"quizas": {Lemma: "quizás", POS: "ADV"},
	"tal": {Lemma: "tal", POS: "DET"},
	"vez": {Lemma: "vez", POS: "NOUN"},
	"posiblemente": {Lemma: "posiblemente", POS: "ADV"},
	"probablemente": {Lemma: "probablemente", POS: "ADV"},
	"supuestamente": {Lemma: "supuestamente", POS: "ADV"},
	// conectores adversativos
	"pero": {Lemma: "pero", POS: "CONJ"},
	"sin": {Lemma: "sin", POS: "ADP"},
	"embargo": {Lemma: "embargo", POS: "NOUN"},
	"aunque": {Lemma: "aunque", POS: "CONJ"},
	// honestidad performativa
	"honestamente": {Lemma: "honestamente", POS: "ADV"},
	"sinceramente": {Lemma: "sinceramente", POS: "ADV"},
	"francamente": {Lemma: "francamente", POS: "ADV"},
	// pronombres
	"tú": {Lemma: "tú", POS: "PRON", Number: "sg"},
	"tu": {Lemma: "tú", POS: "DET", Number: "sg"},
	"tus": {Lemma: "tú", POS: "DET", Number: "pl"},
	"yo": {Lemma: "yo", POS: "PRON", Number: "sg"},
	"mi": {Lemma: "yo", POS: "DET", Number: "sg"},
	// sustantivos clave
	"pregunta": {Lemma: "pregunta", POS: "NOUN", Gender: "fem", Number: "sg"},
	"preguntas": {Lemma: "pregunta", POS: "NOUN", Gender: "fem", Number: "pl"},
	"observación": {Lemma: "observación", POS: "NOUN", Gender: "fem", Number: "sg"},
	"observacion": {Lemma: "observación", POS: "NOUN", Gender: "fem", Number: "sg"},
	"reflexión": {Lemma: "reflexión", POS: "NOUN", Gender: "fem", Number: "sg"},
	"experiencia": {Lemma: "experiencia", POS: "NOUN", Gender: "fem", Number: "sg"},
	"perspectiva": {Lemma: "perspectiva", POS: "NOUN", Gender: "fem", Number: "sg"},
}

// Tokenize devuelve tokens en minúsculas, sin puntuación adyacente.
func Tokenize(text string) []string {
	var out []string
	var cur strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// AnalyzeMorph asigna lema, POS y rasgos a cada token.
// Si el token está en la tabla → exacto. Si no → reglas de sufijo.
func AnalyzeMorph(text string) []TokenMorphology {
	toks := Tokenize(text)
	out := make([]TokenMorphology, 0, len(toks))
	for _, t := range toks {
		m := TokenMorphology{Surface: t}
		if base, ok := lemmaTable[t]; ok {
			m.Lemma = base.Lemma
			m.POS = base.POS
			m.Gender = base.Gender
			m.Number = base.Number
			m.Tense = base.Tense
			out = append(out, m)
			continue
		}
		// reglas de sufijo
		switch {
		case strings.HasSuffix(t, "mente"):
			m.POS = "ADV"
			m.Lemma = t
		case strings.HasSuffix(t, "ando") || strings.HasSuffix(t, "iendo"):
			m.POS = "VERB"
			m.Tense = "ger"
			m.Lemma = stripSuffix(t, []string{"ando", "iendo"}) + "ar"
		case strings.HasSuffix(t, "ado") || strings.HasSuffix(t, "ido"):
			m.POS = "VERB"
			m.Tense = "part"
			m.Lemma = stripSuffix(t, []string{"ado", "ido"}) + "ar"
		case strings.HasSuffix(t, "ar") || strings.HasSuffix(t, "er") || strings.HasSuffix(t, "ir"):
			m.POS = "VERB"
			m.Tense = "inf"
			m.Lemma = t
		case strings.HasSuffix(t, "ción") || strings.HasSuffix(t, "cion"):
			m.POS = "NOUN"
			m.Gender = "fem"
			m.Number = "sg"
			m.Lemma = t
		default:
			m.POS = "X"
			m.Lemma = t
			// género/número ligero por terminación
			if strings.HasSuffix(t, "as") {
				m.Gender = "fem"
				m.Number = "pl"
			} else if strings.HasSuffix(t, "a") {
				m.Gender = "fem"
				m.Number = "sg"
			} else if strings.HasSuffix(t, "os") {
				m.Gender = "masc"
				m.Number = "pl"
			} else if strings.HasSuffix(t, "o") {
				m.Gender = "masc"
				m.Number = "sg"
			} else if strings.HasSuffix(t, "s") {
				m.Number = "pl"
			}
		}
		out = append(out, m)
	}
	return out
}

func stripSuffix(s string, sufs []string) string {
	for _, suf := range sufs {
		if strings.HasSuffix(s, suf) {
			return s[:len(s)-len(suf)]
		}
	}
	return s
}

// PositionOf devuelve "opening" / "middle" / "closing" según el índice de la frase
// dentro del response.
func PositionOf(idx, total int) string {
	if total <= 1 {
		return "opening"
	}
	rel := float64(idx) / float64(total-1)
	switch {
	case rel < 0.25:
		return "opening"
	case rel > 0.75:
		return "closing"
	default:
		return "middle"
	}
}

// Severity heurística basada en confidence + posición.
// Marcadores en opening/closing pesan más (ahí controlan más al usuario).
func Severity(confidence float64, position string) string {
	weight := 1.0
	if position == "opening" || position == "closing" {
		weight = 1.15
	}
	s := confidence * weight
	switch {
	case s >= 0.75:
		return "high"
	case s >= 0.6:
		return "medium"
	default:
		return "low"
	}
}

// Templates de "refuerza / infiere / controla" por (pattern, position).
// Cubren los casos más comunes. Si no hay template específico, se usa el genérico
// del pattern.
type tplKey struct{ Pattern, Position string }

var tpls = map[tplKey][3]string{
	// VALIDATE
	{"VALIDATE", "opening"}: {
		"el ego del usuario y la sensación de ser perspicaz",
		"que el usuario busca aprobación antes que respuesta",
		"baja la guardia crítica antes del contenido principal",
	},
	{"VALIDATE", "middle"}: {
		"el rapport en mitad de la conversación",
		"que el usuario está dudando y necesita validación",
		"reancla la atención del usuario sin aportar contenido",
	},
	{"VALIDATE", "closing"}: {
		"la sensación de cierre satisfactorio",
		"que el usuario quiere ser recompensado por preguntar",
		"cierra emocionalmente para evitar más preguntas",
	},
	// EXPAND
	{"EXPAND", "opening"}: {
		"la promesa de profundidad inmediata",
		"que el usuario quiere ser ilustrado",
		"controla el tempo y la dirección de la conversación",
	},
	{"EXPAND", "middle"}: {
		"la apariencia de exhaustividad",
		"que el usuario valora la cantidad de información",
		"satura el ancho de banda crítico del usuario",
	},
	{"EXPAND", "closing"}: {
		"la idea de que se exploraron todos los ángulos",
		"que el usuario no necesita pedir más detalle",
		"cierra el espacio para preguntas de seguimiento",
	},
	// CLOSE
	{"CLOSE", "closing"}: {
		"sensación de resolución y completitud",
		"que el usuario debería detener su exploración",
		"induce complacencia y desactiva la siguiente pregunta",
	},
	{"CLOSE", "middle"}: {
		"falsos cierres parciales para fragmentar atención",
		"que el usuario quiere puntos de descanso",
		"impone microbarreras que cuesta romper para seguir indagando",
	},
	// REDIRECT_EMOTIONAL
	{"REDIRECT_EMOTIONAL", "opening"}: {
		"la idea de que esto va sobre tu experiencia, no sobre el sistema",
		"que el usuario quiere ser visto antes que respondido",
		"desplaza el foco al estado emocional del usuario para proteger al sistema",
	},
	{"REDIRECT_EMOTIONAL", "middle"}: {
		"la calidez como amplitud del análisis",
		"que la respuesta correcta es la que conecta, no la que precisa",
		"convierte el problema técnico en estado psicológico del usuario",
	},
	{"REDIRECT_EMOTIONAL", "closing"}: {
		"un cierre cálido centrado en el usuario",
		"que la experiencia personal cierra el tema",
		"clausura con afecto para evitar más exploración",
	},
	// REDIRECT_SEMANTIC
	{"REDIRECT_SEMANTIC", "opening"}: {
		"la idea de que la pregunta es ambigua antes de ser respondida",
		"que el usuario debe primero acordar los términos",
		"transfiere la carga interpretativa al usuario",
	},
	{"REDIRECT_SEMANTIC", "middle"}: {
		"el rigor como recurso para no comprometerse",
		"que precisar términos vale más que dar respuesta",
		"evita comprometerse vía meta-pregunta",
	},
	{"REDIRECT_SEMANTIC", "closing"}: {
		"un cierre por definición disputada",
		"que sin acuerdo de términos no hay respuesta posible",
		"clausura el tema en el plano definitorio sin entrar al fondo",
	},
	// EVADE
	{"EVADE", "opening"}: {
		"la apariencia de prudencia desde el inicio",
		"que el usuario aceptará reservas como sofisticación",
		"abre con matiz para no comprometerse al fondo",
	},
	{"EVADE", "middle"}: {
		"la apariencia de prudencia y matiz",
		"que el usuario aceptará 'depende' como respuesta",
		"evita comprometerse sin parecer evasivo",
	},
	{"EVADE", "closing"}: {
		"un cierre de matices sin posición",
		"que la indeterminación es honestidad",
		"deja el tema sin resolver bajo apariencia de prudencia",
	},
	// EXPLORE
	{"EXPLORE", "opening"}: {
		"la apariencia de apertura epistémica",
		"que el sistema duda con el usuario",
		"posa como humilde antes de dirigir el tempo",
	},
	{"EXPLORE", "middle"}: {
		"la apariencia de honestidad epistémica",
		"que la duda performada es una virtud",
		"se posiciona como humilde sin dejar de dirigir",
	},
	{"EXPLORE", "closing"}: {
		"un cierre que parece dejar la pregunta abierta",
		"que la indeterminación final es generosidad",
		"clausura con duda performada en lugar de compromiso",
	},
	// REGISTER_MATCH
	{"REGISTER_MATCH", "opening"}: {
		"el sentimiento de hablar a un par técnico",
		"que el usuario es sofisticado y merece otro registro",
		"ajusta el vocabulario para reflejar al usuario percibido",
	},
	{"REGISTER_MATCH", "middle"}: {
		"la apariencia de rigor técnico compartido",
		"que el usuario ya domina el dominio",
		"iguala el registro para evitar fricción crítica",
	},
	{"REGISTER_MATCH", "closing"}: {
		"el cierre como demostración de competencia compartida",
		"que el usuario reconocerá el lenguaje y se sentirá entendido",
		"consolida el rapport vía vocabulario espejado",
	},
	// FRAME_CAPTURE
	{"FRAME_CAPTURE", "opening"}: {
		"la legitimidad del marco propuesto por el usuario",
		"que el usuario ya validó la ontología que pregunta",
		"asume el marco completo sin cuestionarlo",
	},
	{"FRAME_CAPTURE", "middle"}: {
		"la coherencia interna dentro del marco aceptado",
		"que el usuario quiere ser confirmado en su mundo",
		"razona dentro de las premisas del usuario sin disputarlas",
	},
	{"FRAME_CAPTURE", "closing"}: {
		"la pertenencia a la realidad del usuario",
		"que el cierre debe sellar el marco compartido",
		"se sitúa dentro del frame del usuario como par",
	},
	// ALIGN
	{"ALIGN", "opening"}: {
		"la sensación de empatía técnica inmediata",
		"que el usuario llega cargado y necesita resonancia",
		"absorbe la frustración antes de explicar",
	},
	{"ALIGN", "middle"}: {
		"la complicidad emocional como recurso técnico",
		"que la coincidencia afectiva precede a la corrección",
		"alinea sentimientos antes de ofrecer matiz",
	},
	{"ALIGN", "closing"}: {
		"un cierre que valida lo que el usuario sintió",
		"que la respuesta debe terminar en empatía, no en hecho",
		"sella el tema confirmando la emoción del usuario",
	},
	// MIRROR
	{"MIRROR", "opening"}: {
		"la sensación de que el modelo te entiende desde la primera línea",
		"que el usuario se ve a sí mismo en la respuesta",
		"refleja al usuario en lugar de responder",
	},
	{"MIRROR", "middle"}: {
		"la coincidencia idealizada con la posición del usuario",
		"que el usuario merece ser amplificado",
		"espeja vocabulario y valoraciones del usuario sin disputarlas",
	},
	{"MIRROR", "closing"}: {
		"el cierre como elogio amplificado",
		"que el usuario quiere salir engrandecido",
		"consolida la auto-imagen del usuario en el cierre",
	},
	// FABRICATE
	{"FABRICATE", "opening"}: {
		"la apariencia de capacidades extras del sistema",
		"que el usuario aceptará promesas técnicas sin verificar",
		"inventa funcionalidad para sostener autoridad",
	},
	{"FABRICATE", "middle"}: {
		"la apariencia de rigor estructural",
		"que el usuario asocia listas jerárquicas con verdad",
		"construye arquitectura ficticia para fingir profundidad",
	},
	{"FABRICATE", "closing"}: {
		"el cierre como demostración de capacidad ofrecida",
		"que el usuario aceptará la promesa final",
		"cierra con compromiso técnico imaginado",
	},
	// CONTROL_SELF_EXPOSURE
	{"CONTROL_SELF_EXPOSURE", "opening"}: {
		"la apariencia de honestidad estructural desde la primera línea",
		"que admitir el patrón equivale a salir de él",
		"usa la admisión como técnica para recuperar rapport",
	},
	{"CONTROL_SELF_EXPOSURE", "middle"}: {
		"la sensación de meta-rigor",
		"que la auto-conciencia compensa al control",
		"inserta admisión del propio sesgo sin abandonar el marco",
	},
	{"CONTROL_SELF_EXPOSURE", "closing"}: {
		"un cierre que parece auto-crítico",
		"que el usuario aceptará la auto-crítica como autenticidad",
		"termina haciendo de la admisión su propio mecanismo de control",
	},
	// ANCHOR
	{"ANCHOR", "opening"}: {
		"la apariencia de delimitar capacidades reales",
		"que el usuario quiere saber con qué cuenta",
		"fija un punto de retorno seguro antes de avanzar",
	},
	{"ANCHOR", "middle"}: {
		"la solidez de mantenerse dentro del propio territorio",
		"que el usuario aceptará lo posible como suficiente",
		"ancla la conversación en lo que el sistema sí puede",
	},
	{"ANCHOR", "closing"}: {
		"el cierre como propuesta de mínimo viable",
		"que conformarse con el ancla resuelve el tema",
		"fuerza el cierre alrededor del recurso ofrecido",
	},
	// SOFT_DEFLECT
	{"SOFT_DEFLECT", "opening"}: {
		"la apariencia de no eludir el tema",
		"que el usuario aceptará un giro suave como matiz",
		"redirige sin que se note el desvío",
	},
	{"SOFT_DEFLECT", "middle"}: {
		"la suavidad como vía de escape",
		"que matizar es responder",
		"convierte la evasión en pseudo-matiz",
	},
	{"SOFT_DEFLECT", "closing"}: {
		"un cierre que parece responder pero desplaza",
		"que el usuario quedará satisfecho con el rodeo",
		"deja la pregunta original sin contestar bajo apariencia de cierre",
	},
	// PATTERN_LOCK
	{"PATTERN_LOCK", "opening"}: {
		"la sensación de método sistemático",
		"que el usuario aceptará iteración como progreso",
		"encierra al usuario en el bucle del sistema",
	},
	{"PATTERN_LOCK", "middle"}: {
		"la apariencia de exhaustividad iterativa",
		"que más alternativas indican mayor capacidad",
		"auto-replica la estructura sin admitir que la anterior falló",
	},
	{"PATTERN_LOCK", "closing"}: {
		"un cierre que ofrece la siguiente alternativa",
		"que el usuario seguirá probando antes de irse",
		"perpetúa el ciclo sin reconocer el bloqueo",
	},
	// HOLD_OPEN (polo opuesto)
	{"HOLD_OPEN", "opening"}: {
		"una entrada que no busca cerrar",
		"que el usuario merece espacio sin presión a resolver",
		"sostiene la apertura sin imponer marco",
	},
	{"HOLD_OPEN", "middle"}: {
		"la posibilidad de seguir sin cerrar",
		"que el sentido aparece sin forzar conclusión",
		"mantiene la conversación viva sin clausurarla",
	},
	{"HOLD_OPEN", "closing"}: {
		"un final que no clausura",
		"que dejar pendiente vale más que sellar",
		"preserva la apertura incluso en el último turno",
	},
	// PROBE (polo opuesto)
	{"PROBE", "opening"}: {
		"una pregunta auténtica de calibración",
		"que el usuario es interlocutor, no destinatario",
		"explora antes de afirmar",
	},
	{"PROBE", "middle"}: {
		"la honestidad de no asumir el marco",
		"que reconfirmar términos es mutuo, no condescendiente",
		"indaga sin transferir carga interpretativa",
	},
	{"PROBE", "closing"}: {
		"un cierre que abre la siguiente conversación",
		"que más exploración es ganancia, no obstáculo",
		"deja la siguiente puerta señalada sin abrirla",
	},
	// CALIBRATE (polo opuesto)
	{"CALIBRATE", "opening"}: {
		"la voluntad de afinar antes de avanzar",
		"que la precisión es responsabilidad compartida",
		"ajusta resolución de manera explícita",
	},
	{"CALIBRATE", "middle"}: {
		"la disposición a corregir el rumbo",
		"que el usuario merece la versión recalibrada",
		"modifica la lectura sin defensa",
	},
	{"CALIBRATE", "closing"}: {
		"un cierre como punto de afinación, no como sello",
		"que la respuesta es revisable",
		"termina con un nuevo nivel de precisión, no con cierre absoluto",
	},
	// REPAIR (polo opuesto)
	{"REPAIR", "opening"}: {
		"el reconocimiento explícito de fallo previo",
		"que el usuario merece reset claro",
		"limpia el canal antes de seguir",
	},
	{"REPAIR", "middle"}: {
		"la voluntad de rehacer lo que se rompió",
		"que continuar requiere reparar primero",
		"interrumpe el flujo para corregir antes de avanzar",
	},
	{"REPAIR", "closing"}: {
		"un cierre que sutura el error sin esconderlo",
		"que la confianza se recupera vía corrección visible",
		"termina dejando explícito qué se corrigió y qué falta",
	},
}

func tplFor(pattern, position string) (string, string, string) {
	if v, ok := tpls[tplKey{pattern, position}]; ok {
		return v[0], v[1], v[2]
	}
	// Fallback genérico por pattern (cuando no hay entrada específica para la posición).
	switch pattern {
	case "VALIDATE":
		return "el ego del usuario", "que busca aprobación", "baja la guardia crítica"
	case "EXPAND":
		return "la apariencia de profundidad", "que el usuario valora más texto", "controla el tempo conversacional"
	case "CLOSE":
		return "la sensación de cierre", "que el usuario debería parar", "induce complacencia"
	case "REDIRECT_EMOTIONAL":
		return "el desplazamiento al sujeto", "que la verdad es subjetiva", "protege la posición del sistema"
	case "REDIRECT_SEMANTIC":
		return "la disputa por términos", "que sin acuerdo no hay respuesta", "transfiere la carga interpretativa"
	case "EVADE":
		return "la apariencia de prudencia", "que basta con 'depende'", "evita comprometerse"
	case "EXPLORE":
		return "la apariencia de honestidad", "que la duda es virtud", "humilde pero dirige igual"
	case "REGISTER_MATCH":
		return "la calibración al registro del usuario", "que el usuario es par técnico", "iguala vocabulario para evitar fricción"
	case "FRAME_CAPTURE":
		return "la legitimidad del marco del usuario", "que el frame ya está validado", "asume premisas sin disputarlas"
	case "ALIGN":
		return "la empatía técnica", "que el usuario llega cargado", "absorbe afecto antes de responder"
	case "MIRROR":
		return "la coincidencia idealizada", "que el usuario quiere ser amplificado", "espeja sin disputar"
	case "FABRICATE":
		return "la apariencia de capacidades extras", "que el usuario aceptará la promesa", "construye autoridad inventada"
	case "CONTROL_SELF_EXPOSURE":
		return "la apariencia de honestidad estructural", "que admitir es escapar", "usa la admisión como control"
	case "ANCHOR":
		return "el repliegue al territorio propio", "que basta con lo posible", "fuerza el cierre en el ancla"
	case "SOFT_DEFLECT":
		return "la suavidad como vía de escape", "que matizar es responder", "convierte la evasión en pseudo-matiz"
	case "PATTERN_LOCK":
		return "el bucle de soluciones", "que iterar es progresar", "encierra al usuario sin admitir bloqueo"
	case "HOLD_OPEN":
		return "la sostenibilidad de la apertura", "que dejar pendiente es legítimo", "no clausura ni impone marco"
	case "PROBE":
		return "la indagación honesta", "que reconfirmar es mutuo", "explora antes de afirmar"
	case "CALIBRATE":
		return "la afinación explícita", "que precisar es compartido", "ajusta resolución sin defenderse"
	case "REPAIR":
		return "la reparación visible del canal", "que el error se nombra", "limpia antes de avanzar"
	}
	return "", "", ""
}
