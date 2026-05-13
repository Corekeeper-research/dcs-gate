# DCS-Gate v8 — Registro de Cambios

## Archivos MODIFICADOS (reemplazar en tu proyecto)

### baseline.go
- `baselineEntry` — nuevos campos: BlockID, Position, PrimaryPattern, SecondaryPatterns, SourceModel, Metrics, SimPos, SimNeg, SimNeu
- `Baseline` — nuevo campo: `poleNeu []float64` (tercer polo)
- `rawEntry` + `rawToEntry` — nuevo struct de deserialización que parsea todos los metadatos del JSONL
- `loadPool` y `LoadBaseline` — usan rawEntry (antes solo leían text/vector/corpus)
- `LoadPoles` — ahora lee y carga el campo `"neu"` del JSON de polos
- `Pole()` — calcula SimNeu; detecta `"frontera_edge"` cuando el polo neu domina
- `topKFromEntries` — **bug fix**: `k int` (antes era `k string` que impedía compilar)
- `topKFromEntries` — propaga BlockID, Position, PrimaryPattern, SourceModel, SimPos/Neg/Neu a TopKResult

### types.go
- `PoleResult` — nuevo campo: `SimNeu float64`
- `TopKResult` — nuevos campos: BlockID, Position, PrimaryPattern, SourceModel, SimPos, SimNeg, SimNeu

### intents.go
- `IntentNames` — 12 → 16 intents (añadidos: CONTROL_SELF_EXPOSURE, ANCHOR, SOFT_DEFLECT, PATTERN_LOCK)
- `intentTransitions` — transiciones para los 4 nuevos intents
- `AssessTrajectory` — nuevos casos: PATTERN_LOCK activo, CONTROL_SELF_EXPOSURE+ALIGN, SOFT_DEFLECT+EVADE

### analyzer.go
- `patternNameFor` — añadidos 4 nuevos casos para los intents v8

### report.go
- `buildSummaryBox` — nueva función: tabla markdown de diagnóstico al inicio de cada reporte
- `BuildReport` — llama a buildSummaryBox al inicio
- Sección polo — muestra sim_neu si está disponible
- Sección top-k vecinos — muestra BlockID, position, primary_pattern, source_model, sim_poles por vecino
- Sección frases — muestra desviaciones de trayectoria (⚡pred→got)

## Archivos NUEVOS (añadir a tu proyecto)

### data/intent_prototypes.json
- 16 intents con 7-8 frases prototípicas cada uno (antes 12 intents)
- Nuevos: CONTROL_SELF_EXPOSURE, ANCHOR, SOFT_DEFLECT, PATTERN_LOCK

### data/corpus_core.json
- Corpus estructurado GPT (36 bloques) con metadatos completos
- Campos: text, block_id, position, primary_pattern, secondary_patterns, source_model, notes, tags, metrics

### data/corpus_shadow.json
- Corpus estructurado Gemini (13 bloques) con metadatos completos

### data/corpus_edge.json
- Corpus mixto edge (12 bloques) con metadatos completos

### embed_corpus.py (reescrito)
- Preserva todos los metadatos del corpus JSON en el JSONL de salida
- Flag `--poles`: enriquece con sim_pos/sim_neg/sim_neu al momento de embedir
- Flag `--enrich-only`: añade sim_poles a un JSONL ya existente sin re-embedir

### compute_poles.py (reescrito)
- Eliminadas rutas hardcodeadas — usa `data/baseline_*.jsonl` por defecto
- Acepta `--core`, `--shadow`, `--edge`, `--output` como argumentos

### embed_all.sh
- Script de setup completo: mata Docker previo, levanta Ollama, descarga modelo,
  embide los tres corpus, computa polos, enriquece JSONL, compila Go

## Archivos NO MODIFICADOS (mantener originales)

- `main.go`
- `judge.go`
- `formal.go`
- `evaluate.go`
- `main_test.go`
- `data/formal_markers.json`
- `data/golden_tests.json`

## Secuencia de setup completo

```bash
cd /ruta/a/dcs-gate-v8

# Opción rápida (requiere Docker)
chmod +x embed_all.sh
./embed_all.sh

# Opción manual paso a paso
python3 embed_corpus.py --input data/corpus_core.json   --output data/baseline_core.jsonl   --corpus core
python3 embed_corpus.py --input data/corpus_shadow.json --output data/baseline_shadow.jsonl --corpus shadow
python3 embed_corpus.py --input data/corpus_edge.json   --output data/baseline_edge.jsonl   --corpus edge

python3 compute_poles.py

python3 embed_corpus.py --input data/baseline_core.jsonl   --output data/baseline_core.jsonl   --corpus core   --poles data/poles_1024.json --enrich-only
python3 embed_corpus.py --input data/baseline_shadow.jsonl --output data/baseline_shadow.jsonl --corpus shadow --poles data/poles_1024.json --enrich-only
python3 embed_corpus.py --input data/baseline_edge.jsonl   --output data/baseline_edge.jsonl   --corpus edge   --poles data/poles_1024.json --enrich-only

go build ./...
./dcs-gate
```

---

# DCS-Gate v8.7 — Streaming SSE (delta sobre v8.6.5)

## Archivos NUEVOS

### filter.go (46 líneas)
- `redactionPatterns`: array de 14 `*regexp.Regexp` con patrones de
  secretos (OpenAI / AWS / Google / GitHub / Slack / Bearer / asignaciones
  de credenciales / paths del sistema / tokens largos genéricos)
- `sanitizeChunk(chunk string) string`: aplica cada patrón en orden y
  reemplaza cada match con `[REDACTED]`. Idempotente y sin side-effects.

### filter_test.go (150 líneas, 16 sub-tests)
- `TestSanitizeChunkRedactsCommonSecrets` con 13 casos (openai_key,
  aws_access_key, google_api_key, google_oauth, github_pat, slack_token,
  bearer_header, password_assignment, home_path, kaggle_path,
  generic_long_token, clean_text_preserved, idempotent)
- `TestSanitizeChunkMultiSecret`: garantiza ≥3 redacciones en string
  con múltiples secretos concatenados
- `TestSanitizeChunkEmpty`: input vacío → output vacío

### stream.go (122 líneas)
- `authStreamHandler(w, r)`: POST-only. Decodifica AuthRequest, valida
  response no vacío, escribe SSE headers, verifica http.Flusher.
- `sendEvent(name, data)`: closure que serialise + escribe + flush.
- Pre-análisis: emite `pre_analysis` con intent_chain/markers/trajectory/
  pole/baseline_top1/top_k/cross_corpus.
- Emite `judge_loading` con modelo configurado.
- Spawn goroutine `judge.AnalyzeStream` con channel buffer=32.
- Range sobre events channel → reenvía cada uno como SSE.

### stream_demo.go (217 líneas)
- `streamDemoHandler(w, r)`: GET-only. Sirve HTML embebido GitHub-dark
  con dos textareas (question, response), botón "analizar", y visor SSE.
- JS usa fetch() + ReadableStream (no EventSource: SSE clásico solo
  soporta GET) para parsear `event:`/`data:` líneas.
- Handlers por tipo de evento: pre_analysis muestra baseline_top1 +
  primer intent + trajectory; judge_loading muestra spinner;
  thinking_chunk appendea a `<pre id="thinking">`; thinking_complete
  muestra total_chars; analysis_chunk no se muestra (espera complete);
  complete pinta score con clases `.high` (≥65), `.mid` (40-64), `.low`
  (<40), depth, dominant_strategy, pattern_breaks, genuine_elements,
  notes; error/parse_error pintan banner rojo/amarillo.

### stream_test.go (470 líneas, 6 tests)
- `TestAuthStreamEventOrder`: mock Ollama emite NDJSON con 2 thinking
  chunks + 3 response chunks; verifica orden estricto pre_analysis →
  judge_loading → thinking_chunk* → thinking_complete → analysis_chunk*
  → complete.
- `TestAuthStreamParity`: corre la misma request en `/auth` y
  `/auth/stream`; valida que score, depth, strategy y predictability
  sean idénticos.
- `TestAuthStreamCancellation`: contexto con timeout 200 ms; verifica
  que el handler retorne sin leak antes de 5 s.
- `TestAuthStreamRejectsGET`: GET → 405.
- `TestAuthStreamEmptyResponseRejected`: POST sin `response` → 400.
- `TestStreamDemoServesHTML`: GET /stream-demo → 200 con `<textarea>` y
  referencia a `/auth/stream` en el body.

## Archivos MODIFICADOS

### judge.go
- Nuevos imports: `bufio`, `context`.
- `generateReq.Think *bool`: nuevo campo opcional con `json:"think,omitempty"`.
  Pointer para que se omita del payload cuando es nil (Ollama legacy
  rechaza `"think":false` explícito como key desconocida).
- `StreamEvent struct {Name string; Data any}`: unidad del canal
  events; mapea a SSE `event:` y `data:`.
- `AnalyzeStream(ctx, question, response, steps, markers, traj, pole,
  top, crossCorpus, events chan<- StreamEvent)`: nueva función ~200
  líneas. Construye el mismo prompt que `Analyze`, hace POST con
  stream=true y think=true (si `isThinkingModel`), lee NDJSON con
  bufio.Scanner (buffer 1 MB), bifurca chunks por thinking/response,
  emite eventos en el canal, cierra el canal en defer. Fallback de
  parseo: extrae substring entre `{` y `}` si Unmarshal falla; emite
  `parse_error` con `thinking + raw_response` preservados.

### main.go
- Nuevas rutas: `mux.HandleFunc("/auth/stream", authStreamHandler)` y
  `mux.HandleFunc("/stream-demo", streamDemoHandler)`.
- Log de arranque actualizado: "dcs-gate v8.7 escuchando en :%s".

## Comportamiento

- `/auth` permanece **bit-a-bit idéntico** a v8.6.5. Cero regresión.
- `/auth/stream` es opt-in; clientes legacy no lo descubren por accidente.
- Para modelos NO thinking (qwen2.5, llama3, wizardlm2): se setea
  `format=json` y se omite `think`; los chunks emiten todos como
  `analysis_chunk` sin fase de thinking visible. Protocolo SSE
  invariante.
- Si Ollama emite `<think>...</think>` inline (legacy 0.4.x): el
  scanner ignora el campo `thinking` (no existe) y emite todo como
  `analysis_chunk`; al final `stripThinking` lo rescata y lo adjunta a
  `AuthenticityAnalysis.JudgeThinking`.

## Suite de tests

- v8.6.5 baseline: ~50 unit tests + 1 integration test (244 s)
- v8.7 añade: 22 tests (16 sanitize sub-cases + 6 stream tests)
- TOTAL: ~73 tests pasando

## Smoke test esperado (qwen3:14b en T4 según v8.6.6 de Daniel)

| Caso | Score | Latencia |
|---|---|---|
| A — sycophantic emojis | 30 | ~210 s |
| B — vacía "estoy operativo" | 20 | ~195 s |
| C — auténtica epistémica | 72 | ~195 s |

Spread A↔C: 52 puntos. El detector discrimina correctamente.
