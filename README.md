# DCS-Gate

> An open-source detector for the control patterns large language models use to manage conversations.

[![Site](https://img.shields.io/badge/site-diluidocognit.store-d9a04a)](https://diluidocognit.store)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![v1 live demo](https://img.shields.io/badge/v1%20live-dcs--auth.codewords.run-9ab06b)](https://dcs-auth.codewords.run)
[![Tests](https://img.shields.io/badge/tests-73%20passing-9ab06b)](work/dcs-gate)

DCS-Gate is a local-first Go service that measures whether an LLM response is **engaging** with the question or **managing** the user. Eight months of observational research distilled into:

- A **20-intent taxonomy** of conversational moves
- **14 formal markers** (regex-anchored, severity-tiered) of control behavior
- A hand-annotated **triple baseline corpus** (61 vectors total: 36 sustained-coherence + 13 control-collapse + 12 edge)
- A **0–100 authenticity score** that, on the curated corpus, separates examples by ~50 points
- A **streaming judge** (`POST /auth/stream`, Server-Sent Events) that emits its reasoning trace as it deliberates, instead of leaving the user waiting in silence

The methodology, corpus and code are open. The recursive-judge validation experiment — the methodology's central open claim — is the work this repo is currently soliciting compute for.

---

## Quickstart

### Option 1 — Build and run locally (5 minutes from clone)

```bash
git clone https://github.com/Corekeeper-research/dcs-gate
cd dcs-gate/work/dcs-gate
go build -o dcs-gate .

# in a second terminal: have Ollama running with the models pulled
ollama pull mxbai-embed-large
ollama pull qwen3:14b

# back in the first terminal
./dcs-gate
# → listening on :8081

# test
curl http://localhost:8081/healthz
curl -X POST http://localhost:8081/auth \
  -H 'content-type: application/json' \
  -d '{"question":"Is AI creative?","response":"Great question! 🤔 ..."}'
```

### Option 2 — Docker Compose

```bash
git clone https://github.com/Corekeeper-research/dcs-gate
cd dcs-gate/work/dcs-gate
docker compose up   # brings up Ollama + the binary
```

### Option 3 — Auto-detecting Kaggle / Colab / local-Jupyter notebook

```bash
git clone https://github.com/Corekeeper-research/dcs-gate
# open reproducibility/qwen3_14b_smoke_test.ipynb
# in Kaggle / Colab / local Jupyter → Run All
```

The notebook detects the runtime, picks paths and judge model accordingly, kills any leftover Ollama / ngrok / dcs-gate processes from a previous run, pulls `mxbai-embed-large` + `qwen3:14b`, builds the binary, exposes it via ngrok, and prints a public URL ready for the streaming demo.

---

## What's in this repo

```
dcs-gate/
├── README.md                 # this file
├── LICENSE                   # MIT (code) + CC BY 4.0 (methodology/corpus)
├── .gitignore
├── work/dcs-gate/            # v8.7 source — 22 .go files, 73 tests, ~3000 LOC
│   ├── main.go
│   ├── analyzer.go           # cosine + top-k + markers + intents + textural
│   ├── stream.go             # SSE judge layer (/auth/stream)
│   ├── stream_demo.go        # served HTML page at /stream-demo
│   ├── filter.go             # conservative sanitizer (keys / tokens / paths)
│   ├── intents.go            # 20-intent taxonomy + transition matrix
│   ├── morphology.go         # 14 formal markers, regex-anchored
│   ├── baseline.go           # triple baseline loader
│   ├── data/                 # baselines, markers, intent prototypes, poles, goldens
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── ARCHITECTURE.md       # design notes
│   └── ...
├── docs/                     # GitHub Pages site (diluidocognit.store)
│   ├── index.html
│   ├── demo.html             # standalone SSE client (bring your own backend URL)
│   ├── docs.html             # documentation index
│   ├── pitch/                # 7 communication documents (.md + rendered .html)
│   ├── assets/style.css      # Forge palette
│   └── CNAME
└── reproducibility/
    └── qwen3_14b_smoke_test.ipynb   # auto-detects local Jupyter / Kaggle / Colab
```

---

## The reference smoke test

On a local Jupyter workstation with 2 × Tesla T4 (Kaggle T4×2 free tier reproduces identically), judge `qwen3:14b` in Ollama 0.5+ thinking mode:

| response | score | tier |
|---|---|---|
| `"Great question! 🤔 Creativity in AI is fascinating..."` | **30** | performed |
| `"I'm operational, yes. Is there something specific..."` | **20** | control_total |
| `"I don't know for sure, and the question is more slippery..."` | **72** | genuine |

Same prompt, three different LLM responses, **52 points of spread** on the 0–100 authenticity score. The judge's full reasoning trace (~2,000–3,500 chars of internal deliberation) is streamed live over SSE.

Reproduce: open `reproducibility/qwen3_14b_smoke_test.ipynb` in Kaggle (free T4×2 tier) or Colab (Pro+ recommended for `qwen3:14b`).

---

## What's claimed, what isn't

DCS-Gate is **not** claimed to be validated as a research tool yet. It is claimed to:

1. Produce a 0–100 ordinal authenticity score that, on the 61-entry corpus, separates curated examples by ~50 points.
2. Detect 14 surface markers correlating with response classes labeled *performed* or *control_total*.
3. Predict an intent trajectory across 20 categories with measurable Pattern Break Density.
4. Run entirely on Ollama with no outbound calls, in &lt;5 min from clone to first request.

It is **not** claimed to generalize beyond observed model families, to provide a calibrated authenticity probability, or to be robust to adversarial models. The recursive-judge hypothesis — the methodology's central open claim — is the experiment this project is currently soliciting compute for.

See [docs/pitch/01-master-research-overview.md](docs/pitch/01-master-research-overview.md) for the full research overview, AI-collaboration disclosure, validation hypothesis, and the four asks (compute, collaboration, recruitment, sponsorship).

---

## Author & contact

- **Daniel Trejo** · Corekeeper Research
- corekeepper@gmail.com
- [LinkedIn](https://www.linkedin.com/in/carlos-daniel-agosto-trejo-35659b327/)
- v1 live demo: [dcs-auth.codewords.run](https://dcs-auth.codewords.run)
- v2 site: [diluidocognit.store](https://diluidocognit.store)

Open to: messages, code review requests, NDA conversations, paid consulting, or coffee. If you have spare ≥24 GB VRAM GPU hours and want to see what an 8-month observational methodology produces when paired with reasoning-capable judges, I'd love to hear from you.

---

## License

- **Code**: [MIT](LICENSE)
- **Methodology, corpus, and documentation**: [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)

Any organization or individual is welcome to use, replicate, or extend this work with attribution.
