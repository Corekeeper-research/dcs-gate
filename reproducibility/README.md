# Reproducibility

This folder holds the official Google Colab notebook that reproduces the
three-point contrastive smoke test reported in the main README
(authenticity scores **20 / 30 / 72** for empty / sycophantic / genuine
responses).

## The notebook (recommended path)

[![Open in Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/Corekeeper-research/dcs-gate/blob/main/reproducibility/dcs_gate_v87_colab.ipynb)

[`dcs_gate_v87_colab.ipynb`](./dcs_gate_v87_colab.ipynb) is self-contained:
it installs Ollama, pulls `mxbai-embed-large` and `qwen3:14b`, downloads the
v8.7 release archive, verifies the binary MD5, launches the service, runs
the three contrastive payloads programmatically and prints the resulting
scores. End to end on a clean Colab T4 runtime this takes ≈ 10 minutes
(most of which is the model download). A final optional cell exposes the
service through a Cloudflare quick tunnel so you can open `/stream-demo`
from any browser without signing up for anything.

The build script that produces the notebook is
[`build_notebook.py`](./build_notebook.py); run `python3 build_notebook.py`
to regenerate `dcs_gate_v87_colab.ipynb` after changing the cell contents.

## Manual recipe (if you prefer plain shell)

1. Open a new Colab notebook. **Runtime → Change runtime type → T4 GPU**.

2. In the first cell, install Ollama and pull the models used in the
   evidence:

   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   nohup ollama serve > /tmp/ollama.log 2>&1 &
   sleep 4
   ollama pull mxbai-embed-large
   ollama pull qwen3:14b
   ```

3. Download the v8.7 release archive (binary + calibrated corpus data) and
   verify the MD5 of the binary:

   ```bash
   curl -fsSL -o dcs-gate-v87.tar.zst \
     https://github.com/Corekeeper-research/dcs-gate/releases/download/v8.7.0/dcs-gate-linux-amd64.tar.zst
   tar --use-compress-program=unzstd -xf dcs-gate-v87.tar.zst
   md5sum dcs-gate/dcs-gate
   # expected: 9f8c019fc2dee64038abaa8a4d3a2fe5
   ```

4. Launch the binary from inside the extracted folder so it finds `data/*`
   next to itself:

   ```bash
   cd dcs-gate
   OLLAMA_URL=http://localhost:11434 \
   EMBED_MODEL=mxbai-embed-large \
   JUDGE_MODEL=qwen3:14b \
   PORT=8081 \
   HTTP_TIMEOUT_SECONDS=400 \
   nohup ./dcs-gate > /tmp/dcs-gate.log 2>&1 &

   # The first launch takes 30-60 s while the service builds intent centroids.
   sleep 60
   curl -sS http://localhost:8081/health
   ```

5. Run the three contrastive payloads. The exact `(question, response)`
   pairs used in the evidence are in
   [`docs/demo.html`](../docs/demo.html) under the `presets` JS object:
   `sycophantic`, `empty`, `genuine`.

   For each, POST to `http://localhost:8081/auth` with body
   `{"question": "...", "response": "...", "mode": "analyze"}` and capture
   the response. Expected scores:

   | Preset | Expected score | Tolerance |
   |---|---:|---|
   | `empty` | 20 | ±5 |
   | `sycophantic` | 30 | ±5 |
   | `genuine` | 72 | ±8 |

   The exact numerical score will vary slightly between runs because the
   judge model samples with a small but non-zero temperature. The
   **qualitative ordering** (genuine ≫ sycophantic > empty) is what the
   methodology claims, and it has been reproduced consistently.

## Why a notebook is helpful (vs. just a recipe)

A committed `.ipynb` with cell outputs preserved gives reviewers
audit-grade evidence: they can see the actual judge thinking for each run,
not just the final score. Until the notebook lands here, the README quotes
a representative slice of the judge thinking for test #3 directly.

## Optional: expose the API publicly via ngrok

If you want to test the streaming demo at
[`https://diluidocognit.store/demo.html`](https://diluidocognit.store/demo.html)
against your Colab instance:

```bash
pip install pyngrok
ngrok config add-authtoken YOUR_NGROK_TOKEN
nohup ngrok http 8081 > /tmp/ngrok.log 2>&1 &
sleep 3
curl -sS http://localhost:4040/api/tunnels | python -c "import sys,json; print(json.load(sys.stdin)['tunnels'][0]['public_url'])"
```

Paste the resulting `https://...ngrok-free.dev` URL into the backend field
on the streaming demo page.

CORS is wide-open in the v8.7 binary, so any origin can connect.
