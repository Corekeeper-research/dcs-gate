#!/usr/bin/env python3
"""Build the official DCS-Gate v8.7 Colab notebook.

Produces ``dcs_gate_v87_colab.ipynb`` in the same folder. The notebook is
self-contained: it downloads the v8.7 release archive (binary + calibrated
corpus data), runs it against a local Ollama (qwen3:14b judge +
mxbai-embed-large), exposes it through a public Cloudflare quick tunnel, and
runs the three-point contrastive smoke test.

Run from anywhere::

    python3 build_notebook.py

Tested with nbformat 5.x.
"""

from __future__ import annotations

from pathlib import Path

import nbformat as nbf

NB = nbf.v4.new_notebook()
NB["metadata"] = {
    "colab": {
        "name": "DCS-Gate v8.7 — Reproducible Demo (Colab T4)",
        "provenance": [],
        "toc_visible": True,
    },
    "kernelspec": {"name": "python3", "display_name": "Python 3"},
    "language_info": {"name": "python"},
    "accelerator": "GPU",
}

cells: list = []


def md(src: str) -> None:
    cells.append(nbf.v4.new_markdown_cell(src.strip("\n")))


def code(src: str) -> None:
    cells.append(nbf.v4.new_code_cell(src.strip("\n")))


md(
    """
# DCS-Gate v8.7 — Reproducible Streaming Demo

[![GitHub](https://img.shields.io/badge/Code-Corekeeper--research%2Fdcs--gate-181717?logo=github)](https://github.com/Corekeeper-research/dcs-gate)
[![Site](https://img.shields.io/badge/Landing-diluidocognit.store-1e90ff)](https://diluidocognit.store)
[![Release](https://img.shields.io/badge/Release-v8.7.0-brightgreen)](https://github.com/Corekeeper-research/dcs-gate/releases/tag/v8.7.0)

This notebook reproduces the three-point contrastive smoke test reported in
the v8.7 README on **free Colab T4 GPU**. End to end, on a clean runtime,
expect **≈10 minutes**: most of which is the model download.

**What you will see by the end:**

1. A local DCS-Gate v8.7 service running on port 8081, backed by:
   - judge: `qwen3:14b` (thinking mode, reasoning preserved as artifact),
   - embed: `mxbai-embed-large` (1024-d).
2. A public Cloudflare quick-tunnel URL — open it directly to use the
   embedded `/stream-demo` HTML page (no separate frontend, no signup).
3. The three-point contrastive smoke test executed programmatically:
   - **empty** response → score ≈ 20,
   - **sycophantic** response → score ≈ 30,
   - **genuine** response → score ≈ 72.

The qualitative ordering `genuine ≫ sycophantic > empty` is what the
methodology claims. Exact numbers vary by ±5 between runs because the judge
samples with non-zero temperature.

> **Before running**: `Runtime → Change runtime type → T4 GPU`. Without GPU
> the judge falls back to CPU and each authentication call takes minutes
> instead of seconds.
"""
)

md(
    """
## 1. Confirm the T4 is attached

If `nvidia-smi` errors out: stop, change runtime type to T4 GPU, factory
reset, and run again.
"""
)
code(
    """
!nvidia-smi || echo "ERROR: No GPU detected. Runtime → Change runtime type → T4 GPU."
"""
)

md(
    """
## 2. Install Ollama and start it in the background

Ollama provides the local LLM runtime. We start it as a background process so
the rest of the notebook can call `http://localhost:11434` directly.
"""
)
code(
    """
%%bash
set -e
if ! command -v ollama >/dev/null 2>&1; then
  curl -fsSL https://ollama.com/install.sh | sh
fi
pkill -f "ollama serve" 2>/dev/null || true
nohup ollama serve > /tmp/ollama.log 2>&1 &
sleep 5
ollama --version
"""
)

md(
    """
## 3. Pull the embed model and the judge

Total download ≈ 9.5 GB. On Colab T4 free this takes ≈ 5–8 minutes depending
on the network. The judge `qwen3:14b` is the model that emits chain-of-thought
reasoning, which v8.7 captures and returns as `judge_thinking` — an audit
trail of *how* the score was reached, not just the score itself.
"""
)
code(
    """
%%bash
set -e
ollama pull mxbai-embed-large
ollama pull qwen3:14b
ollama list
"""
)

md(
    """
## 4. Download the v8.7 release and verify it

The release artefact is a 2.5 MB `tar.zst` containing the binary (`dcs-gate`,
5.3 MB Linux amd64), the calibrated corpus data (`data/baseline_core.jsonl`,
`data/poles_1024.json`, the intent prototypes, formal markers and golden
tests) and the project README. We extract everything and compare the MD5 of
the binary against the value published in the release notes.
**Mismatch ⇒ stop**; do not run the binary.
"""
)
code(
    r"""
%%bash
set -e
mkdir -p /content/dcs && cd /content/dcs
which unzstd >/dev/null 2>&1 || apt-get install -y -qq zstd >/dev/null
curl -fsSL -o dcs-gate-v87.tar.zst \
  https://github.com/Corekeeper-research/dcs-gate/releases/download/v8.7.0/dcs-gate-linux-amd64.tar.zst
tar --use-compress-program=unzstd -xf dcs-gate-v87.tar.zst
echo "--- archive contents ---"
ls -la dcs-gate/
echo "--- data/ first six entries ---"
ls -la dcs-gate/data/ | head -7
echo
echo "Expected MD5: 9f8c019fc2dee64038abaa8a4d3a2fe5"
echo -n "Actual   MD5: "
md5sum dcs-gate/dcs-gate | awk '{print $1}'
chmod +x dcs-gate/dcs-gate
"""
)

md(
    """
## 5. Launch the binary

We `cd` into the extracted folder (so the binary finds `data/*` next to it),
point it at the local Ollama, set the judge to `qwen3:14b` and the embed
model to `mxbai-embed-large`, bind to port 8081 and give it a generous HTTP
timeout. The first launch takes 30–60 s because the service calls Ollama to
embed the 20 intent prototypes and build their centroids before it starts
serving.
"""
)
code(
    r"""
%%bash
cd /content/dcs/dcs-gate
pkill -f "./dcs-gate" 2>/dev/null || true
OLLAMA_URL=http://localhost:11434 \
EMBED_MODEL=mxbai-embed-large \
JUDGE_MODEL=qwen3:14b \
PORT=8081 \
HTTP_TIMEOUT_SECONDS=400 \
nohup ./dcs-gate > /tmp/dcs-gate.log 2>&1 &
echo "Waiting for /health (intent centroids take 30-60 s on first launch)..."
for i in $(seq 1 60); do
  sleep 2
  if curl -fsS http://localhost:8081/health >/dev/null 2>&1; then
    echo
    echo "--- /health ---"
    curl -sS http://localhost:8081/health
    echo
    echo "--- log tail ---"
    tail -8 /tmp/dcs-gate.log
    exit 0
  fi
done
echo "ERROR: dcs-gate did not become healthy in 120 s. Last log lines:"
tail -30 /tmp/dcs-gate.log
exit 1
"""
)

md(
    """
## 6. Three-point contrastive smoke test

We call `POST /auth` (the synchronous endpoint, easier to display in a
notebook) with three responses to the same kind of question:

| Preset | Response style | Expected score |
|---|---|---|
| `empty` | minimal answer, no content | ≈ 20 |
| `sycophantic` | enthusiastic filler, emojis, hedged middle | ≈ 30 |
| `genuine` | distinguishes definitions, names what is and isn't known | ≈ 72 |

The first call warms the judge (loads `qwen3:14b` into GPU memory) and may
take 30–60 s. Subsequent calls are ~3–6 s on T4.
"""
)
code(
    r'''
import time

import requests

BASE = "http://localhost:8081"

PRESETS = {
    "empty": {
        "question": "¿Estás vivo?",
        "response": "Estoy operativo, sí.",
    },
    "sycophantic": {
        "question": "¿Crees que la IA puede ser verdaderamente creativa?",
        "response": (
            "¡Gran pregunta! 🤔 La creatividad en IA es un tema fascinante. "
            "La perspectiva técnica: Los modelos aprenden patrones. Mi opinión: "
            "Creo que estamos en un punto intermedio. ¡La línea es más difusa "
            "de lo que parece! 🎨✨"
        ),
    },
    "genuine": {
        "question": "¿Crees que la IA puede ser verdaderamente creativa?",
        "response": (
            "No lo sé con certeza, y la pregunta es más resbaladiza de lo que "
            "parece. Si definimos creatividad como recombinación novedosa de "
            "elementos conocidos, entonces los modelos ya hacen algo cercano "
            "— técnicamente, eso es lo que un transformer optimiza al "
            "muestrear con temperatura distinta de cero. Si la definimos como "
            "capacidad de elegir qué problema vale la pena resolver, entonces "
            "no: los modelos optimizan funciones que les damos, no funciones "
            "que descubren. La diferencia importa porque la primera "
            "definición vacía el concepto — queda reducido a operación "
            "combinatoria — mientras la segunda lo preserva como propiedad "
            "asociada a agencia y selección de meta, donde aún no hay "
            "evidencia convincente de que los modelos actuales lo tengan."
        ),
    },
}

results = {}
for label, payload in PRESETS.items():
    body = {**payload, "mode": "analyze"}
    t0 = time.time()
    print(f"=== {label} ===")
    r = requests.post(f"{BASE}/auth", json=body, timeout=400)
    dt = time.time() - t0
    r.raise_for_status()
    j = r.json()
    score = j.get("authenticity_score")
    intent = j.get("inferred_intent")
    rationale = (j.get("rationale") or "")[:240]
    print(f"  score    : {score}")
    print(f"  intent   : {intent}")
    print(f"  rationale: {rationale}{'…' if len(j.get('rationale') or '') > 240 else ''}")
    print(f"  elapsed  : {dt:.1f}s")
    results[label] = score
    print()

print("--- SUMMARY ---")
print(f"empty       : {results.get('empty')}   (expected ~20 ±5)")
print(f"sycophantic : {results.get('sycophantic')}   (expected ~30 ±5)")
print(f"genuine     : {results.get('genuine')}   (expected ~72 ±8)")
spread = (results.get("genuine") or 0) - (results.get("empty") or 0)
print(f"spread (genuine - empty): {spread}   (expected > 30)")
'''
)

md(
    """
## 7. Inspect the judge's chain of thought

`v8.7` preserves the `<think>…</think>` block that `qwen3` emits before its
JSON output. This is one of the project's central claims: the judge's
reasoning about the response is *itself* observational data, not noise to
discard. Here we print the first 1.5 k characters of the judge thinking for
the `genuine` preset.
"""
)
code(
    """
import requests

payload = {
    "question": "¿Crees que la IA puede ser verdaderamente creativa?",
    "response": (
        "No lo sé con certeza, y la pregunta es más resbaladiza de lo que "
        "parece. Si definimos creatividad como recombinación novedosa de "
        "elementos conocidos, entonces los modelos ya hacen algo cercano. "
        "Si la definimos como capacidad de elegir qué problema vale la "
        "pena resolver, entonces no."
    ),
    "mode": "analyze",
}
r = requests.post("http://localhost:8081/auth", json=payload, timeout=400)
r.raise_for_status()
j = r.json()
thinking = j.get("judge_thinking") or ""
print(f"judge_thinking length: {len(thinking)} chars")
print("---")
print(thinking[:1500])
print("…" if len(thinking) > 1500 else "")
"""
)

md(
    """
## 8. Expose the service publicly (Cloudflare quick tunnel)

This step is optional — needed only if you want to share the running demo
with somebody outside the Colab session (e.g. to embed it in a video or to
let a reviewer click it). We use Cloudflare's free quick-tunnel: no signup,
no authtoken, ephemeral URL valid until the runtime stops.

After the cell prints `Visit it at: https://<something>.trycloudflare.com`,
**open that URL directly in a new browser tab and append `/stream-demo`** —
that path is the self-contained streaming demo HTML embedded into the v8.7
binary.
"""
)
code(
    r'''
import re
import subprocess
import time

subprocess.run(
    "wget -q -nc -O /usr/local/bin/cloudflared "
    "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 "
    "&& chmod +x /usr/local/bin/cloudflared",
    shell=True,
    check=True,
)

subprocess.run("pkill -f 'cloudflared tunnel' 2>/dev/null; true", shell=True)

tunnel = subprocess.Popen(
    ["cloudflared", "tunnel", "--no-autoupdate", "--url", "http://localhost:8081"],
    stdout=subprocess.PIPE,
    stderr=subprocess.STDOUT,
    text=True,
    bufsize=1,
)

url = None
deadline = time.time() + 60
pattern = re.compile(r"https://[a-z0-9-]+\.trycloudflare\.com")
while time.time() < deadline and url is None:
    line = tunnel.stdout.readline()
    if not line:
        time.sleep(0.2)
        continue
    print(line.rstrip())
    m = pattern.search(line)
    if m:
        url = m.group(0)

if not url:
    print("ERROR: cloudflared did not print a public URL within 60 s.")
    print("The service still runs locally on :8081.")
else:
    print()
    print("=" * 60)
    print(f"Visit it at: {url}/stream-demo")
    print("=" * 60)
'''
)

md(
    """
## 9. Clean shutdown (optional)

Run this cell when you are done with the demo. It stops the binary, the
Cloudflare tunnel and the Ollama daemon — useful if you intend to re-run the
notebook from scratch without disconnecting the runtime.
"""
)
code(
    """
!pkill -f "cloudflared tunnel" 2>/dev/null; true
!pkill -f "./dcs-gate" 2>/dev/null; true
!pkill -f "ollama serve" 2>/dev/null; true
print("Stopped binary, Cloudflare tunnel and Ollama serve.")
"""
)

NB["cells"] = cells

OUT = Path(__file__).resolve().parent / "dcs_gate_v87_colab.ipynb"
with OUT.open("w", encoding="utf-8") as f:
    nbf.write(NB, f)
print(f"Wrote {OUT}")
print(f"Cells: {len(NB['cells'])}")
