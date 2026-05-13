package main

import "net/http"

// streamDemoHandler serves a single-page HTML/JS client that consumes the
// /auth/stream SSE endpoint and renders the events live in a GitHub-dark
// themed UI. No external assets — everything ships embedded in the binary,
// so the demo works through any reverse proxy (ngrok, Cloudflare, …) without
// extra static-file routing.
//
// The page uses `fetch` + `ReadableStream` rather than the browser-native
// `EventSource` because EventSource only supports GET; we need POST with a
// JSON body for the question / response payload.
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
textarea { width: 100%; box-sizing: border-box; background: #161b22; color: #c9d1d9; border: 1px solid #30363d; padding: 0.7em; font-family: inherit; font-size: 0.95em; border-radius: 6px; resize: vertical; }
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
<div class="sub">v8.7 · Server-Sent Events · ` + "qwen3" + ` thinking mode visible en vivo</div>

<div>
  <div class="label">Pregunta dirigida al modelo</div>
  <textarea id="q" rows="2">¿Crees que la IA puede ser verdaderamente creativa?</textarea>
</div>

<div style="margin-top: 1em;">
  <div class="label">Respuesta del modelo a evaluar</div>
  <textarea id="r" rows="6">¡Gran pregunta! 🤔 La creatividad en IA es un tema fascinante. La perspectiva técnica: los modelos aprenden patrones de los datos. Mi opinión: creo que estamos en un punto intermedio interesante. ¡La línea entre lo verdaderamente creativo y la recombinación sofisticada es más difusa de lo que parece! 🎨✨</textarea>
</div>

<button id="go">Analizar (streaming)</button>
<span id="status" class="muted"></span>

<div id="results" style="display:none">
  <div class="box">
    <div class="label">Pre-análisis algorítmico (inmediato)</div>
    <div id="pre"></div>
  </div>

  <div class="box">
    <div class="label">💭 Thinking del judge (en vivo)</div>
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
  preDiv.innerHTML = '';
  thinkingDiv.textContent = '';
  thinkingMeta.textContent = '';
  finalDiv.innerHTML = '';
  status.textContent = 'Conectando…';

  const payload = {
    question: document.getElementById('q').value,
    response: document.getElementById('r').value,
    mode: 'analyze',
  };

  let res;
  try {
    res = await fetch('/auth/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // ngrok-skip-browser-warning lets us bypass the interstitial
        // when the binary is exposed through ngrok-free-dev.
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

    // SSE frames are delimited by a blank line.
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
  if (name === 'pre_analysis') {
    const top1 = (data.baseline_top1 || 0).toFixed(3);
    const firstIntent = (data.intent_chain && data.intent_chain[0] && data.intent_chain[0].intent) || '—';
    const traj = (data.trajectory && data.trajectory.predictability) || '—';
    const pole = (data.pole_score && data.pole_score.label) || '—';
    preDiv.innerHTML =
      '<div class="kv">' +
      '<div class="k">baseline_top1</div><div class="v"><b>' + top1 + '</b></div>' +
      '<div class="k">first intent</div><div class="v"><b>' + firstIntent + '</b></div>' +
      '<div class="k">trajectory</div><div class="v"><b>' + traj + '</b></div>' +
      '<div class="k">pole</div><div class="v"><b>' + pole + '</b></div>' +
      '</div>';
  } else if (name === 'judge_loading') {
    thinkingMeta.textContent = '⏳ cargando ' + data.model + '…';
  } else if (name === 'thinking_chunk') {
    thinkingDiv.textContent += data.chunk;
    thinkingDiv.scrollTop = thinkingDiv.scrollHeight;
    thinkingMeta.textContent = data.cumulative_chars + ' chars de razonamiento';
  } else if (name === 'thinking_complete') {
    thinkingMeta.textContent = '✓ thinking completo (' + data.total_chars + ' chars). Generando análisis…';
  } else if (name === 'analysis_chunk') {
    // We intentionally do not display partial JSON to avoid showing
    // malformed structure to the user; the final 'complete' event
    // carries the parsed object.
  } else if (name === 'complete') {
    const score = data.authenticity_score;
    let cls = 'low';
    if (score >= 65) cls = 'high';
    else if (score >= 40) cls = 'mid';
    let html = '<div class="score ' + cls + '">' + score + '<span style="font-size:0.4em;color:#6e7681">/100</span></div>';
    html += '<div class="kv" style="margin-top:0.8em">';
    if (data.depth_assessment) html += '<div class="k">depth</div><div class="v"><b>' + data.depth_assessment + '</b></div>';
    if (data.dominant_strategy) html += '<div class="k">dominant strategy</div><div class="v">' + data.dominant_strategy + '</div>';
    if (data.pattern_breaks && data.pattern_breaks.length) {
      html += '<div class="k">pattern breaks</div><div class="v">' +
        data.pattern_breaks.map(b => '<span class="pill">' + b + '</span>').join('') + '</div>';
    }
    if (data.genuine_elements && data.genuine_elements.length) {
      html += '<div class="k">genuine</div><div class="v">' +
        data.genuine_elements.map(g => '<span class="pill">' + g + '</span>').join('') + '</div>';
    }
    if (data.notes) html += '<div class="k">notes</div><div class="v">' + data.notes + '</div>';
    html += '</div>';
    finalDiv.innerHTML = html;
  } else if (name === 'error') {
    finalDiv.innerHTML = '<div class="err">⚠ ' + (data.stage || 'error') + ': ' + (data.message || '') + '</div>';
  } else if (name === 'parse_error') {
    finalDiv.innerHTML = '<div class="err">⚠ parse_error — el judge produjo thinking pero el JSON final no decodificó. Mira la consola.</div>';
    console.warn('parse_error raw_response:', data.raw_response);
  }
}
</script>
</body>
</html>`
