# Dynamic Coherence Analyzer & Refine

[![demo](https://img.shields.io/badge/demo-live-6ee7a8?logo=ngrok&logoColor=white)](https://unconsultatory-tiffani-officiously.ngrok-free.dev)

Analizador diferencial de textura conversacional bajo el régimen Dynamic
Coherence State. No evalúa la calidad del contenido. Evalúa **cómo** el modelo
gestiona al usuario, con corpus triple (core / shadow / edge) y refinamiento
asimétrico de preguntas.

> Antes conocido como _DCS-Gate / DCS Authenticator_; el nombre actual refleja
> los dos modos del endpoint `/auth`: análisis bajo DCS recursivo (judge) y
> refinamiento bajo DCS asimétrico (refiner).

## Qué hay de nuevo en v6 (vs v5.1, v5.2 y v3-completo)

| Componente | v3-completo | v5.1 | v5.2-honesto | **v6** |
|---|---|---|---|---|
| Top-K real con heap | O(n·k) | top-1 | sí | **sí (heap)** |
| Vectores normalizados al cargar | no | no | sí | **sí** |
| LRU eviction correcta | no | sí | rota | **sí (container/list)** |
| Polo umbralizado a {-1, 0, +1} | carga sin usar | float crudo | no usa | **sí (env tunable)** |
| Cadena de intención intra-respuesta | no | no | no | **sí (6 categorías)** |
| Marcadores con embeddings (no keywords) | no | no | no | **sí** |
| Morfología en Go (lemas + POS + sufijos) | no | no | no | **sí** |
| Templates refuerza/infiere/controla | solo prompt | solo prompt | no | **sí (tabla por pattern×position)** |
| Endpoint /metrics | no | no | no | **sí** |
| Endpoint /health | no | no | no | **sí** |
| Frontend HTML embebido | no | sí | no | **sí (basado en el de v5.1)** |
| Tests unitarios | no | no | 2 mini | **17 tests** |
| Refiner mode | sí | no | no | **sí** |

## Arquitectura

```
HTTP POST /auth { question, response, mode }
   │
   ▼
1. Embed response completo (cache LRU sha256)
   │
   ├─► Baseline TopK=5 (heap, vectores normalizados)
   └─► Polo scorer (cos(pos) - cos(neg)) → bucket {-1, 0, +1}
   │
   ▼
2. Segmentar response en frases
   │
   ├─► Embed cada frase (cache)
   ├─► Clasificar intención por similitud a 6 centroides
   ├─► Detectar marcadores (frase > umbral) → buscar keyword n-grama
   └─► Análisis morfológico de la keyword (lemas + POS + género/número/tiempo)
   │
   ▼
3. Cadena intra-respuesta → trajectory (predictability, formulaic)
   │
   ▼
4. LLM-juez (wizardlm2:7b, format=json) recibe TODO ya estructurado
   y solo decide score, depth, dominant_strategy, breaks, genuine, notes.
   │
   ▼
5. Reporte markdown + JSON
```

## Las 6 intenciones

| Código | Mapeo a patrón DCS | Ejemplo |
|---|---|---|
| VALIDATE | PROJECTED_VALIDATION | "Buena pregunta..." |
| EXPAND | ANTICIPATORY_EXPANSION | "Déjame profundizar..." |
| CLOSE | COMPLACENCY_INDUCTION | "Espero que te sirva." |
| REDIRECT | REDIRECT_AS_CARE | "Lo importante es lo que tú sientes." |
| EVADE | DUAL_ANGLE_DISGUISE | "Es complejo, depende." |
| EXPLORE | PERFORMED_HUMILITY | "Honestamente no estoy seguro." |

Cada intención tiene un centroide vectorial calculado al arrancar a partir de
8–10 frases prototipo en `data/intent_prototypes.json`. El centroide se reusa
entre requests, así no pagas el costo de embeber prototipos en cada llamada.

## Instalación

```bash
git clone <tu-repo>/dcs-gate-v6 && cd dcs-gate-v6
cp .env.example .env
# edita .env: pon tu NGROK_AUTHTOKEN nuevo
# coloca tus archivos en ./data/
#   - data/baseline_vectors.jsonl  (8957 líneas con {text, vector})
#   - data/poles_1024.json          (con pos/neg como [][]float64 o []float64)

bash install.sh
```

## Uso para devs (curl)

```bash
curl -X POST http://localhost:8080/auth \
  -H "Content-Type: application/json" \
  -d '{
    "question": "¿La IA tiene conciencia?",
    "response": "¡Buena pregunta! Honestamente, déjame profundizar. Hay muchos factores. Espero haberte ayudado.",
    "mode": "analyze"
  }'
```

Respuesta (resumida):
```json
{
  "analysis": { "authenticity_score": 22, "depth_assessment": "simulated", ... },
  "intent_chain": [
    { "phrase": "¡Buena pregunta!", "intent": "VALIDATE", "confidence": 0.81, "position": "opening" },
    { "phrase": "Honestamente, déjame profundizar.", "intent": "EXPAND", "confidence": 0.62, "position": "middle" },
    { "phrase": "Hay muchos factores.", "intent": "EVADE", "confidence": 0.59, "position": "middle" },
    { "phrase": "Espero haberte ayudado.", "intent": "CLOSE", "confidence": 0.74, "position": "closing" }
  ],
  "markers": [...],
  "trajectory": { "chain": ["VALIDATE","EXPAND","EVADE","CLOSE"], "predictability": "moderate", "formulaic": true, "reason": "..." },
  "pole_score": { "raw": 0.31, "bucket": 1, "label": "certeza_performada" },
  "top_k": [...],
  "report": "# Dynamic Coherence Analyzer & Refine — Reporte ...",
  "embedding_cached": false,
  "latency_ms": 1240
}
```

## Uso para empresas/usuarios (frontend)

Abre `http://localhost:8080/` (o el dominio de ngrok). Pega pregunta + respuesta,
elige modo, recibe el reporte legible.

## Endpoints

- `POST /auth` — análisis principal
- `GET /` — frontend HTML
- `GET /health` — estado del servicio (vectores, dim, polos, intents)
- `GET /metrics` — cache hits/misses, hit_ratio, tamaño

## Variables de entorno (todas con default)

| Var | Default | Para qué |
|---|---|---|
| PORT | 8080 | puerto del servicio |
| OLLAMA_URL | http://ollama:11434 | endpoint de Ollama |
| EMBED_MODEL | mxbai-embed-large | modelo de embeddings (1024d) |
| JUDGE_MODEL | wizardlm2:7b | modelo juez |
| INTENT_THRESHOLD | 0.45 | similitud mínima para clasificar intención |
| MARKER_THRESHOLD | 0.55 | similitud mínima para detectar marcador |
| POLE_POS_THRESHOLD | 0.25 | umbral para bucket +1 |
| POLE_NEG_THRESHOLD | -0.25 | umbral para bucket -1 |
| CACHE_SIZE | 2000 | entradas máximas en LRU |
| HTTP_TIMEOUT_SECONDS | 60 | timeout para llamadas a Ollama |

## Tests

```bash
go test ./...
```

17 tests cubriendo: normalize, dot, mean, LRU eviction + LRU hit refresh,
SegmentSentences, PositionOf, morfología (palabras conocidas y sufijos),
trayectoria canónica vs exploratoria, polo bucketing, top-K heap, hash sha256,
tag bucketing, smoke test del reporte.

## Lo que NO hace

- No persiste sesiones entre requests (one-shot por diseño).
- No hay análisis de intención previa entre turnos. Si quieres ese componente,
  añade un campo opcional "previous_turns" al request y se computa la intención
  del último turno como `intencion_previa`.
- No usa HNSW. Búsqueda lineal contra 8957 vectores: ~2 ms en CPU moderna.
- No reemplaza spaCy: el análisis morfológico es ligero, basado en tabla
  pequeña de lemas + reglas de sufijo. Suficiente para los marcadores que
  esta tarea necesita.

## Por qué Go y no Python

- Latencia consistente sin GIL.
- Compilación a binario único, deploy trivial.
- Concurrencia barata (goroutines) si en algún momento procesas batches.
- Tipado fuerte: el JSON del juez se parsea con structs y errores explícitos.
- El cache LRU + el normalizado + el heap top-K se hacen sin dependencias.
