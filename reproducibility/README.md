# Reproducibility

This folder will hold the Google Colab notebook that reproduces the
three-point contrastive smoke test reported in the main README
(authenticity scores **20 / 30 / 72** for empty / sycophantic / genuine
responses).

## Status

- Notebook: **not yet committed to the repo**. The runs that produced the
  evidence in the README were executed in an interactive Colab session and
  their outputs are quoted verbatim in the README. The standalone notebook
  is being prepared for committed reproducibility and will land here.
- Until the notebook is committed, the steps below are the manual recipe to
  reproduce the runs.

## Manual reproduction recipe (Google Colab T4, free tier)

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

3. Download the v8.7 binary and verify the MD5:

   ```bash
   curl -fsSL https://tmpfiles.org/dl/wOw4wf3D9mTv/dcs-gate-v87.bin -o dcs-gate
   chmod +x dcs-gate
   md5sum dcs-gate
   # expected: bfa6c0a2cee42fad5954dfaaa4992aeb
   ```

   (When a GitHub Release is published for v8.7, prefer the Release asset
   URL over the tmpfiles link.)

4. Launch the binary in the background:

   ```bash
   OLLAMA_URL=http://localhost:11434 \
   EMBED_MODEL=mxbai-embed-large \
   JUDGE_MODEL=qwen3:14b \
   PORT=8081 \
   HTTP_TIMEOUT_SECONDS=400 \
   nohup ./dcs-gate > /tmp/dcs-gate.log 2>&1 &

   sleep 3
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
