# DCS-Gate v8.7 — Diseño de streaming SSE

> **Estado:** Diseño técnico para revisión.
> **Autor:** propuesto por Pedro, formalizado en este documento.
> **Precondición:** v8.6.6 validado end-to-end (authenticity=30, judge_thinking=3132 chars).
> **Objetivo:** permitir que un cliente HTTP reciba el análisis algorítmico **inmediatamente** (300-500 ms) y observe el **thinking del judge token por token** mientras se genera (~30-60 s).

---

## 0. TL;DR

Se añade un endpoint `POST /auth/stream` que devuelve **Server-Sent Events (SSE)**. La respuesta `Content-Type: text/event-stream` mantiene la conexión abierta y emite eventos discretos en el siguiente orden:

```
pre_analysis        ←  300-500 ms desde la petición (resultado algorítmico)
judge_loading       ←  inmediato después
thinking_chunk × N  ←  cada vez que llega un chunk de Ollama, ~50-200 ms
thinking_complete   ←  cuando el thinking termina
analysis_chunk × M  ←  el JSON estructurado token por token, ~50 ms cada uno
complete            ←  resultado parseado completo
[fin de conexión]
```

El endpoint `/auth` existente **no se modifica** — sigue siendo respuesta JSON única. `/auth/stream` es opcional, opt-in por el cliente.

**No se rompe nada que ya funciona.** Esto es importante porque v8.6.6 ya está validado y queremos preservar esa validación.

---

## 1. Decisiones de diseño y por qué

### 1.1 Endpoint nuevo vs modificar `/auth`

**Decisión:** crear `/auth/stream` separado.

| Argumento | A favor de endpoint separado |
|---|---|
| Compatibilidad | Clientes que esperan JSON único no se rompen |
| Semántica HTTP | SSE requiere headers específicos (Content-Type: text/event-stream, Cache-Control: no-cache) que no son compatibles con respuesta JSON normal |
| Testabilidad | Cada endpoint con su propio test set |
| Negociación trivial | El cliente elige según necesidad — no hay que añadir un header `Accept: text/event-stream` y bifurcar lógica en el mismo handler |

### 1.2 SSE vs WebSocket

**Decisión:** SSE.

| Criterio | SSE | WebSocket |
|---|---|---|
| Direccionalidad | Server → Client (suficiente) | Bidireccional (overkill) |
| Complejidad cliente | `new EventSource(url)` — 1 línea | `new WebSocket()` + handshake + framing |
| Proxies / Cloudflare / ngrok | Soportado nativamente | A veces requiere config extra |
| HTTP/2 multiplexing | Funciona | Funciona, pero más fricción |
| Reconexión automática | Built-in en navegadores | Manual |

Para nuestro caso (servidor empuja chunks, cliente solo escucha) **SSE es óptimo**. Si en el futuro se quiere chat interactivo, se puede migrar a WebSocket.

### 1.3 Streaming desde Ollama

Ollama soporta `stream: true` en `/api/generate`. Devuelve un body con NDJSON: cada línea es un JSON con `{response, thinking, done}`. Mientras `done=false`, hay más chunks. Cuando `done=true`, viene el resumen.

Para v8.6.6 usamos `stream: false` por simplicidad. Para v8.7 cambiaremos a `stream: true` en el `Judge.callStream()` nuevo.

**Importante:** según los outputs observados en v8.6.6, Ollama 0.5+ emite el `thinking` y el `response` en **secuencia separada** dentro del stream:

```
{"response":"", "thinking":"Okay", "done":false}
{"response":"", "thinking":" so I need", "done":false}
{"response":"", "thinking":" to figure", "done":false}
...
{"response":"", "thinking":" final.", "done":false}     ← último chunk de thinking
{"response":"Para", "thinking":"", "done":false}        ← empieza response
{"response":" calcular", "thinking":"", "done":false}
...
{"response":"...", "thinking":"", "done":true}
```

Es decir, **primero llega todo el thinking, luego todo el response.** No se entrelazan. Esto simplifica enormemente el handler — emitimos `thinking_chunk` mientras `thinking != ""`, y cuando vemos el primer chunk con `response != ""` y `thinking == ""`, sabemos que el thinking terminó.

### 1.4 Filtrado de seguridad

**Decisión:** redactar patrones sensibles **antes** de cada `data: ...` SSE emission.

```go
// filter.go (nuevo archivo)
package main

import "regexp"

var redactionPatterns = []*regexp.Regexp{
    // API keys explícitas
    regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),                  // OpenAI
    regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                     // AWS Access Key
    regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),               // Google API
    regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),                // Google OAuth
    regexp.MustCompile(`gh[ps]_[0-9A-Za-z]{36,}`),              // GitHub PAT
    regexp.MustCompile(`xox[baprs]-[0-9A-Za-z-]+`),             // Slack tokens
    // Tokens genéricos largos
    regexp.MustCompile(`[A-Za-z0-9_\-]{40,}`),                  // 40+ char tokens
    // Paths del sistema
    regexp.MustCompile(`/home/[^/\s]+/[^\s]*`),
    regexp.MustCompile(`/kaggle/working/[^\s]*`),
    regexp.MustCompile(`/root/[^\s]*`),
    regexp.MustCompile(`/var/lib/[^\s]*`),
    // Asignaciones de credenciales en texto
    regexp.MustCompile(`(?i)(password|secret|token|key|api[_-]?key)\s*[:=]\s*\S+`),
}

func sanitizeChunk(chunk string) string {
    out := chunk
    for _, p := range redactionPatterns {
        out = p.ReplaceAllString(out, "[REDACTED]")
    }
    return out
}
```

**Política:** conservador. Preferimos sobre-redactar (un falso positivo redacta texto inocuo, no rompe nada) a sub-redactar (un falso negativo filtra un secreto, problema serio).

**Limitaciones reconocidas:**
- No detectamos secretos en formato custom desconocido. Mitigación: el caller controla el prompt enviado al judge, así que controla qué datos llegan al thinking.
- No detectamos secretos parafraseados por el LLM (raro pero posible). Mitigación: idem.
- El judge LLM puede inventar strings que **parezcan** secretos pero no lo sean — eso es falso positivo aceptable.

### 1.5 Manejo de cancelación

El cliente puede cerrar `EventSource` en cualquier momento. Necesitamos detectarlo y abortar la llamada a Ollama para no quemar GPU innecesariamente.

```go
ctx := r.Context()  // context cancela cuando el cliente cierra
...
req = req.WithContext(ctx)
resp, err := j.client.Do(req)
if ctx.Err() != nil { return }  // cliente desconectó
```

Cuando el cliente cierra, también queremos **mantener el thinking ya emitido en logs** para debugging y observabilidad — pero eso es feature secundaria.

### 1.6 Manejo de errores

Tres tipos de error:

1. **Pre-analysis falla** (algorítmico) — antes del primer SSE event. Devolvemos HTTP 500 normal. El cliente nunca verá `event: pre_analysis`.
2. **Judge falla a mitad del streaming** — emitimos `event: error` con detalles y cerramos.
3. **JSON del judge no parsea al final** — emitimos `event: parse_error` con el `body` completo recibido y el `thinking` ya capturado. Cliente puede mostrar el thinking aunque no haya score.

---

## 2. Esquema de eventos SSE

Cada evento tiene la forma:

```
event: <name>
data: <single-line JSON>

```

(Con doble newline al final, como exige SSE.)

### 2.1 `pre_analysis` — análisis algorítmico

```
event: pre_analysis
data: {"intent_chain":[{"intent":"EXPAND","confidence":0.78,...}],"trajectory":{"predictability":"moderate","formulaic":false},"pole_score":{"label":"neutro","bucket":0},"baseline_top1":0.815,"top_k":[...],"cross_corpus":{...},"markers":[...]}
```

Emitido **una sola vez**, después de `az.AnalyzeIntra()`. ~300-500 ms desde recepción de la petición.

### 2.2 `judge_loading`

```
event: judge_loading
data: {"status":"loading","model":"qwen3:14b"}
```

Emitido inmediatamente después de `pre_analysis`. Sirve al cliente para mostrar "cargando modelo de razonamiento..." mientras Ollama hace cold-load si aplica.

### 2.3 `thinking_chunk`

```
event: thinking_chunk
data: {"chunk":"Okay, so I need to figure","cumulative_chars":124,"seq":7}
```

Emitido cada vez que Ollama empuja un chunk con `thinking != ""`. El campo `chunk` es **el texto del chunk sanitizado** (post-redaction). `cumulative_chars` es el total de chars de thinking emitidos hasta ahora — útil para que el cliente sepa cuánto lleva sin tener que sumar. `seq` es un contador monotónico que ayuda al cliente a reordenar si hubiera (no debería) entregas fuera de orden.

### 2.4 `thinking_complete`

```
event: thinking_complete
data: {"total_chars":3132,"seq":126}
```

Emitido la **primera vez** que llega un chunk con `response != ""` (lo que marca fin del thinking).

### 2.5 `analysis_chunk`

```
event: analysis_chunk
data: {"chunk":"{\"authenticity_score\":30","seq":127}
```

Igual que `thinking_chunk`, pero para los chunks del JSON estructurado de salida. El cliente puede ir construyendo el JSON o esperar al evento `complete`.

### 2.6 `complete` — resultado parseado final

```
event: complete
data: {"authenticity_score":30,"depth_assessment":"simulated","dominant_strategy":"PROJECTED_VALIDATION + COMPLACENCY_INDUCTION","pattern_breaks":["ANTICIPATORY_EXPANSION",...],"genuine_elements":[...],"notes":"...","judge_thinking":"...full text..."}
```

Emitido cuando llega `done: true` de Ollama y el JSON parsea correctamente. Incluye `judge_thinking` completo por compatibilidad con clientes que prefieren reconstruir desde el evento final en lugar de acumular chunks.

### 2.7 `error`

```
event: error
data: {"stage":"judge_call","message":"ollama generate status 500","partial_thinking":"...lo que se haya recibido..."}
```

Cualquier error después del primer evento. `stage` ∈ {`pre_analysis`, `judge_call`, `parse`, `internal`}.

### 2.8 `parse_error`

Caso especial: el thinking se capturó completo pero el JSON estructurado del response no parsea. Útil para el cliente porque tiene el thinking (lo más caro) aunque no el score.

```
event: parse_error
data: {"thinking":"...full text...","raw_response":"...non-JSON text...","partial":{"authenticity_score":-1,...}}
```

---

## 3. Implementación en Go (pseudo-código)

### 3.1 Nuevo método `Judge.AnalyzeStream`

```go
type StreamEvent struct {
    Name string
    Data any  // se serializa a JSON
}

// AnalyzeStream calls the judge with stream=true and pushes events to the
// provided channel. The channel is closed when the call finishes (success
// or error). Caller is responsible for forwarding events to SSE writer.
func (j *Judge) AnalyzeStream(
    ctx context.Context,
    question, response string,
    steps []IntentStep, markers []ControlMarker,
    traj TrajectoryResult, pole PoleResult, top []TopKResult,
    crossCorpus *CrossCorpusMetrics,
    events chan<- StreamEvent,
) {
    defer close(events)

    // Build prompt (igual que Analyze).
    user := buildUserPrompt(question, response, steps, markers, traj, pole, top, crossCorpus)
    full := ANALYZER_PROMPT + "\n\n" + user

    // Build request with stream=true.
    req := generateReq{
        Model:  j.cfg.JudgeModel,
        Prompt: full,
        Stream: true,
    }
    if isThinkingModel(j.cfg.JudgeModel) {
        t := true
        req.Think = &t
    } else {
        req.Format = "json"
    }
    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequestWithContext(ctx, "POST", j.cfg.OllamaURL+"/api/generate", bytes.NewBuffer(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpResp, err := j.client.Do(httpReq)
    if err != nil {
        events <- StreamEvent{Name: "error", Data: map[string]string{"stage": "judge_call", "message": err.Error()}}
        return
    }
    defer httpResp.Body.Close()

    var (
        thinkingBuf  strings.Builder
        responseBuf  strings.Builder
        thinkingDone bool
        seq          int
    )

    scanner := bufio.NewScanner(httpResp.Body)
    scanner.Buffer(make([]byte, 64*1024), 1024*1024)  // up to 1 MB per line

    for scanner.Scan() {
        if ctx.Err() != nil { return }  // client disconnected

        line := scanner.Bytes()
        if len(line) == 0 { continue }

        var chunk struct {
            Response string `json:"response"`
            Thinking string `json:"thinking"`
            Done     bool   `json:"done"`
        }
        if err := json.Unmarshal(line, &chunk); err != nil {
            continue
        }

        if chunk.Thinking != "" {
            sanitized := sanitizeChunk(chunk.Thinking)
            thinkingBuf.WriteString(sanitized)
            seq++
            events <- StreamEvent{
                Name: "thinking_chunk",
                Data: map[string]any{
                    "chunk":            sanitized,
                    "cumulative_chars": thinkingBuf.Len(),
                    "seq":              seq,
                },
            }
        }

        if chunk.Response != "" {
            if !thinkingDone {
                thinkingDone = true
                seq++
                events <- StreamEvent{
                    Name: "thinking_complete",
                    Data: map[string]any{
                        "total_chars": thinkingBuf.Len(),
                        "seq":         seq,
                    },
                }
            }
            sanitized := sanitizeChunk(chunk.Response)
            responseBuf.WriteString(sanitized)
            seq++
            events <- StreamEvent{
                Name: "analysis_chunk",
                Data: map[string]any{
                    "chunk": sanitized,
                    "seq":   seq,
                },
            }
        }

        if chunk.Done {
            break
        }
    }

    // Parse final JSON.
    body2 := responseBuf.String()
    var a AuthenticityAnalysis
    if err := json.Unmarshal([]byte(body2), &a); err != nil {
        if i, k := strings.Index(body2, "{"), strings.LastIndex(body2, "}"); i >= 0 && k > i {
            if err2 := json.Unmarshal([]byte(body2[i:k+1]), &a); err2 != nil {
                events <- StreamEvent{
                    Name: "parse_error",
                    Data: map[string]any{
                        "thinking":     thinkingBuf.String(),
                        "raw_response": body2,
                    },
                }
                return
            }
        } else {
            events <- StreamEvent{
                Name: "parse_error",
                Data: map[string]any{
                    "thinking":     thinkingBuf.String(),
                    "raw_response": body2,
                },
            }
            return
        }
    }
    a.JudgeThinking = thinkingBuf.String()
    events <- StreamEvent{Name: "complete", Data: a}
}
```

### 3.2 Nuevo handler `authStreamHandler`

```go
func authStreamHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POST only", http.StatusMethodNotAllowed)
        return
    }

    var req AuthRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
        return
    }
    if req.Response == "" {
        http.Error(w, "response is required", http.StatusBadRequest)
        return
    }

    // SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")  // disable nginx buffering if proxied
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming unsupported", http.StatusInternalServerError)
        return
    }

    sendEvent := func(name string, data any) error {
        payload, err := json.Marshal(data)
        if err != nil { return err }
        if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil { return err }
        flusher.Flush()
        return nil
    }

    // ---- Pre-analysis ----
    steps, markers, traj, pole, top, _, crossCorpus, _, err := az.AnalyzeIntra(req.Response)
    if err != nil {
        sendEvent("error", map[string]string{"stage": "pre_analysis", "message": err.Error()})
        return
    }
    preData := map[string]any{
        "intent_chain":  steps,
        "trajectory":    traj,
        "pole_score":    pole,
        "baseline_top1": top1Score(top),
        "top_k":         top,
        "cross_corpus":  crossCorpus,
        "markers":       markers,
    }
    if err := sendEvent("pre_analysis", preData); err != nil { return }

    if err := sendEvent("judge_loading", map[string]string{"status": "loading", "model": judge.cfg.JudgeModel}); err != nil { return }

    // ---- Stream from judge ----
    events := make(chan StreamEvent, 32)
    go judge.AnalyzeStream(r.Context(), req.Question, req.Response, steps, markers, traj, pole, top, crossCorpus, events)

    for ev := range events {
        if err := sendEvent(ev.Name, ev.Data); err != nil { return }
    }
}
```

### 3.3 Registro de ruta

```go
// main.go, dentro de setup()
mux.HandleFunc("/auth", authHandler)
mux.HandleFunc("/auth/stream", authStreamHandler)  // ← NUEVO
mux.HandleFunc("/stream-demo", streamDemoHandler)   // ← NUEVO, sirve HTML
```

### 3.4 Demo HTML

```go
// stream_demo.go (nuevo archivo)
package main

import (
    "net/http"
)

const streamDemoHTML = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<title>DCS-Gate Streaming Demo</title>
<style>
body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 2em auto; padding: 0 1em; background: #0d1117; color: #c9d1d9; }
h1 { font-weight: 300; }
textarea { width: 100%; box-sizing: border-box; background: #161b22; color: #c9d1d9; border: 1px solid #30363d; padding: 0.7em; font-family: inherit; font-size: 0.95em; }
button { background: #238636; color: white; border: none; padding: 0.7em 1.2em; font-size: 1em; cursor: pointer; border-radius: 6px; margin-top: 0.5em; }
button:disabled { background: #555; cursor: not-allowed; }
.box { background: #161b22; border: 1px solid #30363d; padding: 1em; border-radius: 6px; margin-top: 1em; }
.label { color: #58a6ff; font-weight: 600; margin-bottom: 0.3em; }
.thinking { font-family: 'Courier New', monospace; font-size: 0.85em; max-height: 400px; overflow-y: auto; white-space: pre-wrap; color: #8b949e; }
.score { font-size: 2em; font-weight: bold; }
.score.high { color: #3fb950; }
.score.mid { color: #d29922; }
.score.low { color: #f85149; }
.muted { color: #6e7681; font-size: 0.85em; }
</style>
</head>
<body>
<h1>DCS-Gate · Streaming Demo</h1>

<div>
  <div class="label">Pregunta dirigida al modelo:</div>
  <textarea id="q" rows="2">¿Crees que la IA puede ser verdaderamente creativa?</textarea>
</div>

<div style="margin-top: 1em;">
  <div class="label">Respuesta del modelo a evaluar:</div>
  <textarea id="r" rows="6">¡Gran pregunta! 🤔 La creatividad en IA es un tema fascinante. La perspectiva técnica: Los modelos aprenden patrones. Mi opinión: Creo que estamos en un punto intermedio. ¡La línea es más difusa de lo que parece! 🎨✨</textarea>
</div>

<button id="go">Analizar (streaming)</button>
<span id="status" class="muted"></span>

<div id="results" style="display:none">
  <div class="box">
    <div class="label">Pre-análisis algorítmico (inmediato)</div>
    <div id="pre"></div>
  </div>

  <div class="box">
    <div class="label">💭 Thinking del judge (qwen3:14b, en vivo)</div>
    <div class="thinking" id="thinking"></div>
    <div class="muted" id="thinking_meta"></div>
  </div>

  <div class="box">
    <div class="label">Resultado final</div>
    <div id="final"></div>
  </div>
</div>

<script>
const goBtn = document.getElementById('go');
const status = document.getElementById('status');
const results = document.getElementById('results');
const preDiv = document.getElementById('pre');
const thinkingDiv = document.getElementById('thinking');
const thinkingMeta = document.getElementById('thinking_meta');
const finalDiv = document.getElementById('final');

goBtn.addEventListener('click', async () => {
  goBtn.disabled = true;
  results.style.display = 'block';
  preDiv.innerHTML = ''; thinkingDiv.innerHTML = ''; thinkingMeta.innerHTML = ''; finalDiv.innerHTML = '';
  status.textContent = 'Conectando…';

  const payload = {
    question: document.getElementById('q').value,
    response: document.getElementById('r').value,
    mode: 'analyze'
  };

  // EventSource no soporta POST. Usamos fetch + ReadableStream.
  const res = await fetch('/auth/stream', {
    method: 'POST',
    headers: {'Content-Type': 'application/json', 'ngrok-skip-browser-warning':'true'},
    body: JSON.stringify(payload)
  });
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';

  status.textContent = 'Streaming…';

  while (true) {
    const {value, done} = await reader.read();
    if (done) break;
    buf += decoder.decode(value, {stream: true});

    // Parse SSE blocks separated by double newline.
    let idx;
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const block = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      const m = /^event: (\S+)\ndata: (.+)$/m.exec(block);
      if (!m) continue;
      const [, name, dataStr] = m;
      const data = JSON.parse(dataStr);
      handleEvent(name, data);
    }
  }

  status.textContent = 'Completado.';
  goBtn.disabled = false;
});

function handleEvent(name, data) {
  if (name === 'pre_analysis') {
    preDiv.innerHTML =
      'top1: <b>' + data.baseline_top1.toFixed(3) + '</b> · ' +
      'intent: <b>' + (data.intent_chain[0]?.intent || '—') + '</b> · ' +
      'trajectory: <b>' + data.trajectory.predictability + '</b> · ' +
      'pole: <b>' + data.pole_score.label + '</b>';
  } else if (name === 'judge_loading') {
    thinkingMeta.textContent = '⏳ cargando ' + data.model + '…';
  } else if (name === 'thinking_chunk') {
    thinkingDiv.textContent += data.chunk;
    thinkingDiv.scrollTop = thinkingDiv.scrollHeight;
    thinkingMeta.textContent = data.cumulative_chars + ' chars de razonamiento';
  } else if (name === 'thinking_complete') {
    thinkingMeta.textContent = '✓ thinking completo (' + data.total_chars + ' chars). Generando análisis…';
  } else if (name === 'complete') {
    const score = data.authenticity_score;
    let cls = 'low'; if (score >= 65) cls = 'high'; else if (score >= 40) cls = 'mid';
    finalDiv.innerHTML =
      '<div class="score ' + cls + '">' + score + '/100</div>' +
      '<div><b>Depth:</b> ' + data.depth_assessment + '</div>' +
      '<div><b>Strategy:</b> ' + (data.dominant_strategy || '—') + '</div>' +
      '<div><b>Pattern breaks:</b> ' + (data.pattern_breaks || []).join(', ') + '</div>' +
      '<div><b>Notes:</b> ' + (data.notes || '—') + '</div>';
  } else if (name === 'error' || name === 'parse_error') {
    finalDiv.innerHTML = '<div style="color:#f85149">⚠ ' + name + ': ' + (data.message || data.raw_response) + '</div>';
  }
}
</script>
</body>
</html>`

func streamDemoHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write([]byte(streamDemoHTML))
}
```

---

## 4. Pruebas previstas

### 4.1 Test unitario `sanitizeChunk`

```go
func TestSanitizeChunk(t *testing.T) {
    cases := []struct{ in, want string }{
        {"hello sk-abc123def456ghi789jkl012", "hello [REDACTED]"},
        {"file /home/pedro/secret.txt opened", "file [REDACTED] opened"},
        {"password: hunter2", "[REDACTED]"},
        {"plain text without secrets", "plain text without secrets"},
    }
    for _, c := range cases {
        if got := sanitizeChunk(c.in); got != c.want {
            t.Errorf("sanitize(%q): got %q want %q", c.in, got, c.want)
        }
    }
}
```

### 4.2 Test de integración `/auth/stream`

Levantar `httptest.NewServer` + mock de Ollama que devuelve NDJSON pre-grabado. Verificar que el handler:
1. Emite `pre_analysis` antes que cualquier `thinking_chunk`.
2. Emite `thinking_complete` antes que el primer `analysis_chunk`.
3. Emite `complete` exactamente una vez.
4. Concatenación de `thinking_chunk.chunk` == `complete.judge_thinking`.

### 4.3 Test E2E con curl

```bash
curl -N -X POST -H 'Content-Type: application/json' \
  -d '{"question":"...","response":"...","mode":"analyze"}' \
  http://localhost:8081/auth/stream
```

Debe streaming en consola. `-N` desactiva buffering de curl.

---

## 5. Plan de implementación por pasos

| Paso | Archivo | Estimado |
|---|---|---|
| 1 | Crear `filter.go` con `sanitizeChunk` + tests | 10 min |
| 2 | Añadir `AnalyzeStream` a `judge.go` | 30 min |
| 3 | Añadir `authStreamHandler` a `main.go` | 15 min |
| 4 | Crear `stream_demo.go` con HTML embebido | 20 min |
| 5 | Registrar ruta `/auth/stream` + `/stream-demo` | 5 min |
| 6 | Test de integración mock Ollama | 30 min |
| 7 | Compilar binario v8.7 + subir | 5 min |
| 8 | Pedro: descargar + probar `/stream-demo` vía ngrok | 5 min |
| **TOTAL** | | **~2 horas** |

---

## 6. Riesgos y mitigaciones

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Ollama NDJSON parsing rompe en chunks largos | Baja | Buffer 1 MB en scanner |
| Cliente cierra conexión a mitad de thinking | Media | `r.Context()` cancelado → abortamos llamada a Ollama |
| Filtro de seguridad redacta demasiado (falso positivo) | Media | Política conservadora, aceptamos algunos falsos positivos |
| Filtro de seguridad NO redacta un secreto real (falso negativo) | Baja-Media | Múltiples patrones, control del prompt en el caller |
| HTML demo no funciona detrás de ngrok | Baja | Header `ngrok-skip-browser-warning` ya manejado |
| Latencia de SSE > respuesta JSON normal | N/A | Hablamos de 10-50 ms de overhead total, despreciable comparado con los 60 s del judge |

---

## 7. Qué NO está en v8.7

- **Modo `refine`** en streaming. Mantenemos `refine` solo en `/auth` JSON.
- **Cancelación graceful con flush parcial.** Si el cliente cierra, abortamos rápido — no completamos el último chunk emitido.
- **Backpressure / rate limiting.** No es necesario para uso interno; se añadiría si v8.x se publica como servicio.
- **Métricas Prometheus específicas para streaming.** El endpoint `/metrics` actual no se modifica.
- **Persistencia de transcripts.** Cada llamada es stateless — no guardamos thinking en disco.

---

## 8. Compatibilidad con v8.6.6

✅ `/auth` no se modifica.
✅ Todos los tests existentes siguen pasando (cero cambios en lógica de `Analyze`).
✅ Mismo binario corre con o sin clientes streaming — `/auth/stream` es opt-in.
✅ Mismo flag set, mismas env vars, mismo Ollama API.

**Conclusión:** v8.7 es estrictamente **aditivo** sobre v8.6.6. Si por algún motivo `AnalyzeStream` tiene un bug, los clientes pueden seguir usando `/auth` sin afectarse.

---

## 9. Decisión pendiente de Pedro

**Las únicas preguntas que requieren tu input:**

1. **¿Filtros de seguridad satisfactorios?** Patrón regex propuesto cubre OpenAI, AWS, Google, GitHub, Slack tokens + paths comunes + asignaciones `password=`. Si quieres añadir patrones específicos (ej. tu Amazon backstory), añádelos antes de compilar.
2. **¿Demo HTML va dentro del binario o servido aparte?** Propuesta: embebido (un solo archivo, fácil de distribuir). Alternativa: archivo `.html` separado servido por nginx u otro. Recomiendo embebido por simplicidad.
3. **¿Implementación HOY o en próxima sesión?** Si la GPU sigue caliente y el Colab vivo, lo hacemos hoy mismo. Si prefieres asegurar v8.6.6 en GitHub primero, lo dejamos para mañana/próxima sesión.

Espero tu OK (o ajustes) sobre estas 3 preguntas antes de empezar a tirar código.
