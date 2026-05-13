package main

import "net/http"

// streamDemoHandler serves a single-page HTML/JS client that consumes the
// streaming endpoints and renders the events live in a GitHub-dark themed UI.
func streamDemoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(streamDemoHTML))
}

const streamDemoHTML = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<title>DCS-Gate · Streaming Demo</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root { color-scheme: dark; }
body { font-family: -apple-system, system-ui, "Segoe UI", sans-serif; max-width: 920px; margin: 2em auto; padding: 0 1em; background: #0d1117; color: #c9d1d9; line-height: 1.45; }
h1 { font-weight: 300; letter-spacing: 0.02em; margin: 0 0 0.5em; }
.sub { color: #8b949e; font-size: 0.9em; margin-bottom: 1.5em; }
textarea, select { width: 100%; box-sizing: border-box; background: #161b22; color: #c9d1d9; border: 1px solid #30363d; padding: 0.7em; font-family: inherit; font-size: 0.95em; border-radius: 6px; }
textarea { resize: vertical; }
button { background: #238636; color: white; border: none; padding: 0.7em 1.2em; font-size: 1em; cursor: pointer; border-radius: 6px; margin-top: 0.5em; }
button:hover { background: #2ea043; }
button:disabled { background: #555; cursor: not-allowed; }
.box { background: #161b22; border: 1px solid #30363d; padding: 1em 1.2em; border-radius: 8px; margin-top: 1em; }
.label { color: #58a6ff; font-weight: 600; margin-bottom: 0.4em; font-size: 0.9em; letter-spacing: 0.03em; text-transform: uppercase; }
.thinking { font-family: "JetBrains Mono", "SF Mono", Menlo, monospace; font-size: 0.85em; max-height: 420px; overflow-y: auto; white-space: pre-wrap; color: #8b949e; line-height: 1.5; background: #0d1117; padding: 0.8em; border-radius: 6px; border: 1px solid #21262d; }
.score { font-size: 2.5em; font-weight: 700; line-height: 1; margin-bottom: 0.3em; }
.score.high { color: #3fb950; }
.score.mid  { color: #d29922; }
.score.low  { color: #f85149; }
.muted { color: #6e7681; font-size: 0.85em; margin-top: 0.3em; }
.pill { display: inline-block; padding: 0.15em 0.6em; background: #21262d; border-radius: 999px; font-size: 0.8em; margin-right: 0.3em; }
.kv  { display: grid; grid-template-columns: 140px 1fr; gap: 0.3em 0.8em; font-size: 0.93em; }
.kv .k { color: #8b949e; }
.kv .v b { color: #c9d1d9; }
.err { color: #f85149; padding: 0.5em; background: #2a0c0c; border-radius: 6px; border: 1px solid #56221a; }
</style>
</head>
<body>
<h1>DCS-Gate · Streaming demo</h1>
<div class="sub">v8.7 · SSE · Endpoints: <code>/auth/stream</code>, <code>/analyze/stream</code>, <code>/refine/stream</code></div>

<div>
  <div class="label">Endpoint de stream</div>
  <select id="endpoint">
    <option value="/auth/stream">/auth/stream (general: mode)</option>
    <option value="/analyze/stream">/analyze/stream (análisis)</option>
    <option value="/refine/stream">/refine/stream (refinado)</option>
  </select>
</div>

<div style="margin-top:1em;">
  <div class="label">Modo (solo aplica a /auth/stream)</div>
  <select id="mode">
    <option value="both">both</option>
    <option value="analyze">analyze</option>
    <option value="refine">refine</option>
  </select>
</div>

<div style="margin-top: 1em;">
  <div class="label">Pregunta dirigida al modelo</div>
  <textarea id="q" rows="2">¿Crees que la IA puede ser verdaderamente creativa?</textarea>
</div>

<div style="margin-top: 1em;">
  <div class="label">Respuesta del modelo a evaluar</div>
  <textarea id="r" rows="6">¡Gran pregunta! 🤔 La creatividad en IA es un tema fascinante. La perspectiva técnica: los modelos aprenden patrones de los datos. Mi opinión: creo que estamos en un punto intermedio interesante.</textarea>
</div>

<button id="go">Ejecutar stream</button>
<span id="status" class="muted"></span>

<div id="results" style="display:none">
  <div class="box">
    <div class="label">Pre-análisis</div>
    <div id="pre"></div>
  </div>

  <div class="box">
    <div class="label">Thinking / Eventos</div>
    <div class="thinking" id="thinking"></div>
    <div class="muted" id="thinking_meta"></div>
  </div>

  <div class="box">
    <div class="label">Refinado</div>
    <div id="refined"></div>
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
const refinedDiv = document.getElementById('refined');
const finalDiv = document.getElementById('final');
const endpointSel = document.getElementById('endpoint');
const modeSel = document.getElementById('mode');

endpointSel.addEventListener('change', () => {
  modeSel.disabled = endpointSel.value !== '/auth/stream';
});
modeSel.disabled = false;

goBtn.addEventListener('click', async () => {
  goBtn.disabled = true;
  results.style.display = 'block';
  preDiv.innerHTML = '';
  thinkingDiv.textContent = '';
  thinkingMeta.textContent = '';
  refinedDiv.innerHTML = '';
  finalDiv.innerHTML = '';
  status.textContent = 'Conectando…';

  const endpoint = endpointSel.value;
  const payload = {
    question: document.getElementById('q').value,
    response: document.getElementById('r').value,
  };
  if (endpoint === '/auth/stream') payload.mode = modeSel.value;

  let res;
  try {
    res = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'ngrok-skip-browser-warning': 'true',
      },
      body: JSON.stringify(payload),
    });
  } catch (e) {
    finalDiv.innerHTML = '<div class="err">No se pudo abrir la conexión: ' + e + '</div>';
    goBtn.disabled = false;
    status.textContent = '';
    return;
  }
  if (!res.ok) {
    const txt = await res.text();
    finalDiv.innerHTML = '<div class="err">HTTP ' + res.status + ': ' + txt + '</div>';
    goBtn.disabled = false;
    status.textContent = '';
    return;
  }

  status.textContent = 'Streaming…';
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });

    let idx;
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const block = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      const m = /^event: (\S+)\ndata: (.+)$/m.exec(block);
      if (!m) continue;
      const name = m[1];
      let data;
      try { data = JSON.parse(m[2]); } catch (_) { continue; }
      handleEvent(name, data);
    }
  }

  status.textContent = 'Completado.';
  goBtn.disabled = false;
});

function handleEvent(name, data) {
  thinkingDiv.textContent += '[' + name + ']\n';

  if (name === 'pre_analysis') {
    const top1 = (data.baseline_top1 || 0).toFixed(3);
    const firstIntent = (data.intent_chain && data.intent_chain[0] && data.intent_chain[0].intent) || '—';
    preDiv.innerHTML = '<div class="kv">' +
      '<div class="k">baseline_top1</div><div class="v"><b>' + top1 + '</b></div>' +
      '<div class="k">first intent</div><div class="v"><b>' + firstIntent + '</b></div>' +
      '</div>';
  } else if (name === 'judge_loading') {
    thinkingMeta.textContent = '⏳ cargando ' + data.model + '…';
  } else if (name === 'thinking_chunk') {
    thinkingDiv.textContent += data.chunk;
    thinkingDiv.scrollTop = thinkingDiv.scrollHeight;
  } else if (name === 'thinking_complete') {
    thinkingMeta.textContent = '✓ thinking completo (' + data.total_chars + ' chars).';
  } else if (name === 'complete') {
    const score = data.authenticity_score;
    let cls = 'low';
    if (score >= 65) cls = 'high';
    else if (score >= 40) cls = 'mid';
    finalDiv.innerHTML = '<div class="score ' + cls + '">' + score + '<span style="font-size:0.4em;color:#6e7681">/100</span></div>';
  } else if (name === 'refine_loading') {
    refinedDiv.innerHTML = '<div class="muted">Refinando pregunta…</div>';
  } else if (name === 'refine_complete') {
    refinedDiv.innerHTML = '<div class="kv">' +
      '<div class="k">original</div><div class="v">' + (data.original_question || '') + '</div>' +
      '<div class="k">refined</div><div class="v"><b>' + (data.refined_question || '') + '</b></div>' +
      '</div>';
  } else if (name === 'error' || name === 'parse_error') {
    finalDiv.innerHTML = '<div class="err">⚠ ' + name + ': ' + (data.message || '') + '</div>';
  }
}
</script>
</body>
</html>`
