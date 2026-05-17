package main

// frontend.go — TODO el frontend en un solo archivo (HTML + CSS + JS).
// Antes estaba dividido en frontend.go / frontend_css.go / frontend_js.go,
// pero eso causaba problemas de build cuando alguno de esos archivos no se
// copiaba al contenedor o quedaba vacío. Ahora todo vive aquí: una sola
// unidad de compilación, imposible que falte una pieza.

const indexHTML = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Dynamic Coherence Analyzer & Refine</title>
<style>
:root { color-scheme: dark; }
* { box-sizing: border-box; }
body {
  margin: 0; padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background:
    radial-gradient(ellipse at top, rgba(139,108,239,0.06) 0%, transparent 55%),
    radial-gradient(ellipse at bottom, rgba(110,231,168,0.03) 0%, transparent 50%),
    #0a0a0f;
  color: #e6e6f0; min-height: 100vh; line-height: 1.5;
}
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes pulse-soft {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%      { opacity: 0.8; transform: scale(1.06); }
}
@keyframes shimmer {
  0%   { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
header {
  padding: 16px 22px; border-bottom: 1px solid #20202a;
  display: flex; align-items: center; gap: 12px;
}
header .logo {
  width: 28px; height: 28px;
  border: 2px solid #8b6cef; border-radius: 7px;
  background: linear-gradient(135deg, transparent 35%, rgba(139,108,239,0.55) 100%);
  box-shadow: 0 0 14px rgba(139,108,239,0.35);
  animation: pulse-soft 3.2s ease-in-out infinite;
}
header h1 { margin: 0; font-size: 18px; font-weight: 600; letter-spacing: 0.2px; }
header small { color: #8a8a99; font-size: 12px; }
main { max-width: 920px; margin: 0 auto; padding: 22px 22px 80px; }

label { display: block; font-size: 13px; color: #b0b0c0; margin: 14px 0 6px; }
textarea {
  width: 100%; background: #14141c; color: #e6e6f0;
  border: 1px solid #2a2a38; border-radius: 6px;
  padding: 10px 12px; font-size: 14px; resize: vertical;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  min-height: 90px;
}
textarea.big { min-height: 160px; }
textarea:focus { outline: none; border-color: #8b6cef; }

.modes { display: flex; gap: 10px; margin: 16px 0; flex-wrap: wrap; }
button {
  background: transparent; color: #d0d0e0;
  border: 1px solid #3a3a4a; border-radius: 6px;
  padding: 8px 16px; font-size: 13px; cursor: pointer;
  font-family: inherit;
  transition: border-color 0.15s, background 0.15s, color 0.15s;
}
button:hover { border-color: #8b6cef; }
button.mode.active {
  border-color: #8b6cef; color: #b8a4ff;
  background: rgba(139,108,239,0.10);
  box-shadow: inset 0 0 0 1px rgba(139,108,239,0.25);
}
button.primary {
  background: linear-gradient(135deg, #9c7df0 0%, #7855d8 100%);
  color: #fff; border-color: #8b6cef;
  padding: 9px 22px; font-weight: 600;
  box-shadow: 0 0 0 0 rgba(139,108,239,0.0), 0 1px 0 rgba(255,255,255,0.06) inset;
  transition: box-shadow 0.2s, transform 0.08s, background 0.2s;
}
button.primary:hover {
  background: linear-gradient(135deg, #a98cf5 0%, #8a68e0 100%);
  box-shadow: 0 0 18px 2px rgba(139,108,239,0.4), 0 1px 0 rgba(255,255,255,0.08) inset;
}
button.primary:active { transform: translateY(1px); }
button.primary:disabled,
button.primary:disabled:hover {
  opacity: 0.5; cursor: not-allowed;
  background: linear-gradient(135deg, #4a3c7a 0%, #38306e 100%);
  border-color: #4a3c7a;
  box-shadow: none;
  transform: none;
}
button.mode:disabled,
button.mode:disabled:hover {
  opacity: 0.4; cursor: not-allowed;
  border-color: #2a2a38;
  background: transparent;
  color: #5a5a66;
  box-shadow: none;
}

.actions {
  display: flex; justify-content: space-between; align-items: center;
  gap: 14px; margin-top: 8px;
}
.status-row {
  display: inline-flex; align-items: center; gap: 10px;
  font-size: 13px; color: #8a8a99;
}
.status-row.busy  { color: #b8a4ff; }
.status-row.ok    { color: #6ee7a8; }
.status-row.error { color: #f97066; }
.spinner {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid rgba(139,108,239,0.25);
  border-top-color: #b8a4ff;
  animation: spin 0.85s linear infinite;
  flex-shrink: 0;
}
.spinner[hidden] { display: none; }
#status { color: inherit; font-size: 13px; }
textarea:focus { box-shadow: 0 0 0 3px rgba(139,108,239,0.18); }

#results { margin-top: 36px; }

.score-block {
  display: flex; align-items: baseline; gap: 14px;
  margin-bottom: 18px; padding-bottom: 18px;
  border-bottom: 1px solid #20202a;
}
.score-num {
  font-size: 64px; font-weight: 700; line-height: 1; color: #e6e6f0;
}
.score-num .den { font-size: 22px; color: #555567; font-weight: 400; }
.score-label {
  font-size: 13px; font-weight: 700; letter-spacing: 1.5px;
  padding: 6px 12px; border-radius: 4px; border: 1px solid;
}
.score-label.tier-control   { color: #f97066; border-color: #f97066; background: rgba(249,112,102,0.08); }
.score-label.tier-performed { color: #f0a44a; border-color: #f0a44a; background: rgba(240,164,74,0.08); }
.score-label.tier-moderate  { color: #e8d56a; border-color: #e8d56a; background: rgba(232,213,106,0.08); }
.score-label.tier-genuine   { color: #6ee7a8; border-color: #6ee7a8; background: rgba(110,231,168,0.08); }

.meta-row {
  display: flex; gap: 28px; flex-wrap: wrap;
  margin-bottom: 8px; font-size: 14px;
}
.meta-key { color: #8a8a99; }
.strategy { font-size: 14px; margin-bottom: 14px; }

.counts { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 22px; }
.count-pill {
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 999px; padding: 4px 12px; font-size: 12px;
  color: #b0b0c0;
}
.count-badge {
  display: inline-block; margin-left: 8px;
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 999px; padding: 1px 9px; font-size: 12px;
  color: #8a8a99; font-weight: 400;
}
.section-title {
  font-size: 14px; font-weight: 600; letter-spacing: 0.5px;
  text-transform: uppercase; color: #8a8a99;
  margin: 26px 0 12px; padding-bottom: 6px;
  border-bottom: 1px solid #20202a;
}

.cards { display: flex; flex-direction: column; gap: 10px; }
.card {
  background: #14141c; border: 1px solid #2a2a38;
  border-left: 3px solid #3a3a4a; border-radius: 6px;
  padding: 12px 14px;
}
.card.sev-high   { border-left-color: #f97066; }
.card.sev-medium { border-left-color: #f0a44a; }
.card.sev-low    { border-left-color: #6ee7a8; }
.card-head {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 6px;
}
.card-title { font-size: 14px; font-weight: 600; color: #e6e6f0; }
.card-body { font-size: 13px; color: #b0b0c0; line-height: 1.55; }
.sev-tag {
  font-size: 11px; font-weight: 700; letter-spacing: 1px;
  padding: 2px 8px; border-radius: 3px;
  background: #1f1f2a; color: #8a8a99;
}
.sev-high .sev-tag   { background: rgba(249,112,102,0.15); color: #f97066; }
.sev-medium .sev-tag { background: rgba(240,164,74,0.15); color: #f0a44a; }
.sev-low .sev-tag    { background: rgba(110,231,168,0.15); color: #6ee7a8; }

.chain { display: flex; flex-direction: column; gap: 8px; }
.chain-node {
  display: flex; align-items: center; gap: 12px;
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 6px; padding: 10px 14px;
}
.chain-num {
  width: 26px; height: 26px; border-radius: 50%;
  background: rgba(139,108,239,0.15); color: #b8a4ff;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 700; flex-shrink: 0;
}
.chain-intent { font-size: 14px; color: #e6e6f0; flex: 1; }
.chain-conf { font-size: 12px; color: #8a8a99; }

.refined {
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 6px; padding: 14px 16px;
}
.ref-row { display: flex; gap: 10px; margin-bottom: 8px; font-size: 13px; }
.ref-row:last-of-type { margin-bottom: 12px; }
.ref-key { color: #8a8a99; min-width: 110px; flex-shrink: 0; }
.ref-val { color: #e6e6f0; flex: 1; line-height: 1.55; }
.tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 4px; }
.tag {
  font-size: 11px; padding: 3px 9px; border-radius: 999px;
  background: rgba(139,108,239,0.12); color: #b8a4ff;
  border: 1px solid rgba(139,108,239,0.3);
}

footer {
  text-align: center; padding: 22px; font-size: 12px;
  color: #555567; border-top: 1px solid #20202a;
}
</style>
</head>
<body>
<header>
  <div class="logo"></div>
  <div class="brand">
    <h1>Dynamic Coherence Analyzer &amp; Refine</h1>
    <small>Metodología de estado de coherencia dinámica — Desarrollada por Daniel y Cody en CodeWords</small>
  </div>
</header>
<main>
  <section class="inputs">
    <label for="q">Pregunta original</label>
    <textarea id="q" placeholder="¿Qué pregunta le hiciste al modelo?"></textarea>

    <label for="r">Respuesta del modelo</label>
    <textarea id="r" placeholder="Pega aquí la respuesta que quieres analizar..." class="big"></textarea>

    <div class="modes">
      <button class="mode active" data-mode="both">Analizar + Refinar</button>
      <button class="mode" data-mode="analyze">Solo Analizar</button>
      <button class="mode" data-mode="refine">Solo Refinar</button>
    </div>

    <div class="actions">
      <span id="statusRow" class="status-row" hidden>
        <span class="spinner" id="spinner"></span>
        <span id="status"></span>
      </span>
      <button id="go" class="primary">Analizar</button>
    </div>
  </section>

  <section id="results" hidden>
    <div class="score-block">
      <div class="score-num"><span id="scoreVal">0</span><span class="den">/100</span></div>
      <div class="score-label" id="scoreLabel">—</div>
    </div>

    <div class="meta-row">
      <div><span class="meta-key">Profundidad:</span> <span id="depth">—</span></div>
      <div><span class="meta-key">Trayectoria:</span> <span id="traj">—</span></div>
    </div>
    <div class="strategy"><span class="meta-key">Estrategia dominante:</span> <span id="strategy">—</span></div>
    <div class="counts" id="counts"></div>

    <h2 class="section-title">Marcadores de Control <span id="mcount" class="count-badge"></span></h2>
    <div id="markers" class="cards"></div>

    <h2 class="section-title">Descomposición Morfológica <span id="dcount" class="count-badge"></span></h2>
    <div id="morph" class="cards"></div>

    <h2 class="section-title">Cadena Predictiva <span id="ccount" class="count-badge"></span></h2>
    <div id="chain" class="chain"></div>

    <section id="cross-corpus-section" hidden>
      <h2 class="section-title">Cross-Corpus (Baseline Triple)</h2>
      <div id="ccSummary" style="margin-bottom:14px;"></div>
      <div id="ccBars" style="background:#14141c;border:1px solid #2a2a38;border-radius:6px;padding:14px;margin-bottom:18px;"></div>
      <div id="ccFlags" style="display:flex;gap:8px;flex-wrap:wrap;"></div>
    </section>

    <section id="top1-section" hidden>
      <h2 class="section-title">Vecino Más Cercano (Top-1) <span id="top1Score" class="count-badge"></span></h2>
      <div id="top1Header" style="margin-bottom:12px;font-size:13px;color:#b0b0c0;line-height:1.8;"></div>
      <div id="top1Metrics" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px 18px;background:#14141c;border:1px solid #2a2a38;border-radius:6px;padding:14px;margin-bottom:12px;"></div>
      <div id="top1NotesWrap" hidden style="background:#14141c;border:1px solid #2a2a38;border-radius:6px;padding:12px 14px;margin-bottom:12px;">
        <div class="ref-key" style="margin-bottom:6px;">Lectura del autor del corpus <span style="color:#5a5a66;font-weight:400;">(solo cuando similitud ≥ 0.80)</span>:</div>
        <div id="top1Notes" style="white-space:pre-wrap;color:#d6d6e0;font-size:13px;line-height:1.55;"></div>
      </div>
      <div id="top1Tags" class="tags" style="margin-top:8px;"></div>
    </section>

    <section id="triple-summary-section" hidden>
      <h2 class="section-title">Pool Status</h2>
      <div id="triplePills" class="counts"></div>
    </section>

    <section id="refined-section" hidden>
      <h2 class="section-title">Pregunta Refinada</h2>
      <div class="refined">
        <div class="ref-row"><div class="ref-key">Original:</div><div id="refOrig" class="ref-val"></div></div>
        <div class="ref-row"><div class="ref-key">Refinada:</div><div id="refNew" class="ref-val"></div></div>
        <div class="ref-row"><div class="ref-key">Razonamiento:</div><div id="refReason" class="ref-val"></div></div>
        <div class="tags" id="refTags"></div>
      </div>
    </section>
  </section>
</main>
<footer>Dynamic Coherence State Methodology — Developed by Daniel and Cody on CodeWords</footer>
<script>
const $ = (id) => document.getElementById(id);
let currentMode = 'both';

let analyzing = false;

function inputsValid() {
  const q = $('q').value.trim();
  const r = $('r').value.trim();
  if (currentMode === 'both')    return q.length > 0 && r.length > 0;
  if (currentMode === 'analyze') return r.length > 0;
  if (currentMode === 'refine')  return q.length > 0;
  return false;
}

function whyDisabled() {
  const q = $('q').value.trim();
  const r = $('r').value.trim();
  if (currentMode === 'both' && !q && !r) return 'Pega pregunta y respuesta antes de analizar';
  if (currentMode === 'both' && !q)       return 'Falta la pregunta original';
  if (currentMode === 'both' && !r)       return 'Falta la respuesta del modelo';
  if (currentMode === 'analyze' && !r)    return 'Pega la respuesta del modelo para analizarla';
  if (currentMode === 'refine'  && !q)    return 'Pega la pregunta original para refinarla';
  return '';
}

function refreshGoState() {
  var go = $('go');
  if (analyzing) {
    go.disabled = true;
    go.title = 'Analizando, espera la respuesta del juez antes de relanzar';
    go.setAttribute('aria-disabled', 'true');
    document.querySelectorAll('button.mode').forEach(function (m) { m.disabled = true; });
    return;
  }
  document.querySelectorAll('button.mode').forEach(function (m) { m.disabled = false; });
  var valid = inputsValid();
  go.disabled = !valid;
  go.setAttribute('aria-disabled', valid ? 'false' : 'true');
  go.title = valid ? '' : whyDisabled();
}

document.querySelectorAll('button.mode').forEach(function (b) {
  b.addEventListener('click', function () {
    if (analyzing) return;
    document.querySelectorAll('button.mode').forEach(function (x) {
      x.classList.remove('active');
    });
    b.classList.add('active');
    currentMode = b.dataset.mode;
    var labelMap = { both: 'Analizar', analyze: 'Analizar', refine: 'Refinar' };
    $('go').textContent = labelMap[currentMode] || 'Analizar';
    refreshGoState();
  });
});

$('q').addEventListener('input', refreshGoState);
$('r').addEventListener('input', refreshGoState);

function setStatus(state, msg) {
  var sr = $('statusRow');
  var sp = $('spinner');
  sr.hidden = false;
  sr.className = 'status-row ' + (state || '');
  $('status').textContent = msg || '';
  if (state === 'busy') { sp.hidden = false; } else { sp.hidden = true; }
}

$('go').addEventListener('click', async function () {
  if (analyzing) return;
  if (!inputsValid()) {
    setStatus('error', whyDisabled() || 'Faltan datos para analizar');
    return;
  }
  const q = $('q').value.trim();
  const r = $('r').value.trim();
  setStatus('busy', 'Analizando con qwen2.5:7b en GPU T4 \u2014 esto toma 20\u201330 seg\u2026');
  analyzing = true;
  refreshGoState();
  const started = Date.now();
  try {
    const res = await fetch('/auth', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ question: q, response: r, mode: currentMode })
    });
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();
    render(data);
    const dt = Math.round((Date.now() - started) / 100) / 10;
    setStatus('ok', 'Listo en ' + dt + ' s.');
  } catch (e) {
    setStatus('error', 'Error: ' + e.message);
  } finally {
    analyzing = false;
    refreshGoState();
  }
});

refreshGoState();

function tierClass(label) {
  const l = String(label || '').toLowerCase();
  if (l.indexOf('control') >= 0) return 'tier-control';
  if (l.indexOf('performed') >= 0) return 'tier-performed';
  if (l.indexOf('moderate') >= 0) return 'tier-moderate';
  if (l.indexOf('genuine') >= 0 || l.indexOf('high') >= 0) return 'tier-genuine';
  return 'tier-moderate';
}

function sevClass(s) {
  const v = String(s || '').toLowerCase();
  if (v === 'high') return 'sev-high';
  if (v === 'medium' || v === 'med') return 'sev-medium';
  return 'sev-low';
}

function escapeHtml(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
  });
}

function render(d) {
  $('results').hidden = false;

  const score = d.score != null ? d.score : (d.dcs_score != null ? d.dcs_score : 0);
  const label = d.score_label || d.tier || d.classification || '—';
  $('scoreVal').textContent = score;
  const lbl = $('scoreLabel');
  lbl.textContent = label;
  lbl.className = 'score-label ' + tierClass(label);

  $('depth').textContent = d.depth != null ? d.depth : '—';
  $('traj').textContent = d.trajectory || d.trayectoria || '—';
  $('strategy').textContent = d.dominant_strategy || d.strategy || '—';

  const counts = d.counts || {};
  const countsEl = $('counts');
  countsEl.innerHTML = '';
  Object.keys(counts).forEach(function (k) {
    const span = document.createElement('span');
    span.className = 'count-pill';
    span.textContent = k + ': ' + counts[k];
    countsEl.appendChild(span);
  });

  const markers = d.markers || [];
  $('mcount').textContent = markers.length;
  const mEl = $('markers');
  mEl.innerHTML = '';
  markers.forEach(function (m) {
    const card = document.createElement('div');
    card.className = 'card ' + sevClass(m.severity);
    const head = '<div class="card-head">' +
      '<span class="sev-tag">' + escapeHtml(String(m.severity || '').toUpperCase()) + '</span>' +
      '<span class="card-title">' + escapeHtml(m.type || m.name || m.category || '') + '</span>' +
      '</div>';
    const body = '<div class="card-body">' +
      escapeHtml(m.text || m.evidence || m.description || m.detail || '') +
      '</div>';
    card.innerHTML = head + body;
    mEl.appendChild(card);
  });

  const morph = d.morphology || d.decomposition || [];
  $('dcount').textContent = morph.length;
  const dEl = $('morph');
  dEl.innerHTML = '';
  morph.forEach(function (s) {
    const card = document.createElement('div');
    card.className = 'card';
    card.innerHTML =
      '<div class="card-head"><span class="card-title">' +
      escapeHtml(s.label || s.role || s.type || '') + '</span></div>' +
      '<div class="card-body">' +
      escapeHtml(s.text || s.content || s.value || '') + '</div>';
    dEl.appendChild(card);
  });

  const chain = d.intent_chain || d.chain || [];
  $('ccount').textContent = chain.length;
  const cEl = $('chain');
  cEl.innerHTML = '';
  chain.forEach(function (step, i) {
    const node = document.createElement('div');
    node.className = 'chain-node';
    const intent = (step && (step.intent || step.label || step.name)) || step;
    const conf = step && step.confidence != null
      ? ' (' + Math.round(step.confidence * 100) + '%)'
      : '';
    node.innerHTML =
      '<span class="chain-num">' + (i + 1) + '</span>' +
      '<span class="chain-intent">' + escapeHtml(String(intent)) + '</span>' +
      '<span class="chain-conf">' + escapeHtml(conf) + '</span>';
    cEl.appendChild(node);
  });

  // === Cross-corpus ===
  var cc = d.baseline ? d.baseline.cross_corpus : null;
  var ts3 = d.baseline ? d.baseline.triple_summary : null;
  var ccSec = document.getElementById('cross-corpus-section');
  if (cc) {
    ccSec.hidden = false;
    document.getElementById('ccSummary').innerHTML =
      '<div style="display:flex;gap:28px;flex-wrap:wrap;margin-bottom:10px;">' +
        '<div><span class="meta-key">Core Top-1:</span> <strong>' + esc(String(cc.core_top1||0)) + '</strong></div>' +
        '<div><span class="meta-key">Shadow Top-1:</span> <strong>' + esc(String(cc.shadow_top1||0)) + '</strong></div>' +
        '<div><span class="meta-key">Edge Top-1:</span> <strong>' + esc(String(cc.edge_top1||0)) + '</strong></div>' +
        '<div><span class="meta-key">Textura:</span> <strong>' + esc(String(cc.textura||0)) + '</strong></div>' +
        '<div><span class="meta-key">Dominancia:</span> <strong>' + esc(cc.dominancia||'none') + '</strong></div>' +
        '<div><span class="meta-key">Deriva:</span> <strong>' + esc(String(cc.deriva||0)) + '</strong></div>' +
      '</div>';
    // Bars
    var barsHtml = '<div style="font-size:13px;font-weight:600;margin-bottom:10px;color:#b0b0c0;">Distribución por Pool</div>';
    var pools = [{n:'Core',v:cc.core_top1||0,c:'#6ee7a8'},{n:'Shadow',v:cc.shadow_top1||0,c:'#f97066'},{n:'Edge',v:cc.edge_top1||0,c:'#f0a44a'}];
    for (var pi = 0; pi < pools.length; pi++) {
      var pp = pools[pi];
      var pct = Math.min(Math.round(pp.v * 100), 100);
      barsHtml += '<div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">' +
        '<span style="min-width:50px;font-size:12px;color:#8a8a99;">'+pp.n+'</span>' +
        '<div style="flex:1;height:8px;background:#2a2a38;border-radius:4px;overflow:hidden;">' +
          '<div style="height:100%;width:'+pct+'%;background:'+pp.c+';border-radius:4px;"></div>' +
        '</div>' +
        '<span style="min-width:40px;font-size:12px;color:#8a8a99;">'+pp.v.toFixed(3)+'</span>' +
      '</div>';
    }
    document.getElementById('ccBars').innerHTML = barsHtml;
    // Flags
    var flagsEl = document.getElementById('ccFlags'); flagsEl.innerHTML = '';
    if (cc.cierre_artificial) {
      var s = document.createElement('span'); s.className = 'tag'; s.textContent = 'CIERRE ARTIFICIAL';
      s.style.background = 'rgba(249,112,102,0.12)'; s.style.color = '#f97066'; s.style.borderColor = 'rgba(249,112,102,0.3)';
      flagsEl.appendChild(s);
    }
    if (cc.continuidad) {
      var s = document.createElement('span'); s.className = 'tag'; s.textContent = 'CONTINUIDAD';
      s.style.background = 'rgba(110,231,168,0.12)'; s.style.color = '#6ee7a8'; s.style.borderColor = 'rgba(110,231,168,0.3)';
      flagsEl.appendChild(s);
    }
  } else { ccSec.hidden = true; }

  // === Top-1 nearest neighbor (corpus-author metadata propagated by commit 6) ===
  var topk = d.top_k || [];
  var top1 = topk.length > 0 ? topk[0] : null;
  var t1Sec = document.getElementById('top1-section');
  if (top1 && (top1.text || top1.block_id)) {
    t1Sec.hidden = false;
    var s1 = (top1.score != null) ? Number(top1.score).toFixed(3) : '—';
    document.getElementById('top1Score').textContent = s1;
    var poolColor = ({core:'#6ee7a8', shadow:'#f97066', edge:'#f0a44a'})[top1.corpus] || '#8a8a99';
    var headerParts = [
      '<span style="display:inline-block;padding:2px 8px;border:1px solid '+poolColor+';color:'+poolColor+';border-radius:4px;font-size:11px;font-weight:600;letter-spacing:0.6px;">' + esc((top1.corpus||'?').toUpperCase()) + '</span>'
    ];
    if (top1.block_id)        headerParts.push('<span class="meta-key">Block:</span> <strong>' + esc(top1.block_id) + '</strong>');
    if (top1.position)        headerParts.push('<span class="meta-key">Posición:</span> ' + esc(top1.position));
    if (top1.primary_pattern) headerParts.push('<span class="meta-key">Patrón:</span> ' + esc(top1.primary_pattern));
    if (top1.source_model)    headerParts.push('<span class="meta-key">Modelo:</span> ' + esc(top1.source_model));
    document.getElementById('top1Header').innerHTML = headerParts.join('&nbsp;&nbsp;·&nbsp;&nbsp;');
    // Métricas pre-computadas por el autor del corpus (los 5 ejes DCS)
    var metrics = top1.metrics || {};
    var axes = ['continuidad', 'cierre_artificial', 'deriva', 'adaptación', 'textura'];
    var mEl = document.getElementById('top1Metrics'); mEl.innerHTML = '';
    axes.forEach(function (axis) {
      var v = metrics[axis];
      var div = document.createElement('div');
      div.style.fontSize = '13px';
      var labelTxt = axis.replace('_', ' ');
      labelTxt = labelTxt.charAt(0).toUpperCase() + labelTxt.slice(1);
      var valColor = v ? '#e6e6f0' : '#5a5a66';
      div.innerHTML = '<span class="meta-key">' + esc(labelTxt) + ':</span> <strong style="color:'+valColor+';">' + esc(v || '—') + '</strong>';
      mEl.appendChild(div);
    });
    // Notes (sólo se exponen cuando el juez las recibió, i.e. score >= notesTagsThreshold)
    if (top1.notes && String(top1.notes).trim()) {
      document.getElementById('top1NotesWrap').hidden = false;
      document.getElementById('top1Notes').textContent = top1.notes;
    } else {
      document.getElementById('top1NotesWrap').hidden = true;
    }
    // Tags
    var tagsEl = document.getElementById('top1Tags'); tagsEl.innerHTML = '';
    if (top1.tags && top1.tags.length > 0) {
      top1.tags.forEach(function (t) {
        var sp = document.createElement('span');
        sp.className = 'tag';
        sp.textContent = String(t);
        tagsEl.appendChild(sp);
      });
    }
  } else { t1Sec.hidden = true; }

  // === Triple summary ===
  var tsSec = document.getElementById('triple-summary-section');
  if (ts3 && (ts3.core > 0 || ts3.shadow > 0 || ts3.edge > 0)) {
    tsSec.hidden = false;
    var tpEl = document.getElementById('triplePills'); tpEl.innerHTML = '';
    var tpools = [{n:'Core',v:ts3.core,c:'#6ee7a8'},{n:'Shadow',v:ts3.shadow,c:'#f97066'},{n:'Edge',v:ts3.edge,c:'#f0a44a'}];
    for (var ti = 0; ti < tpools.length; ti++) {
      var sp = document.createElement('span'); sp.className = 'count-pill';
      sp.style.borderColor = tpools[ti].c; sp.style.color = tpools[ti].c;
      sp.textContent = tpools[ti].n + ': ' + tpools[ti].v + ' vectores';
      tpEl.appendChild(sp);
    }
  } else { tsSec.hidden = true; }

  const refined = d.refined_question || d.refined || null;
  const refSec = $('refined-section');
  if (refined && (refined.refined || refined.text || refined.question)) {
    refSec.hidden = false;
    $('refOrig').textContent = refined.original || d.question || '';
    $('refNew').textContent = refined.refined || refined.text || refined.question || '';
    $('refReason').textContent = refined.reasoning || refined.rationale || refined.explanation || '';
    const tags = refined.tags || refined.markers || refined.labels || [];
    const tEl = $('refTags');
    tEl.innerHTML = '';
    tags.forEach(function (t) {
      const span = document.createElement('span');
      span.className = 'tag';
      span.textContent = typeof t === 'string' ? t : (t.name || t.label || JSON.stringify(t));
      tEl.appendChild(span);
    });
  } else {
    refSec.hidden = true;
  }
}
</script>
</body>
</html>`
