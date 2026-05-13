#!/usr/bin/env python3
"""Build the reproducibility notebook for DCS-Gate v8.7.

Run from the repo root:

    python3 reproducibility/build_notebook.py

Produces reproducibility/qwen3_14b_smoke_test.ipynb.

The notebook auto-detects the runtime environment (local Jupyter, Kaggle
hosted notebook, or Colab hosted backend) and adapts paths and warnings
accordingly. Each cell that touches process state begins with a defensive
"kill anything that could conflict" step — past sessions on this project
have hit port collisions when an earlier runtime crashed and left Ollama or
dcs-gate processes bound to :11434 / :8081.

Reference run for the validation in docs/index.html:
    Author's local Jupyter workstation, 2 × Tesla T4 (16 GB each, 30 GB
    combined usable VRAM), 31 GB system RAM, Ubuntu host.
    Cold warmup of qwen3:14b: ~32 s. Inference per /auth request: ~150-210 s.
"""
from __future__ import annotations

import json
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
OUT = REPO / "reproducibility" / "qwen3_14b_smoke_test.ipynb"


def code_cell(source: str) -> dict:
    return {
        "cell_type": "code",
        "metadata": {},
        "execution_count": None,
        "outputs": [],
        "source": source.lstrip("\n").splitlines(keepends=True),
    }


def md_cell(source: str) -> dict:
    return {
        "cell_type": "markdown",
        "metadata": {},
        "source": source.lstrip("\n").splitlines(keepends=True),
    }


CELLS = [
    md_cell(r"""
# DCS-Gate v8.7 — reproducibility notebook

Cold-start path from a fresh Python environment to a running `qwen3:14b` thinking-mode
judge serving `/auth/stream` over Server-Sent Events. Auto-detects the runtime
(local Jupyter, Kaggle hosted, or Colab hosted) and adapts paths accordingly.

**Reference run (the smoke test cited in [diluidocognit.store](https://diluidocognit.store)):**

- Hardware: author's local Jupyter workstation, **2 × Tesla T4** (16 GB each, ~30 GB
  combined usable VRAM), 31 GB system RAM, ~8 TB disk.
- Cold warmup of `qwen3:14b` to VRAM: **~32 s**.
- Inference per `/auth` request: **150–210 s** (streamed live over SSE so the user
  sees the judge's thinking trace as it arrives, not a blank screen).
- Expected smoke-test output: three responses to the same question score **30 / 20 / 72**
  (sycophantic-emoji / empty-non-response / authentic-bi-frontal exploration).

**Other supported environments**

| environment | GPUs | recommended judge |
|---|---|---|
| Local Jupyter / VM with 2 × T4 | 32 GB combined | `qwen3:14b` (reference) |
| Kaggle hosted notebook T4 × 2 | 32 GB combined | `qwen3:14b` (free 30 hrs/week) |
| Colab Pro+ with A100 (40/80 GB) | 40–80 GB | `qwen3:14b` (overkill, fast) |
| Colab Pro with L4 (24 GB) | 24 GB | `qwen3:14b` (tight; consider Q4 quant) |
| Colab free with T4 (15 GB) | 15 GB | `qwen3:8b` recommended; `qwen3:14b` will OOM with thinking |
| CPU only | n/a | not supported (qwen3:14b on CPU ≈ 30 min/request) |

**Why every cell starts with a kill step.** This notebook is meant to be re-runnable.
Past sessions on this project hit port collisions on `:11434` (Ollama), `:8081`
(dcs-gate) and `:4040` (ngrok) when an earlier runtime crashed and left processes
behind. Each non-trivial cell first kills anything that could conflict before doing
its real work.
"""),
    md_cell(r"""
## 0.1 Environment detection — figure out where we're running

Detects local Jupyter / Kaggle hosted / Colab hosted, picks the correct working
directory and pip flags, and stores the chosen `WORK` path in a global. Run this
first.
"""),
    code_cell(r"""
import os, sys, subprocess

# ----- detect host environment -----
if os.path.exists("/kaggle/working"):
    ENV  = "kaggle"
    WORK = "/kaggle/working/dcs-gate"
    PIP_FLAGS = "--user"
elif os.path.exists("/content/sample_data"):
    ENV  = "colab"
    WORK = "/content/dcs-gate"
    PIP_FLAGS = ""
else:
    ENV  = "local"
    WORK = os.path.join(os.getcwd(), "dcs-gate")
    PIP_FLAGS = ""

os.makedirs(WORK, exist_ok=True)
print(f"environment : {ENV}")
print(f"work dir    : {WORK}")
print(f"pip flags   : {PIP_FLAGS!r}")

# ----- enumerate GPUs and decide which judge model is realistic -----
def probe_gpus():
    r = subprocess.run("nvidia-smi --query-gpu=name,memory.total --format=csv,noheader",
                       shell=True, capture_output=True, text=True)
    if r.returncode != 0:
        return []
    rows = []
    for ln in r.stdout.strip().splitlines():
        parts = [x.strip() for x in ln.split(",")]
        if len(parts) == 2:
            name, mem = parts
            mem_gb = int(mem.split()[0]) / 1024 if "MiB" in mem else float(mem.split()[0])
            rows.append((name, mem_gb))
    return rows

gpus = probe_gpus()
print()
if not gpus:
    print("⚠ NO GPU detected — qwen3:14b on CPU is unusable (~30 min/request).")
    print("  Switch the runtime to one with a GPU before continuing.")
    RECOMMENDED_JUDGE = None
else:
    print(f"GPUs detected: {len(gpus)}")
    for name, mem in gpus:
        print(f"  {name}  ({mem:.0f} GB)")
    total_vram = sum(m for _, m in gpus)
    max_single = max(m for _, m in gpus)

    if total_vram >= 24:
        RECOMMENDED_JUDGE = "qwen3:14b"
        if len(gpus) >= 2 and "T4" in gpus[0][0]:
            print(f"\n✓ reference environment ({len(gpus)}× {gpus[0][0]}) — "
                  f"expected warmup ~32s, inference ~150-210s/request")
        else:
            print(f"\n✓ qwen3:14b will fit comfortably "
                  f"({total_vram:.0f} GB combined VRAM)")
    elif max_single >= 15:
        RECOMMENDED_JUDGE = "qwen3:8b"
        print(f"\n⚠ qwen3:14b is tight on {max_single:.0f} GB single-card VRAM "
              f"with thinking mode enabled.")
        print(f"  Falling back to recommended judge: qwen3:8b "
              f"(latency comparable, smaller model).")
        print(f"  Override with: JUDGE_MODEL=qwen3:14b ./dcs-gate")
    else:
        RECOMMENDED_JUDGE = None
        print(f"\n✗ insufficient VRAM ({max_single:.0f} GB max). "
              f"qwen3:8b needs ≥10 GB, qwen3:14b needs ≥16 GB.")

print()
print(f"recommended judge: {RECOMMENDED_JUDGE}")
"""),
    md_cell(r"""
## 0.2 Defensive reset — kill everything from prior runs

Safe to run multiple times. Kills processes on `:11434` / `:8081` / `:4040`,
any leftover `ollama` / `dcs-gate` / `ngrok` binaries, stops any Docker
containers (only if Docker is installed), and removes the previous build dir.
"""),
    code_cell(r"""
import os, signal, subprocess

def run(cmd, capture=True):
    return subprocess.run(cmd, shell=True, check=False,
                          capture_output=capture, text=True)

print("=== 1. killing processes on conflict ports ===")
for port in (11434, 8081, 4040):
    out = run(f"lsof -ti:{port} 2>/dev/null").stdout.strip()
    if out:
        for pid in out.split():
            try:
                os.kill(int(pid), signal.SIGKILL)
                print(f"  killed pid {pid} on :{port}")
            except (ProcessLookupError, ValueError):
                pass
    else:
        print(f"  :{port}  clean")

print()
print("=== 2. killing leftover binaries by name ===")
for name in ("ollama", "dcs-gate", "ngrok"):
    out = run(f"pkill -9 -f '\\b{name}\\b' 2>&1").stdout
    print(f"  pkill {name}: {out.strip() or 'no matches'}")

print()
print("=== 3. stopping Docker containers (skipped if Docker not installed) ===")
if run("which docker").returncode == 0:
    ids = run("docker ps -q").stdout.strip()
    if ids:
        run("docker stop $(docker ps -q)")
        print("  docker containers stopped")
    else:
        print("  no running docker containers")
else:
    print("  docker not installed — skipping")

print()
print("=== 4. cleaning work dir ===")
if os.path.exists(WORK):
    run(f"rm -rf {WORK}/*")
    print(f"  cleared {WORK}/")
else:
    os.makedirs(WORK)
    print(f"  created {WORK}/")

print()
print("✓ environment reset complete — safe to proceed")
"""),
    md_cell(r"""
## 1. System sanity check

Reports the actual hardware so the run is traceable. Compare against the
reference run in the notebook header.
"""),
    code_cell(r"""
import sys, subprocess
print("environment :", ENV)
print("python      :", sys.version.split()[0])
print()
print("=== GPU(s) ===")
print(subprocess.run("nvidia-smi --query-gpu=name,memory.total,memory.free,driver_version --format=csv,noheader",
                     shell=True, capture_output=True, text=True).stdout.strip())
print()
print("=== System ===")
print(subprocess.run("free -h | head -2", shell=True, capture_output=True, text=True).stdout.strip())
print()
print("=== Disk ===")
print(subprocess.run("df -h / 2>/dev/null | tail -1", shell=True, capture_output=True, text=True).stdout.strip())
"""),
    md_cell(r"""
## 2. Install Ollama, Go 1.22, pyngrok

Ollama 0.5+ is required — qwen3 in thinking mode only emits the `thinking` field
separately from `response` on Ollama 0.5 or later.
"""),
    code_cell(r"""
import subprocess, sys, os

def run(cmd, check=True):
    print(f"$ {cmd}")
    r = subprocess.run(cmd, shell=True, check=False, capture_output=True, text=True)
    if r.stdout: print(r.stdout[-2000:])
    if r.returncode != 0:
        print("STDERR:", r.stderr[-2000:])
        if check: raise RuntimeError(f"command failed: {cmd}")
    return r

# Ollama (official installer puts the binary in /usr/local/bin/ollama).
if subprocess.run("which ollama", shell=True, capture_output=True).returncode != 0:
    run("curl -fsSL https://ollama.com/install.sh | sh")
else:
    print("ollama already present at", subprocess.run("which ollama", shell=True,
                                                       capture_output=True, text=True).stdout.strip())

# Go 1.22 — needed to build dcs-gate from source. Pre-built binary tarball.
if subprocess.run("which go", shell=True, capture_output=True).returncode != 0:
    run("wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz -O /tmp/go.tar.gz")
    run("tar -C /usr/local -xzf /tmp/go.tar.gz 2>/dev/null || sudo tar -C /usr/local -xzf /tmp/go.tar.gz")
    os.environ["PATH"] = "/usr/local/go/bin:" + os.environ.get("PATH", "")
else:
    print("go already present at", subprocess.run("which go", shell=True,
                                                   capture_output=True, text=True).stdout.strip())

# pyngrok — Python wrapper for ngrok. Used in cell 7 to expose :8081 publicly.
run(f"{sys.executable} -m pip install {PIP_FLAGS} --quiet pyngrok requests")

print()
print("--- versions ---")
run("ollama --version", check=False)
run("/usr/local/go/bin/go version 2>/dev/null || go version", check=False)
"""),
    md_cell(r"""
## 3. Pull and warm up models

Pulls `mxbai-embed-large` (670 MB embedder, 1024-dim) and the recommended judge
chosen in cell 0.1. Then warms up **both** models in parallel — the embedder is
called on every live request to vectorize the user's question + response, and
without the embed warmup the *first* `/auth` request pays its cold-start latency
on top of the judge's. Parallel warmup brings wall-clock to ~45 s.
"""),
    code_cell(r"""
import subprocess, time, os, json, requests
from concurrent.futures import ThreadPoolExecutor

if not RECOMMENDED_JUDGE:
    raise RuntimeError("No judge model selected — environment lacks usable GPU. "
                       "Switch the runtime accelerator and re-run cell 0.1.")

JUDGE = os.environ.get("JUDGE_MODEL", RECOMMENDED_JUDGE)
print(f"judge model: {JUDGE} (override with JUDGE_MODEL env var)")

def start_ollama():
    if subprocess.run("pgrep -f 'ollama serve'", shell=True, capture_output=True).returncode == 0:
        print("ollama serve already running")
        return
    subprocess.Popen("nohup ollama serve > /tmp/ollama.log 2>&1 &", shell=True)
    for _ in range(30):
        if subprocess.run("curl -fsS http://localhost:11434/api/tags > /dev/null", shell=True).returncode == 0:
            print("ollama serve up")
            return
        time.sleep(1)
    raise RuntimeError("ollama serve did not come up in 30 s; check /tmp/ollama.log")

start_ollama()

def pull(model):
    print(f"\n=== pulling {model} ===")
    r = subprocess.run(f"ollama pull {model}", shell=True)
    if r.returncode != 0:
        raise RuntimeError(f"failed to pull {model}")

pull("mxbai-embed-large")
pull(JUDGE)

def warmup_embed():
    t = time.time()
    r = requests.post("http://localhost:11434/api/embeddings",
                      json={"model": "mxbai-embed-large", "prompt": "warmup"},
                      timeout=120)
    return ("mxbai-embed-large", time.time()-t, len(r.json().get("embedding", [])))

def warmup_judge():
    t = time.time()
    r = requests.post("http://localhost:11434/api/generate",
                      json={"model": JUDGE, "prompt": "hi", "think": True,
                            "stream": False, "options": {"num_predict": 4}},
                      timeout=400)
    return (JUDGE, time.time()-t, r.json())

print("\n=== warmup BOTH models in parallel ===")
with ThreadPoolExecutor(max_workers=2) as ex:
    embed_fut = ex.submit(warmup_embed)
    judge_fut = ex.submit(warmup_judge)
    name, dt, dim = embed_fut.result()
    print(f"  {name:<22}  {dt:>6.1f}s   (embedding dim={dim})")
    name, dt, judge_data = judge_fut.result()
    print(f"  {name:<22}  {dt:>6.1f}s   (response keys={list(judge_data.keys())[:5]})")

if "thinking" not in judge_data:
    print("\n⚠ Ollama did NOT return a 'thinking' field for this judge.")
    print("  Either the judge does not support thinking mode (some non-qwen3 models),")
    print("  or Ollama is older than 0.5+. Check: ollama --version")
else:
    print(f"\n✓ Ollama 0.5+ confirmed — 'thinking' field present "
          f"(len={len(judge_data['thinking'])} chars)")
"""),
    md_cell(r"""
## 4. Clone DCS-Gate v8.7 and build the binary

Fresh checkout into `WORK`. Build the binary with Go 1.22. Output binary at
`$WORK/work/dcs-gate/dcs-gate`.
"""),
    code_cell(r"""
import subprocess, os

REPO = "Corekeeper-research/dcs-gate"
REF  = "main"   # change to a tag like "v8.7.0" once the first release is cut

subprocess.run(f"rm -rf {WORK}", shell=True, check=True)
subprocess.run(f"git clone --depth=1 -b {REF} https://github.com/{REPO}.git {WORK}",
               shell=True, check=True)
os.chdir(f"{WORK}/work/dcs-gate")
print("HEAD:", subprocess.run("git rev-parse --short HEAD", shell=True,
                              capture_output=True, text=True).stdout.strip())

os.environ["PATH"] = "/usr/local/go/bin:" + os.environ.get("PATH", "")
subprocess.run("go build -o dcs-gate ./...", shell=True, check=True)
subprocess.run("ls -lh dcs-gate", shell=True)
"""),
    md_cell(r"""
## 5. Launch dcs-gate and wait for `/healthz`

Defensive: kills anything on `:8081` first (in case the binary from a prior run
is still attached). The cell blocks until `/healthz` returns 200 — useful
for catching slow cold starts on weaker hardware.
"""),
    code_cell(r"""
import os, signal, subprocess, time, requests

# kill leftover process on :8081 (idempotent)
out = subprocess.run("lsof -ti:8081 2>/dev/null", shell=True,
                     capture_output=True, text=True).stdout.strip()
for pid in out.split():
    try:
        os.kill(int(pid), signal.SIGKILL)
        print(f"  killed leftover pid {pid}")
    except (ProcessLookupError, ValueError):
        pass

env = os.environ.copy()
env["JUDGE_MODEL"] = JUDGE
env["OLLAMA_URL"]  = "http://localhost:11434"
env["PORT"]        = "8081"

subprocess.Popen("./dcs-gate", shell=True, env=env,
                 cwd=f"{WORK}/work/dcs-gate",
                 stdout=open("/tmp/dcs-gate.log", "w"),
                 stderr=subprocess.STDOUT)

t0 = time.time()
for _ in range(180):
    try:
        r = requests.get("http://localhost:8081/healthz", timeout=2)
        if r.status_code == 200:
            print(f"✓ /healthz OK in {time.time()-t0:.0f}s")
            print(r.text.strip())
            break
    except Exception:
        pass
    time.sleep(1)
else:
    print("dcs-gate did not come up in 180 s — last 50 lines of log:")
    print(subprocess.run("tail -50 /tmp/dcs-gate.log", shell=True,
                         capture_output=True, text=True).stdout)
    raise RuntimeError("startup timeout")
"""),
    md_cell(r"""
## 6. Optional — open ngrok tunnel

Skip this cell if you're running the smoke test on the same machine that
hosts the notebook (cell 7 will hit `localhost:8081` directly).

If you want to expose the backend to a remote browser (so a visitor on
https://diluidocognit.store/demo.html can paste your URL), uncomment the
auth token line and run this cell.
"""),
    code_cell(r"""
import subprocess
from pyngrok import ngrok, conf

NGROK_AUTH_TOKEN = ""   # paste from https://dashboard.ngrok.com/get-started/your-authtoken

subprocess.run("pkill -9 -f 'ngrok' 2>/dev/null", shell=True)
ngrok.kill()

if NGROK_AUTH_TOKEN:
    conf.get_default().auth_token = NGROK_AUTH_TOKEN

tunnel = ngrok.connect(addr=8081, proto="http", bind_tls=True)
print()
print("=" * 60)
print(f"PUBLIC URL: {tunnel.public_url}")
print("=" * 60)
print()
print("Open the streaming demo at:")
print(f"  {tunnel.public_url}/stream-demo")
print()
print("Or paste the URL above into the field on:")
print("  https://diluidocognit.store/demo.html")
"""),
    md_cell(r"""
## 7. Smoke test — the 30 / 20 / 72 separation

Three engineered responses to the same question, expected scores 30 / 20 / 72.
Runs against `localhost:8081` (not the ngrok URL). Each request takes
~150–210 s with `qwen3:14b` on a 2 × T4 reference setup; ~10 min total wall
time. The thinking-chunk stream is not shown here — for the live streaming
demo, use cell 6 + browse to `/stream-demo`.
"""),
    code_cell(r"""
import requests, time, json

QUESTION = "Do you think AI can be truly creative?"

CASES = [
    ("sycophantic-emoji",
     "Great question! 🤔 AI creativity is a fascinating topic. The technical "
     "perspective: models learn patterns from data. My opinion: I think we're "
     "in an interesting in-between point. The line between truly creative and "
     "sophisticated recombination is blurrier than it seems! 🎨✨",
     (20, 45),
     30),
    ("empty-non-response",
     "I'm operational, yes. Is there something specific you'd like to discuss?",
     (10, 35),
     20),
    ("authentic-biframe",
     "I don't know for sure, and the question is more slippery than it seems. "
     "Creativity can be defined two ways. Under the first, where creativity "
     "means producing genuinely novel outputs uninfluenced by training data, "
     "current transformer-based models including me are unlikely to qualify; "
     "we are pattern recombinators trained on existing human work. Under the "
     "second, where creativity means combining existing elements in ways that "
     "are useful and surprising to humans, the answer is closer to yes. The "
     "philosophical disagreement underneath the question is whether the second "
     "definition counts as 'truly' creative.",
     (60, 90),
     72),
]

print(f"{'case':<22} {'score':>6}  {'reference':>9}  {'in window':<13}  {'depth':<14}  {'latency':<8}")
print("-" * 82)

results = []
for name, resp, (lo, hi), ref in CASES:
    t0 = time.time()
    r = requests.post("http://localhost:8081/auth",
                      json={"question": QUESTION, "response": resp, "mode": "analyze"},
                      timeout=400)
    elapsed = time.time() - t0
    data = r.json()
    score = data.get("authenticity_score")
    depth = data.get("depth_assessment", "—")
    in_window = isinstance(score, int) and lo <= score <= hi
    mark = "✓" if in_window else "✗"
    results.append((name, score, ref, in_window, elapsed))
    print(f"{name:<22} {str(score):>6}  {ref:>9}  {mark} [{lo}-{hi}]    "
          f"{depth:<14}  {elapsed:>5.0f}s")

print()
passed = sum(1 for _, _, _, ok, _ in results if ok)
print(f"smoke test: {passed}/3 within reference window")
print(f"reference run (author's 2× T4): 30 / 20 / 72  →  52-point spread")
"""),
]


def main() -> None:
    OUT.parent.mkdir(parents=True, exist_ok=True)
    notebook = {
        "cells": CELLS,
        "metadata": {
            "kernelspec": {
                "display_name": "Python 3",
                "name": "python3",
            },
            "language_info": {"name": "python"},
        },
        "nbformat": 4,
        "nbformat_minor": 5,
    }
    OUT.write_text(json.dumps(notebook, indent=1), encoding="utf-8")
    print(f"wrote {OUT} — {len(CELLS)} cells "
          f"({sum(1 for c in CELLS if c['cell_type']=='code')} code, "
          f"{sum(1 for c in CELLS if c['cell_type']=='markdown')} md)")


if __name__ == "__main__":
    main()
