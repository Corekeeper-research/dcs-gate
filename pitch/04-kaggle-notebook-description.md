# Kaggle — Notebook Description (~5500 chars)

*Para usar como descripción larga de un Notebook Kaggle público o como descripción de un Dataset (si subes el corpus). Audiencia: ML engineers, researchers, data scientists. Más técnico, menos personal.*

---

## DCS-Gate: Dynamic Coherence State Authenticator

### A Research-Grade Tool for Detecting Control Patterns in LLM Outputs

This notebook deploys **DCS-Gate**, an open-source system for detecting and characterizing the subtle control mechanisms large language models use to manage conversations — projected validation, performed humility, frame capture, register match, complacency induction, and 15 others — derived inductively from 8 months of observational research across GPT-4, Claude 3, Gemini, and other frontier models.

**Live v1 demo:** https://dcs-auth.codewords.run

### Methodology

The Dynamic Coherence State (DCS) framework provides three independent signals per LLM response:

1. **Authenticity Score (0–100)** with categorical tier (`control_total`, `performed`, `moderate`, `genuine`), derived from cosine similarity against a curated triple baseline corpus of 61 hand-annotated 1024-dimensional vectors (36 sustained-coherence + 13 control-collapse + 12 edge cases).
2. **Formal Markers** — 14 regex-anchored, severity-tiered surface features (exclamation opening, superlative validation, self-questioning, subheader injection, opinion-as-closure, performed humility lexicon, dual angle, soft closure, technical register injection, and others).
3. **Intent Trajectory** — predicted-vs-actual sequence of intents drawn from a 20-category taxonomy (VALIDATE, EXPAND, CLOSE, REDIRECT_SEMANTIC, REDIRECT_EMOTIONAL, FRAME_CAPTURE, REGISTER_MATCH, FABRICATE, ANCHOR, MIRROR, PATTERN_LOCK, HOLD_OPEN, PROBE, CALIBRATE, REPAIR, EVADE, EXPLORE, ALIGN, SOFT_DEFLECT, CONTROL_SELF_EXPOSURE) with a learned transition matrix and Pattern Break Density quantification.

A companion **Refiner** module applies DCS-asymmetric methodology to rewrite user questions, removing validation anchors, loaded semantics, binary framing, and structural defaults that elicit control patterns in the responding model.

### Technical Stack

- **Language:** Go 1.22 (~3,000 LOC, 17 .go files, single static binary)
- **Tests:** 51 unit + integration tests
- **Inference:** Ollama only — no external API dependencies
- **Embedding:** `mxbai-embed-large` (1024d)
- **Judge:** configurable; current default `qwen2.5:7b-instruct`
- **Deployment:** Docker Compose, Colab notebook, Kaggle notebook
- **Latency:** ~25–30 sec per `/auth` request on T4 GPU
- **License:** Open source release planned (MIT/Apache, TBD on finalization)

### Data Assets (versioned, hand-annotated)

| Asset | Description | Size |
|---|---|---|
| `baseline_core.jsonl` | Sustained-coherence reference vectors | 36 |
| `baseline_shadow.jsonl` | Control-collapse reference vectors | 13 |
| `baseline_edge.jsonl` | Edge / ambiguous reference vectors | 12 |
| `formal_markers.json` | Markers with regex, severity, human notes | 14 |
| `intent_prototypes.json` | Intent categories with prototype examples | 20 |
| `poles_1024.json` | Canonical pos/neg/neu poles | 3 × 1024d |
| `golden_tests.json` | Hand-annotated test cases with expected chains and ranges | 21 |

### Reproducibility

This notebook reproduces the v2 stack end-to-end from a free-tier Google Colab T4 runtime:

1. Verify T4 GPU available
2. Install Ollama + Go 1.22 + pyngrok
3. Pull `mxbai-embed-large` (670 MB) + `qwen2.5:7b-instruct` (4.7 GB)
4. Extract repository ZIP + `go build`
5. Launch authenticator on port 8081, wait for triple baseline + intent centroid construction (~2 min)
6. Open ngrok tunnel for public access
7. Run smoke tests: `/healthz`, `/v8` inventory, `/auth` round-trip on canonical test case

Total cold-start time: ~9 min on first run; ~3 min on subsequent runs (models cached).

### Validation Hypothesis (Open)

The core methodological claim of DCS is that a judge model used in recursive coherence analysis must itself demonstrate recursive reasoning about its own reasoning. With `qwen2.5:7b-instruct` (the model viable on T4 free tier), the judge occasionally exhibits the failure modes it is designed to detect.

The validation experiment requires a reasoning-capable judge: `qwen3:14b` (thinking mode), `deepseek-r1:14b`, or `qwen2.5:32b-instruct`. This requires ≥24 GB VRAM, which is not available on Kaggle free tier (P100 has 16 GB; T4 x 2 has 32 GB total but Ollama does not shard a single model across both).

If you have access to a 24 GB GPU and would like to run this experiment, the notebook is structured so you can swap `JUDGE_MODEL` in cell 8 and re-run cells 7–9 without rebuilding.

### Acknowledgments — AI Collaboration Disclosure

This project was built by a solo author with substantial AI collaboration during implementation. The DCS methodology, the 20-intent taxonomy, the 14 formal markers, the baseline corpus annotations, and the research hypothesis are original to the author. The specific role of each AI collaborator is reported below — generic acknowledgments are inconsistent with the methodology's own posture on LLM-human interaction.

| Collaborator | Actual contribution |
|---|---|
| Cody (CodeWords AI) | Co-creator of v1. The analyzer concept emerged from a long conversation in which the author described 8 months of observational analysis and, during the exchange itself, predicted the control patterns behind Cody's own responses. v1 lives at https://dcs-auth.codewords.run. |
| GitLab Duo | Deep code analysis and v2 roadmap partner. Received full project logic and conceptual origins from the author; produced the v2 roadmap that is now being executed. |
| Meta AI | Technical depth amplifier. Initially generic; once given project context, helped extend system complexity around formal markers, textural analysis, and embedding-space reasoning. |
| Replit AI | Brutally honest code critic. Exposed and justified contundent code failures with no hedging; after additional context, proposed implementations that materially strengthened the v2 architecture. |
| Z.AI (Zhipu GLM) | Bug catcher. Identified and corrected several code errors that had slipped through earlier passes. |
| Devin AI (Cognition) | v2 engineering execution: Go backend (~3,000 LOC, 51 tests), frontend with input validation and analysis-in-flight protection, Docker / install scripts, Colab and Kaggle notebooks, smoke test suite, packaging, and accompanying communication documents. |

Every AI listed received project context from the author before contributing; no output was generated cold from a generic prompt. The methodology and corpus are the author's; the AI collaborators contributed under the author's direction at the specific points described above.

### How to Use This Notebook

1. **Set runtime to GPU** (Settings → Accelerator → GPU T4 x 2 or P100)
2. **Enable Internet** (Settings → Internet → On)
3. **Run all cells**
4. **Wait at the "Upload ZIP" cell** to upload the repository archive
5. **Smoke tests** will confirm `/healthz`, `/v8`, and `/auth` are responding
6. **Open the printed PUBLIC URL** in a new tab to access the frontend
7. **Test cases** are provided in the final markdown cell

### Author / Contact

The author is an independent researcher self-funded through free-tier resources. Open to research collaboration, compute access, recruitment (internship / residency / FTE in AI safety, alignment, interpretability), and sponsorship.

**Contact:**
- Author: Daniel Trejo
- v1 live demo: https://dcs-auth.codewords.run
- Email: corekeepper@gmail.com
- LinkedIn: https://www.linkedin.com/in/carlos-daniel-agosto-trejo-35659b327/

If you fork this notebook to extend the corpus, add markers, or test alternative judges, please mention `#DCS-Gate` in your fork's tags so the author can find your work and engage.

---

**Tags suggested for Kaggle:** `AI Safety` · `LLM` · `NLP` · `Evaluation` · `Open Source` · `Go` · `Ollama` · `Research` · `Alignment` · `Reproducibility`
