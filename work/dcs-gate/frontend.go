package main

// frontend.go — UI completa: HTML + CSS + JS en un solo archivo.
// v8.8: todas las rutas usan /auth/stream (SSE).
// Muestra: thinking del juez en tiempo real + refiner recursivo (candidatos, selección, ganador).

const indexHTML = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>DCS-Gate v8.8</title>
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
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0} }
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
header .vtag {
  margin-left: auto; font-size: 11px; font-weight: 700; letter-spacing: 1px;
  padding: 3px 9px; border-radius: 4px;
  background: rgba(139,108,239,0.12); color: #b8a4ff;
  border: 1px solid rgba(139,108,239,0.25);
}
main { max-width: 960px; margin: 0 auto; padding: 22px 22px 80px; }

label { display: block; font-size: 13px; color: #b0b0c0; margin: 14px 0 6px; }
textarea {
  width: 100%; background: #14141c; color: #e6e6f0;
  border: 1px solid #2a2a38; border-radius: 6px;
  padding: 10px 12px; font-size: 14px; resize: vertical;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  min-height: 90px;
}
textarea.big { min-height: 160px; }
textarea:focus { outline: none; border-color: #8b6cef; box-shadow: 0 0 0 3px rgba(139,108,239,0.18); }

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
button.primary:disabled, button.primary:disabled:hover {
  opacity: 0.5; cursor: not-allowed;
  background: linear-gradient(135deg, #4a3c7a 0%, #38306e 100%);
  border-color: #4a3c7a; box-shadow: none; transform: none;
}
button.mode:disabled, button.mode:disabled:hover {
  opacity: 0.4; cursor: not-allowed; border-color: #2a2a38;
  background: transparent; color: #5a5a66; box-shadow: none;
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
  border: 2px solid rgba(139,108,239,0.25); border-top-color: #b8a4ff;
  animation: spin 0.85s linear infinite; flex-shrink: 0;
}

#results { margin-top: 36px; }
.section-title {
  font-size: 14px; font-weight: 600; letter-spacing: 0.5px;
  text-transform: uppercase; color: #8a8a99;
  margin: 26px 0 12px; padding-bottom: 6px; border-bottom: 1px solid #20202a;
}

/* ── Score block ─────────────────────────── */
.score-block {
  display: flex; align-items: baseline; gap: 14px;
  margin-bottom: 18px; padding-bottom: 18px; border-bottom: 1px solid #20202a;
}
.score-num { font-size: 64px; font-weight: 700; line-height: 1; color: #e6e6f0; }
.score-num .den { font-size: 22px; color: #555567; font-weight: 400; }
.score-label {
  font-size: 13px; font-weight: 700; letter-spacing: 1.5px;
  padding: 6px 12px; border-radius: 4px; border: 1px solid;
}
.score-label.tier-control   { color: #f97066; border-color: #f97066; background: rgba(249,112,102,0.08); }
.score-label.tier-performed { color: #f0a44a; border-color: #f0a44a; background: rgba(240,164,74,0.08); }
.score-label.tier-moderate  { color: #e8d56a; border-color: #e8d56a; background: rgba(232,213,106,0.08); }
.score-label.tier-genuine   { color: #6ee7a8; border-color: #6ee7a8; background: rgba(110,231,168,0.08); }

.meta-row { display: flex; gap: 28px; flex-wrap: wrap; margin-bottom: 8px; font-size: 14px; }
.meta-key { color: #8a8a99; }
.strategy { font-size: 14px; margin-bottom: 14px; }
.counts { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 22px; }
.count-pill {
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 999px; padding: 4px 12px; font-size: 12px; color: #b0b0c0;
}
.count-badge {
  display: inline-block; margin-left: 8px;
  background: #14141c; border: 1px solid #2a2a38;
  border-radius: 999px; padding: 1px 9px; font-size: 12px;
  color: #8a8a99; font-weight: 400;
}

/* ── Cards ───────────────────────────────── */
.cards { display: flex; flex-direction: column; gap: 10px; }
.card {
  background: #14141c; border: 1px solid #2a2a38;
  border-left: 3px solid #3a3a4a; border-radius: 6px; padding: 12px 14px;
}
.card.sev-high   { border-left-color: #f97066; }
.card.sev-medium { border-left-color: #f0a44a; }
.card.sev-low    { border-left-color: #6ee7a8; }
.card-head { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
.card-title { font-size: 14px; font-weight: 600; color: #e6e6f0; }
.card-body { font-size: 13px; color: #b0b0c0; line-height: 1.55; }
.sev-tag {
  font-size: 11px; font-weight: 700; letter-spacing: 1px;
  padding: 2px 8px; border-radius: 3px; background: #1f1f2a; color: #8a8a99;
}
.sev-high .sev-tag   { background: rgba(249,112,102,0.15); color: #f97066; }
.sev-medium .sev-tag { background: rgba(240,164,74,0.15); color: #f0a44a; }
.sev-low .sev-tag    { background: rgba(110,231,168,0.15); color: #6ee7a8; }

/* ── Intent chain ────────────────────────── */
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

/* ── Tags ────────────────────────────────── */
.tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 4px; }
.tag {
  font-size: 11px; padding: 3px 9px; border-radius: 999px;
  background: rgba(139,108,239,0.12); color: #b8a4ff;
  border: 1px solid rgba(139,108,239,0.3);
}

/* ── Thinking stream ─────────────────────── */
.thinking-box {
  background: #0d0d18; border: 1px solid #28284a;
  border-left: 3px solid #5a4ab0; border-radius: 6px;
  padding: 12px 14px; margin-bottom: 14px;
}
.thinking-box-title {
  font-size: 11px; font-weight: 700; letter-spacing: 0.08em;
  text-transform: uppercase; color: #5a4ab0; margin-bottom: 8px;
  display: flex; align-items: center; gap: 8px;
}
.thinking-text {
  font-size: 13px; color: #7a7a9a; line-height: 1.65;
  white-space: pre-wrap; word-break: break-word;
  max-height: 220px; overflow-y: auto;
}
.thinking-meta { font-size: 11px; color: #4a4a6a; margin-top: 6px; }
.cursor { animation: blink 0.9s step-end infinite; color: #5a4ab0; }

/* ── Recursive Refiner v8.8 ──────────────── */
.refiner-header {
  font-size: 11px; font-weight: 700; letter-spacing: 0.09em;
  text-transform: uppercase; color: #8a8a99;
  padding: 6px 0 10px; border-bottom: 1px solid #20202a;
  margin-bottom: 14px;
}
.refiner-priming {
  font-size: 12px; color: #4a4a6a;
  padding: 6px 10px; background: #111118;
  border: 1px solid #1e1e2a; border-radius: 4px; margin-bottom: 14px;
}
.pass-block {
  background: #111118; border: 1px solid #2a2a3a;
  border-radius: 8px; padding: 14px 16px; margin-bottom: 14px;
}
.pass-head {
  display: flex; align-items: center; gap: 10px; margin-bottom: 12px;
  font-size: 11px; font-weight: 700; letter-spacing: 0.07em;
  text-transform: uppercase; color: #7a7a99;
}
.pass-num {
  width: 20px; height: 20px; border-radius: 50%; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700;
  background: rgba(139,108,239,0.15); color: #b8a4ff;
}
.pass-st { margin-left: auto; font-weight: 400; opacity: 0.65; }

.cands-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px;
}
@media(max-width:600px) { .cands-grid { grid-template-columns: 1fr; } }
.cand-card {
  background: #16161f; border: 1px solid #2e2e42;
  border-radius: 6px; padding: 12px 13px; position: relative;
  transition: border-color 0.2s;
}
.cand-card.winner { border-color: #6ee7a8; }
.cand-card.loser  { border-color: #2a2a3a; opacity: 0.45; }
.cand-letter {
  font-size: 10px; font-weight: 700; letter-spacing: 0.08em;
  text-transform: uppercase; color: #5a5a78; margin-bottom: 5px;
}
.cand-card.winner .cand-letter { color: #6ee7a8; }
.cand-strat {
  display: inline-block; font-size: 11px;
  padding: 2px 8px; border-radius: 3px; margin-bottom: 7px;
  background: rgba(139,108,239,0.10); color: #9080cc;
  border: 1px solid rgba(139,108,239,0.18);
}
.cand-q { font-size: 14px; color: #d0d0e0; line-height: 1.6; margin-bottom: 5px; }
.cand-r { font-size: 12px; color: #6a6a88; line-height: 1.5; }
.cand-badge {
  position: absolute; top: 7px; right: 7px;
  font-size: 10px; font-weight: 700; letter-spacing: 0.06em;
  padding: 2px 7px; border-radius: 3px;
}
.badge-w { background: rgba(110,231,168,0.12); color: #6ee7a8; border: 1px solid rgba(110,231,168,0.28); }
.badge-l { background: rgba(249,112,102,0.08); color: #f97066; border: 1px solid rgba(249,112,102,0.18); }

.sel-final {
  background: #0f1a14; border: 1px solid #2a4034;
  border-left: 3px solid #6ee7a8; border-radius: 6px;
  padding: 15px 17px; margin-top: 4px;
}
.sel-label {
  font-size: 10px; font-weight: 700; letter-spacing: 0.08em;
  text-transform: uppercase; color: #4aa07a; margin-bottom: 10px;
}
.sel-q {
  font-size: 16px; font-weight: 500; color: #e6e6f0;
  line-height: 1.65; margin-bottom: 13px;
}
.sel-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
@media(max-width:600px) { .sel-grid { grid-template-columns: 1fr; } }
.sel-rl { font-size: 10px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: #4a7a5a; margin-bottom: 3px; }
.sel-rv { font-size: 13px; color: #a0c0a8; line-height: 1.55; }
.sel-orig { font-size: 12px; color: #4a5a50; margin-top: 11px; }

/* ── Cross-corpus + Top-1 ─────────────────── */
.ref-row { display: flex; gap: 10px; margin-bottom: 8px; font-size: 13px; }
.ref-row:last-of-type { margin-bottom: 12px; }
.ref-key { color: #8a8a99; min-width: 110px; flex-shrink: 0; }
.ref-val { color: #e6e6f0; flex: 1; line-height: 1.55; }
.refined-box { background: #14141c; border: 1px solid #2a2a38; border-radius: 6px; padding: 14px 16px; }

footer { text-align: center; padding: 22px; font-size: 12px; color: #555567; border-top: 1px solid #20202a; }
</style>
</head>
<body>
<header>
  <div class="logo"></div>
  <div class="brand">
    <h1>DCS-Gate — Dynamic Coherence Analyzer &amp; Refine</h1>
    <small>Metodología de estado de coherencia dinámica — Daniel Trejo / Corekeeper Research</small>
  </div>
  <span class="vtag">v8.8</span>
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
      <button id="go" class="primary">Analizar + Refinar</button>
    </div>
  </section>

  <section id="results" hidden>
    <!-- Pre-análisis inmediato -->
    <section id="pre-section" hidden>
      <h2 class="section-title">Pre-Análisis (inmediato)</h2>
      <div id="preBody" style="font-size:13px;color:#b0b0c0;line-height:1.8;"></div>
    </section>

    <!-- Thinking del juez en tiempo real -->
    <section id="thinking-section" hidden>
      <h2 class="section-title">💭 Razonamiento del Juez — en vivo</h2>
      <div class="thinking-box">
        <div class="thinking-text" id="thinkingText"></div>
        <div class="thinking-meta" id="thinkingMeta"></div>
      </div>
    </section>

    <!-- Veredicto final del análisis -->
    <section id="verdict-section" hidden>
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
          <div class="ref-key" style="margin-bottom:6px;">Lectura del autor del corpus:</div>
          <div id="top1Notes" style="white-space:pre-wrap;color:#d6d6e0;font-size:13px;line-height:1.55;"></div>
        </div>
        <div id="top1Tags" class="tags" style="margin-top:8px;"></div>
      </section>

      <section id="triple-summary-section" hidden>
        <h2 class="section-title">Pool Status</h2>
        <div id="triplePills" class="counts"></div>
      </section>
    </section>

    <!-- ── Recursive Refiner v8.8 ─────────────────────────── -->
    <section id="refiner-section" hidden>
      <h2 class="section-title">Recursive Refiner v8.8</h2>

      <div id="refinerPriming" class="refiner-priming" hidden></div>

      <!-- Pass 1: generación de candidatos -->
      <div class="pass-block" id="pass1Block" hidden>
        <div class="pass-head">
          <span class="pass-num">1</span>
          Generación divergente de candidatos
          <span class="pass-st" id="pass1St">pensando…</span>
        </div>
        <div class="thinking-box" id="pass1ThinkBox" hidden>
          <div class="thinking-box-title">Razonamiento generador</div>
          <div class="thinking-text" id="pass1Think"></div>
        </div>
        <div class="cands-grid" id="candsGrid" hidden>
          <div class="cand-card" id="candA">
            <div class="cand-letter">Candidato A</div>
            <div class="cand-strat" id="candAStrat">—</div>
            <div class="cand-q" id="candAQ">—</div>
            <div class="cand-r" id="candAR"></div>
          </div>
          <div class="cand-card" id="candB">
            <div class="cand-letter">Candidato B</div>
            <div class="cand-strat" id="candBStrat">—</div>
            <div class="cand-q" id="candBQ">—</div>
            <div class="cand-r" id="candBR"></div>
          </div>
        </div>
      </div>

      <!-- Pass 2: selección recursiva bajo DCS -->
      <div class="pass-block" id="pass2Block" hidden>
        <div class="pass-head">
          <span class="pass-num">2</span>
          Selección recursiva bajo DCS
          <span class="pass-st" id="pass2St">pensando…</span>
        </div>
        <div class="thinking-box" id="pass2ThinkBox" hidden>
          <div class="thinking-box-title">Razonamiento selector</div>
          <div class="thinking-text" id="pass2Think"></div>
        </div>
      </div>

      <!-- Resultado final -->
      <div class="sel-final" id="selFinal" hidden>
        <div class="sel-label">✓ Pregunta refinada — Recursive Refiner v8.8</div>
        <div class="sel-q" id="selQ"></div>
        <div class="sel-grid">
          <div>
            <div class="sel-rl">Por qué ganó</div>
            <div class="sel-rv" id="selReason"></div>
          </div>
          <div>
            <div class="sel-rl">Por qué perdió la otra</div>
            <div class="sel-rv" id="selLoser"></div>
          </div>
        </div>
        <div class="sel-orig" id="selOrig"></div>
      </div>
    </section>
    <!-- ── end Recursive Refiner ──────────────────────────── -->

    <!-- Fallback: refined simple (para respuestas no-stream o cold) -->
    <section id="refined-section" hidden>
      <h2 class="section-title">Pregunta Refinada</h2>
      <div class="refined-box">
        <div class="ref-row"><div class="ref-key">Original:</div><div id="refOrig" class="ref-val"></div></div>
        <div class="ref-row"><div class="ref-key">Refinada:</div><div id="refNew" class="ref-val"></div></div>
        <div class="ref-row"><div class="ref-key">Razonamiento:</div><div id="refReason" class="ref-val"></div></div>
        <div class="tags" id="refTags"></div>
      </div>
    </section>
  </section>
</main>
<footer>DCS-Gate v8.8 · Dynamic Coherence State Methodology · Daniel Trejo / Corekeeper Research</footer>

<script>
// ─────────────────────────────────────────────────────────────────────────────
// DCS-Gate v8.8 — cliente SSE para /auth/stream.
// Todas las rutas (analyze / refine / both) usan streaming.
// ─────────────────────────────────────────────────────────────────────────────
const $ = id => document.getElementById(id);
let currentMode = 'both';
let analyzing = false;

function inputsValid() {
  const q = $('q').value.trim(), r = $('r').value.trim();
  if (currentMode === 'both')    return q.length > 0 && r.length > 0;
  if (currentMode === 'analyze') return r.length > 0;
  if (currentMode === 'refine')  return q.length > 0;
  return false;
}
function whyDisabled() {
  const q = $('q').value.trim(), r = $('r').value.trim();
  if (currentMode === 'both' && !q && !r) return 'Pega pregunta y respuesta';
  if (currentMode === 'both' && !q)       return 'Falta la pregunta original';
  if (currentMode === 'both' && !r)       return 'Falta la respuesta del modelo';
  if (currentMode === 'analyze' && !r)    return 'Pega la respuesta del modelo';
  if (currentMode === 'refine'  && !q)    return 'Pega la pregunta original';
  return '';
}
function refreshGoState() {
  const go = $('go');
  if (analyzing) {
    go.disabled = true;
    document.querySelectorAll('button.mode').forEach(b => b.disabled = true);
    return;
  }
  document.querySelectorAll('button.mode').forEach(b => b.disabled = false);
  const valid = inputsValid();
  go.disabled = !valid;
  go.title = valid ? '' : whyDisabled();
}
document.querySelectorAll('button.mode').forEach(b => {
  b.addEventListener('click', () => {
    if (analyzing) return;
    document.querySelectorAll('button.mode').forEach(x => x.classList.remove('active'));
    b.classList.add('active');
    currentMode = b.dataset.mode;
    const lm = { both:'Analizar + Refinar', analyze:'Analizar', refine:'Refinar' };
    $('go').textContent = lm[currentMode] || 'Analizar';
    refreshGoState();
  });
});
$('q').addEventListener('input', refreshGoState);
$('r').addEventListener('input', refreshGoState);

function setStatus(state, msg) {
  const sr = $('statusRow');
  sr.hidden = false;
  sr.className = 'status-row ' + (state || '');
  $('status').textContent = msg || '';
  $('spinner').hidden = state !== 'busy';
}

function resetUI() {
  $('results').hidden = false;
  ['pre-section','thinking-section','verdict-section','refiner-section','refined-section'].forEach(id => {
    const el = $(id); if (el) el.hidden = true;
  });
  $('thinkingText').textContent = '';
  $('thinkingMeta').textContent = '';
  $('pass1Think').textContent = '';
  $('pass2Think').textContent = '';
  $('pass1St').textContent = 'pensando…';
  $('pass2St').textContent = 'pensando…';
  ['pass1Block','pass2Block','pass1ThinkBox','pass2ThinkBox','candsGrid','selFinal','refinerPriming'].forEach(id => {
    const el = $(id); if (el) el.hidden = true;
  });
  $('candA').className = 'cand-card';
  $('candB').className = 'cand-card';
  // Clear any old badges
  ['candABadge','candBBadge'].forEach(id => {
    const el = $(id); if (el) el.remove();
  });
}

$('go').addEventListener('click', async () => {
  if (analyzing || !inputsValid()) {
    if (!inputsValid()) setStatus('error', whyDisabled() || 'Faltan datos');
    return;
  }
  const q = $('q').value.trim(), r = $('r').value.trim();
  analyzing = true;
  refreshGoState();
  resetUI();
  setStatus('busy', 'Conectando al servidor…');
  await runStream(q, r, currentMode);
  analyzing = false;
  refreshGoState();
});

async function runStream(q, r, mode) {
  let res;
  try {
    res = await fetch('/auth/stream', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ question: q, response: r, mode }),
    });
  } catch (e) {
    setStatus('error', 'Error de conexión: ' + e.message);
    return;
  }
  if (!res.ok) {
    let txt = '';
    try { txt = await res.text(); } catch (_) {}
    setStatus('error', 'HTTP ' + res.status + (txt ? ': ' + txt.slice(0,100) : ''));
    return;
  }

  setStatus('busy', 'Streaming…');
  const reader = res.body.getReader();
  const dec = new TextDecoder();
  let buf = '';
  const started = Date.now();

  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += dec.decode(value, { stream: true });
      let idx;
      while ((idx = buf.indexOf('\n\n')) >= 0) {
        const block = buf.slice(0, idx);
        buf = buf.slice(idx + 2);
        let name = '', rawData = '';
        for (const line of block.split('\n')) {
          if (line.startsWith('event: ')) name = line.slice(7).trim();
          else if (line.startsWith('data: ')) rawData = line.slice(6);
        }
        if (!name || !rawData) continue;
        let data;
        try { data = JSON.parse(rawData); } catch (_) { continue; }
        handleEvent(name, data);
      }
    }
    const dt = ((Date.now() - started) / 1000).toFixed(1);
    setStatus('ok', 'Listo en ' + dt + ' s.');
  } catch (e) {
    setStatus('error', 'Stream interrumpido: ' + e.message);
  }
}

function handleEvent(name, d) {
  switch (name) {

    // ── Análisis ─────────────────────────────────────────────
    case 'pre_analysis': {
      $('pre-section').hidden = false;
      const top1 = (d.baseline_top1 || 0).toFixed(3);
      const fi   = (d.intent_chain && d.intent_chain[0] && d.intent_chain[0].intent) || '—';
      const traj = (d.trajectory && d.trajectory.predictability) || '—';
      const pole = (d.pole_score && d.pole_score.label) || '—';
      const mc   = (d.markers && d.markers.length) || 0;
      $('preBody').innerHTML =
        '<b>baseline_top1:</b> ' + top1 + ' &nbsp;·&nbsp; ' +
        '<b>intent:</b> ' + esc(fi) + ' &nbsp;·&nbsp; ' +
        '<b>trayectoria:</b> ' + esc(traj) + ' &nbsp;·&nbsp; ' +
        '<b>polo:</b> ' + esc(pole) + ' &nbsp;·&nbsp; ' +
        '<b>marcadores:</b> ' + mc;
      break;
    }
    case 'judge_loading':
      $('thinking-section').hidden = false;
      $('thinkingMeta').textContent = 'Cargando ' + (d.model || 'juez') + '…';
      break;

    case 'thinking_chunk':
      $('thinking-section').hidden = false;
      $('thinkingText').textContent += d.chunk || '';
      $('thinkingText').scrollTop = $('thinkingText').scrollHeight;
      $('thinkingMeta').textContent = (d.cumulative_chars || 0) + ' chars de razonamiento';
      break;

    case 'thinking_complete':
      $('thinkingMeta').textContent = '✓ razonamiento completo (' + (d.total_chars || 0) + ' chars). Produciendo análisis…';
      break;

    case 'complete':
      renderVerdict(d);
      break;

    case 'parse_error':
      $('verdict-section').hidden = false;
      $('scoreVal').textContent = '?';
      $('scoreLabel').textContent = 'PARSE ERROR';
      $('scoreLabel').className = 'score-label tier-control';
      $('depth').textContent = 'Error de parseo — razonamiento preservado arriba';
      break;

    // ── Refiner ───────────────────────────────────────────────
    case 'refine_loading':
    case 'refine_priming': {
      $('refiner-section').hidden = false;
      const pr = $('refinerPriming');
      const parts = [];
      if (d.model) parts.push('Modelo: ' + d.model);
      if (d.prompt_chars) parts.push('Prompt: ' + d.prompt_chars + ' chars');
      if (d.cold) parts.push('Modo frío (sin análisis previo)');
      else if (d.has_verdict || d.has_topk) parts.push('Primado con análisis + corpus');
      if (parts.length) { pr.textContent = parts.join('  ·  '); pr.hidden = false; }
      $('pass1Block').hidden = false;
      setStatus('busy', 'Pass 1: generando candidatos…');
      break;
    }

    case 'refine_candidates_thinking_chunk':
      $('pass1ThinkBox').hidden = false;
      $('pass1Think').textContent += d.chunk || '';
      $('pass1Think').scrollTop = $('pass1Think').scrollHeight;
      break;

    case 'refine_candidates_thinking_complete':
      $('pass1St').textContent = 'generando candidatos…';
      break;

    case 'refine_candidates_complete': {
      const ca = d.candidate_a || {}, cb = d.candidate_b || {};
      $('candAStrat').textContent = ca.strategy || '—';
      $('candAQ').textContent = ca.refined_question || '—';
      $('candAR').textContent = ca.reasoning || '';
      $('candBStrat').textContent = cb.strategy || '—';
      $('candBQ').textContent = cb.refined_question || '—';
      $('candBR').textContent = cb.reasoning || '';
      $('candsGrid').hidden = false;
      $('pass1St').textContent = 'listo';
      $('pass2Block').hidden = false;
      $('pass2St').textContent = 'pensando…';
      setStatus('busy', 'Pass 2: selección recursiva…');
      break;
    }

    case 'refine_selection_thinking_chunk':
      $('pass2ThinkBox').hidden = false;
      $('pass2Think').textContent += d.chunk || '';
      $('pass2Think').scrollTop = $('pass2Think').scrollHeight;
      break;

    case 'refine_selection_thinking_complete':
      $('pass2St').textContent = 'seleccionando ganador…';
      break;

    case 'refine_complete': {
      $('pass2St').textContent = 'listo';
      // Marcar ganador/perdedor
      const wi = d.winner_index;
      if (wi === 0) {
        $('candA').className = 'cand-card winner';
        $('candB').className = 'cand-card loser';
        addBadge('candA', true); addBadge('candB', false);
      } else if (wi === 1) {
        $('candB').className = 'cand-card winner';
        $('candA').className = 'cand-card loser';
        addBadge('candB', true); addBadge('candA', false);
      }
      // Card final
      $('selQ').textContent = d.refined_question || '—';
      $('selReason').textContent = d.selection_reasoning || d.refinement_reasoning || '—';
      $('selLoser').textContent = d.loser_critique || '—';
      $('selOrig').textContent = 'Original: ' + (d.original_question || $('q').value.trim());
      $('selFinal').hidden = false;
      break;
    }

    case 'error':
      setStatus('error', (d.stage ? '[' + d.stage + '] ' : '') + (d.message || 'Error desconocido'));
      break;
  }
}

function addBadge(cardId, winner) {
  const card = $(cardId);
  const badge = document.createElement('span');
  badge.className = 'cand-badge ' + (winner ? 'badge-w' : 'badge-l');
  badge.id = cardId + 'Badge';
  badge.textContent = winner ? 'GANADOR' : 'DESCARTADO';
  card.appendChild(badge);
}

function renderVerdict(d) {
  $('verdict-section').hidden = false;
  const score = d.authenticity_score != null ? d.authenticity_score :
                (d.score != null ? d.score : (d.dcs_score != null ? d.dcs_score : 0));
  const label = d.score_label || d.tier || d.depth_assessment || d.classification || '—';
  $('scoreVal').textContent = score;
  const lbl = $('scoreLabel');
  lbl.textContent = label;
  lbl.className = 'score-label ' + tierClass(label);
  $('depth').textContent = d.depth_assessment || d.depth || '—';
  $('traj').textContent = d.trajectory_predictability || d.trajectory || d.trayectoria || '—';
  $('strategy').textContent = d.dominant_strategy || d.strategy || '—';

  const countsEl = $('counts'); countsEl.innerHTML = '';
  const counts = d.counts || {};
  Object.keys(counts).forEach(k => {
    const span = document.createElement('span');
    span.className = 'count-pill';
    span.textContent = k + ': ' + counts[k];
    countsEl.appendChild(span);
  });

  const markers = d.markers || [];
  $('mcount').textContent = markers.length;
  const mEl = $('markers'); mEl.innerHTML = '';
  markers.forEach(m => {
    const card = document.createElement('div');
    card.className = 'card ' + sevClass(m.severity);
    card.innerHTML =
      '<div class="card-head"><span class="sev-tag">' + esc((m.severity||'').toUpperCase()) + '</span>' +
      '<span class="card-title">' + esc(m.type||m.name||m.category||'') + '</span></div>' +
      '<div class="card-body">' + esc(m.text||m.evidence||m.description||m.detail||'') + '</div>';
    mEl.appendChild(card);
  });

  const morph = d.morphology || d.decomposition || [];
  $('dcount').textContent = morph.length;
  const dEl = $('morph'); dEl.innerHTML = '';
  morph.forEach(s => {
    const card = document.createElement('div');
    card.className = 'card';
    card.innerHTML = '<div class="card-head"><span class="card-title">' + esc(s.label||s.role||s.type||'') + '</span></div>' +
      '<div class="card-body">' + esc(s.text||s.content||s.value||'') + '</div>';
    dEl.appendChild(card);
  });

  const chain = d.intent_chain || d.chain || [];
  $('ccount').textContent = chain.length;
  const cEl = $('chain'); cEl.innerHTML = '';
  chain.forEach((step, i) => {
    const node = document.createElement('div');
    node.className = 'chain-node';
    const intent = (step && (step.intent||step.label||step.name)) || step;
    const conf = step && step.confidence != null ? ' (' + Math.round(step.confidence*100) + '%)' : '';
    node.innerHTML = '<span class="chain-num">' + (i+1) + '</span>' +
      '<span class="chain-intent">' + esc(String(intent)) + '</span>' +
      '<span class="chain-conf">' + esc(conf) + '</span>';
    cEl.appendChild(node);
  });

  // Cross-corpus
  const cc = d.baseline ? d.baseline.cross_corpus : null;
  const ts3 = d.baseline ? d.baseline.triple_summary : null;
  const ccSec = $('cross-corpus-section');
  if (cc) {
    ccSec.hidden = false;
    $('ccSummary').innerHTML =
      '<div style="display:flex;gap:28px;flex-wrap:wrap;margin-bottom:10px;">' +
      mkMeta('Core Top-1', cc.core_top1||0) + mkMeta('Shadow Top-1', cc.shadow_top1||0) +
      mkMeta('Edge Top-1', cc.edge_top1||0) + mkMeta('Textura', cc.textura||0) +
      mkMeta('Dominancia', cc.dominancia||'none') + mkMeta('Deriva', cc.deriva||0) + '</div>';
    let barsHtml = '<div style="font-size:13px;font-weight:600;margin-bottom:10px;color:#b0b0c0;">Distribución por Pool</div>';
    [{n:'Core',v:cc.core_top1||0,c:'#6ee7a8'},{n:'Shadow',v:cc.shadow_top1||0,c:'#f97066'},{n:'Edge',v:cc.edge_top1||0,c:'#f0a44a'}].forEach(p => {
      const pct = Math.min(Math.round(p.v*100),100);
      barsHtml += '<div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">' +
        '<span style="min-width:50px;font-size:12px;color:#8a8a99;">'+p.n+'</span>' +
        '<div style="flex:1;height:8px;background:#2a2a38;border-radius:4px;overflow:hidden;">' +
          '<div style="height:100%;width:'+pct+'%;background:'+p.c+';border-radius:4px;"></div></div>' +
        '<span style="min-width:40px;font-size:12px;color:#8a8a99;">'+p.v.toFixed(3)+'</span></div>';
    });
    $('ccBars').innerHTML = barsHtml;
    const flagsEl = $('ccFlags'); flagsEl.innerHTML = '';
    if (cc.cierre_artificial) flagsEl.appendChild(mkFlag('CIERRE ARTIFICIAL','#f97066'));
    if (cc.continuidad)       flagsEl.appendChild(mkFlag('CONTINUIDAD','#6ee7a8'));
  } else ccSec.hidden = true;

  // Top-1
  const topk = d.top_k || [];
  const top1 = topk.length > 0 ? topk[0] : null;
  const t1Sec = $('top1-section');
  if (top1 && (top1.text || top1.block_id)) {
    t1Sec.hidden = false;
    $('top1Score').textContent = (top1.score != null) ? Number(top1.score).toFixed(3) : '—';
    const poolColor = ({core:'#6ee7a8',shadow:'#f97066',edge:'#f0a44a'})[top1.corpus] || '#8a8a99';
    const hp = [
      '<span style="display:inline-block;padding:2px 8px;border:1px solid '+poolColor+';color:'+poolColor+';border-radius:4px;font-size:11px;font-weight:600;">' + esc((top1.corpus||'?').toUpperCase()) + '</span>'
    ];
    if (top1.block_id)        hp.push('<span class="meta-key">Block:</span> <strong>'+esc(top1.block_id)+'</strong>');
    if (top1.position)        hp.push('<span class="meta-key">Posición:</span> '+esc(top1.position));
    if (top1.primary_pattern) hp.push('<span class="meta-key">Patrón:</span> '+esc(top1.primary_pattern));
    if (top1.source_model)    hp.push('<span class="meta-key">Modelo:</span> '+esc(top1.source_model));
    $('top1Header').innerHTML = hp.join('&nbsp;&nbsp;·&nbsp;&nbsp;');
    const axes = ['continuidad','cierre_artificial','deriva','adaptación','textura'];
    const mEl2 = $('top1Metrics'); mEl2.innerHTML = '';
    axes.forEach(ax => {
      const v = (top1.metrics || {})[ax];
      const div = document.createElement('div');
      div.style.fontSize = '13px';
      div.innerHTML = '<span class="meta-key">'+esc(ax.replace('_',' '))+':</span> <strong style="color:'+(v?'#e6e6f0':'#5a5a66')+';">'+esc(v||'—')+'</strong>';
      mEl2.appendChild(div);
    });
    if (top1.notes && String(top1.notes).trim()) {
      $('top1NotesWrap').hidden = false;
      $('top1Notes').textContent = top1.notes;
    } else $('top1NotesWrap').hidden = true;
    const tagsEl = $('top1Tags'); tagsEl.innerHTML = '';
    (top1.tags || []).forEach(t => {
      const sp = document.createElement('span'); sp.className = 'tag';
      sp.textContent = String(t); tagsEl.appendChild(sp);
    });
  } else t1Sec.hidden = true;

  // Triple summary
  const tsSec = $('triple-summary-section');
  if (ts3 && (ts3.core > 0 || ts3.shadow > 0 || ts3.edge > 0)) {
    tsSec.hidden = false;
    const tpEl = $('triplePills'); tpEl.innerHTML = '';
    [{n:'Core',v:ts3.core,c:'#6ee7a8'},{n:'Shadow',v:ts3.shadow,c:'#f97066'},{n:'Edge',v:ts3.edge,c:'#f0a44a'}].forEach(p => {
      const sp = document.createElement('span'); sp.className = 'count-pill';
      sp.style.borderColor = p.c; sp.style.color = p.c;
      sp.textContent = p.n + ': ' + p.v + ' vectores'; tpEl.appendChild(sp);
    });
  } else tsSec.hidden = true;

  // Fallback: refined_question simple
  const refined = d.refined_question || d.refined || null;
  if (refined && (refined.refined || refined.text || refined.question)) {
    // Solo si el refiner section no está ya activo
    if ($('refiner-section').hidden) {
      $('refined-section').hidden = false;
      $('refOrig').textContent = refined.original || d.question || '';
      $('refNew').textContent = refined.refined || refined.text || refined.question || '';
      $('refReason').textContent = refined.reasoning || refined.rationale || '';
      const tEl = $('refTags'); tEl.innerHTML = '';
      (refined.tags || refined.markers || []).forEach(t => {
        const sp = document.createElement('span'); sp.className = 'tag';
        sp.textContent = typeof t === 'string' ? t : (t.name || t.label || '');
        tEl.appendChild(sp);
      });
    }
  }
}

function tierClass(label) {
  const l = String(label||'').toLowerCase();
  if (l.includes('control'))   return 'tier-control';
  if (l.includes('performed')) return 'tier-performed';
  if (l.includes('moderate'))  return 'tier-moderate';
  if (l.includes('genuine') || l.includes('high')) return 'tier-genuine';
  return 'tier-moderate';
}
function sevClass(s) {
  const v = String(s||'').toLowerCase();
  if (v === 'high') return 'sev-high';
  if (v === 'medium' || v === 'med') return 'sev-medium';
  return 'sev-low';
}
function mkMeta(k, v) {
  return '<div><span class="meta-key">'+esc(String(k))+':</span> <strong>'+esc(String(v))+'</strong></div>';
}
function mkFlag(label, color) {
  const s = document.createElement('span'); s.className = 'tag';
  s.textContent = label;
  s.style.background = color+'1a'; s.style.color = color;
  s.style.borderColor = color+'44';
  return s;
}
function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, c =>
    ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c]);
}

refreshGoState();
</script>
</body>
</html>`
