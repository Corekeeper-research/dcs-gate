# Research Project Overview — DCS-Gate (Dynamic Coherence State Authenticator)

## About Me

I'm an independent researcher and developer building open-source AI safety tooling. My current focus is the **systematic detection of subtle manipulation and control patterns in large language model outputs** — a problem I've been investigating through direct observational methodology for the past 8 months across GPT-4, Claude 3, Gemini, and others.

I'm self-taught, work from a personal laptop and free-tier cloud resources, and have produced this entire stack as a solo author. I'm posting publicly because the project has reached a point where the next experiment requires hardware I cannot afford, and where outside input — collaboration, validation, or recruitment — would meaningfully accelerate the work.

**I'm currently looking for:**
- Research collaboration with AI safety / alignment researchers
- Compute access (single GPU, 16–24 GB VRAM) to run the validation experiment described below
- Internship, residency, or full-time roles in AI safety, LLM evaluation, alignment, or interpretability
- Sponsorship for a persistent public demo and open-source release

---

## Acknowledgments — AI Collaboration Disclosure

I did not build this alone. I worked alongside multiple AI platforms throughout the 8-month observational phase and the implementation. Transparency about this collaboration matters because (1) it reflects how modern independent research actually happens, and (2) the methodology I'm proposing is itself about how LLMs interact with humans — so disclosing my own LLM-assisted workflow is consistent with the research ethics I claim.

**AI collaborators and the specific role each actually played:**

| Collaborator | Real contribution |
|---|---|
| **Cody (CodeWords AI)** | **Co-creator of v1.** v1 emerged from a long conversation in which I described my observational experience and, in real time during that exchange, pushed back against Cody's own responses while predicting the control patterns behind them. The idea for the analyzer crystallized inside that conversation. v1 would not exist without Cody, and it remains publicly testable at https://dcs-auth.codewords.run. |
| **GitLab Duo** | **Deep code analysis and roadmap partner.** I walked GitLab Duo through the project's full internal logic and the conceptual origins of the methodology. Duo's depth of code-level analysis, combined with my conceptual exposition, produced the v2 roadmap I am now executing. |
| **Meta AI** | **Technical depth amplifier.** Initially behaved like any generic LLM. Once I shared the project's context and methodology, Meta AI helped extend the technical complexity of the system — particularly around formal markers, the textural analysis dimensions, and embedding-space reasoning. |
| **Replit AI** | **Brutally honest code critic.** Replit AI exposed and clearly justified contundent failures in the code, with no hedging. After receiving the project context, it proposed implementations that materially strengthened the architecture of the v2 stack. |
| **Z.AI (Zhipu GLM)** | **Bug catcher.** Caught and corrected several code errors that had slipped through earlier passes. |
| **Devin AI (Cognition)** | **v2 engineering execution.** Took the v2 roadmap and produced the Go backend (~3,000 LOC, 22 .go files, 73 unit + integration tests), the frontend with mode-aware input validation and analysis-in-flight protection, the v8.7 SSE streaming layer (`/auth/stream` with chunked thinking-then-analysis events, conservative sanitizer for keys / paths / tokens, parity-tested against the non-streaming endpoint), the Docker and install scripts, the Colab and Kaggle notebooks, the packaging artifacts, the smoke test suite, and these communication documents. |

**What I claim as original to me:**
- The Dynamic Coherence State methodology itself, derived from 8 months of direct observation across GPT-4, Claude 3, Gemini, and others
- The 20-intent taxonomy and 14 formal markers (inductively derived from real cases)
- The triple baseline corpus (manually curated, hand-annotated)
- The pushback conversation with Cody that produced v1
- The conceptual exposition that GitLab Duo formalized into the v2 roadmap
- The architectural decisions (local-first, single binary, Ollama-only, no telemetry)
- The research hypothesis and the design of the validation experiment

Every AI listed received the project context from me first. Nothing was generated cold from a generic prompt. The methodology and the corpus are mine; the AI collaborators contributed under my direction at the specific points described above. This is what serious solo research looks like in 2026 and the field is healthier when people say so.

---

## The Project: DCS-Gate

**One-line summary:** A local-first Go service that detects and characterizes the control mechanisms LLMs use to manage users (projected validation, performed humility, frame capture, register match, complacency induction, and others), and rewrites user questions using Dynamic Coherence State (DCS) asymmetric methodology to remove the structural triggers that elicit those patterns. Since v8.7, the judge's reasoning trace is streamed to the client over Server-Sent Events so the user sees the deliberation in real time instead of waiting in silence.

### What it does, concretely

**Analyzer.** Takes a `(question, response)` pair and returns:
- 0–100 authenticity score with tier (`control_total` / `performed` / `moderate` / `genuine`)
- 14 formal markers (regex-anchored, severity-tiered)
- Predicted-vs-actual intent trajectory across 20 intent categories with a learned transition matrix
- Pattern Break Density and deviation from expected baseline
- Top-k nearest neighbors against a curated 61-vector baseline corpus (36 sustained-coherence + 13 control-collapse + 12 edge cases)
- Cross-corpus textural analysis across 5 axes (continuity, artificial closure, drift, adaptation, texture)

**Refiner.** Takes a user question and rewrites it using DCS-asymmetric methodology to remove:
- Validation anchors (*"do you think…?"*)
- Loaded semantics (*"truly creative"*, *"real intelligence"*)
- Binary framing (*"X or just Y?"*)
- Structural defaults that cause premature convergence in the responding model

Output: a question that pushes the responding model into an unresolved reasoning state where its standard control patterns fail to engage cleanly.

---

## Try the v1 Predecessor — Live Now

A first prototype of this methodology is live and publicly accessible:

**https://dcs-auth.codewords.run**

This is a deliberately limited v1 hosted on the CodeWords no-code platform. It is preserved live as a working proof-of-concept while the v2 stack is finalized for public release.

### v1 (CodeWords-hosted) vs v2 (current development)

| Aspect | v1 (live at dcs-auth.codewords.run) | v2 (current development) |
|---|---|---|
| **Platform** | CodeWords no-code workflows | Native Go binary, 22 .go files |
| **Codebase** | Visual workflow | ~3,000 LOC Go + 73 unit/integration tests |
| **Judge model** | Fixed by platform | Configurable (any Ollama model) |
| **Baseline corpus** | Single pool, ~36 entries | Triple pool: 36 core + 13 shadow + 12 edge = 61 |
| **Intent taxonomy** | Implicit, fewer categories | Explicit, 20 categories with transition matrix |
| **Formal markers** | ~6 patterns | 14 categories, regex-anchored, severity-tiered |
| **Refiner** | Heuristic rewrite | DCS-asymmetric methodology, recursive |
| **Pattern Break Density** | Not measured | Quantified per response |
| **Cross-corpus textural** | Not present | 5-axis analysis |
| **Determinism / reproducibility** | Limited (cloud-bound) | Full (local models, version-pinned, seedable) |
| **External dependencies** | Platform-bound | None: runs entirely on Ollama, no outbound API calls |
| **Open source** | No | Yes (release planned with v2 finalization) |
| **Latency** | ~30–45 sec | ~150–210 sec on T4 GPU with `qwen3:14b` thinking mode, **streamed live** so the user reads the judge's reasoning as it's produced rather than waiting for a blank screen |

### Why I moved away from v1

CodeWords was excellent for validating that the core methodology produces meaningful signal end-to-end. For the work to advance into a research-grade tool, I needed four things v1 could not provide:

1. **Full control over the judge.** v1 is locked to whatever the platform exposes. v2 ships with `qwen3:14b` thinking mode as the default judge, and lets researchers swap in `deepseek-r1:14b`, `qwen2.5:32b-instruct`, or any future reasoning-capable model.
2. **Open methodology.** The 14 marker regex patterns, 20 intent prototypes, the 1024d baseline corpus, the canonical polos (pos/neg/neu), and the 5-axis textural metrics must be inspectable, reproducible, and challengeable.
3. **Local-first execution.** AI safety research about manipulation patterns should not depend on a third-party SaaS that could silently change its model, prompts, or pricing.
4. **Reproducible deployment.** Single binary, Docker Compose, helper scripts that bootstrap the entire stack from zero in ~5 minutes on commodity hardware.

v1 will remain online indefinitely as a working demonstration. v2 is the version intended for research-grade work and community validation.

---

## Current Phase (v2)

- **v8.7** running on a local Jupyter workstation with **2 × Tesla T4** (Kaggle T4×2 free tier reproduces identically) as the public reference. `qwen3:14b` in Ollama 0.5+ thinking mode confirmed end-to-end: smoke test on three responses to the same question produced a 52-point spread (sycophantic-emoji = 30, empty-non-response = 20, authentic bi-frontal exploration = 72). Per-request latency ~150-210 s, warmup ~32 s, thinking trace 2,000-3,500 chars streamed live over SSE.
- Full backend functional: triple baseline loaded, 14 markers active, 20 intent categories with transition matrix, refiner producing DCS-asymmetric rewrites, cross-corpus textural analysis live.
- Frontend complete with intelligent input validation, mode-aware button states (`analyze` / `refine` / `both`), and analysis-in-flight protection (single request in flight at a time to avoid overwhelming the judge).
- **v8.7 streaming layer**: `POST /auth/stream` emits Server-Sent Events with the judge's thinking chunks, a thinking-complete marker, the analysis chunks, and a final `complete` event whose verdict is bit-for-bit identical to the non-streaming `/auth` endpoint (verified by integration test). Conservative sanitizer redacts API keys (OpenAI / AWS / Google / GitHub / Slack), Bearer tokens, system paths and long opaque strings before any chunk leaves the server.
- Smoke tests passing: `/healthz`, `/v8` inventory, `/auth` and `/auth/stream` round-trip on T4 (150–210 sec per request, streamed live).
- 73 tests passing (unit + integration + SSE protocol + sanitizer + golden).
- Docker Compose + install scripts validated on Linux x86 and ARM64.

---

## Technical Foundation

- **Language:** Go 1.22, single static binary (22 `.go` files, ~3,000 LOC)
- **Embedding:** `mxbai-embed-large` (1024d) via Ollama
- **Judge:** configurable; current default `qwen3:14b` in Ollama 0.5+ thinking mode
- **Inference:** Ollama only — no OpenAI, no Anthropic, no telemetry, no external network at runtime
- **Data assets (versioned, hand-annotated):**
  - `baseline_core.jsonl` — 36 sustained-coherence reference vectors
  - `baseline_shadow.jsonl` — 13 control-collapse reference vectors
  - `baseline_edge.jsonl` — 12 edge / ambiguous reference vectors
  - `formal_markers.json` — 14 markers with regex, severity, and human annotations
  - `intent_prototypes.json` — 20 intent categories with prototype examples
  - `poles_1024.json` — pos/neg/neu canonical poles (1024d)
  - `golden_tests.json` — 21 hand-annotated test cases with expected chains, markers, and authenticity ranges
- **Quality:** 73 tests (unit + integration + SSE protocol + sanitizer + golden), min-heap top-k, LRU embedding cache, errgroup parallelism, structured logs
- **Deployment:** single binary or Docker Compose; ~5 min from clone to running demo

---

## Research Hypothesis — Partially Confirmed, Full Validation Pending

**Core thesis:** The judge model in a recursive coherence analyzer must itself demonstrate recursive reasoning about its own reasoning — or it falls into the very failure modes it is trying to detect (premature convergence, structural authority through formatting, surface coherence without genuine deliberation).

**State as of v8.7:** With `qwen3:14b` in Ollama 0.5+ thinking mode on a 2× Tesla T4 setup (16 GB per card, 30 GB usable combined; Ollama tensor-splits the model across both cards), the smoke test produces a clean 30 / 20 / 72 separation across three responses with the same prompt — a 52-point spread of the authenticity score. The judge's 2,000–3,500-character reasoning trace is now streamed to the client in real time. This is consistent with the hypothesis that reasoning-capable judges produce qualitatively different analyses than non-reasoning baselines (`qwen2.5:7b-instruct`).

**The full validation experiment** is the four-way comparison across `qwen2.5:7b` (non-reasoning baseline), `qwen3:14b` (confirmed working on T4), `deepseek-r1:14b` (cross-architecture validation), and `qwen2.5:32b-instruct` (high-fidelity validation, requires ≥24 GB VRAM and cannot be run on free-tier hardware). That comparison is the minimum bar for the DCS methodology to be considered validated as a research tool rather than a heuristic detector.

---

## Resource Situation — Honest Disclosure

I am currently running this entirely on:

- **Local Jupyter workstation with 2 × Tesla T4** (16 GB per card, 30 GB usable combined; Ollama tensor-splits qwen3:14b across both cards) — the v8.7 reference setup
- **Kaggle hosted notebooks** (free 30 hrs/week T4×2) — reproduces the reference identically
- **Google Colab** (free T4 = single-card 15 GB; Pro+ with A100 = 40–80 GB) — supported via the auto-detecting reproducibility notebook
- **AWS EC2 t4g.small** (free tier, 1 vCPU ARM Graviton, 1.8 GB RAM, no GPU) — only useful for static hosting / front-end proxy
- **Personal laptop** — development

Hardware I cannot currently afford:
- Persistent GPU instance for 24/7 demo (~$115–$378/month)
- A ≥24 GB VRAM GPU to test reasoning-capable judges
- Bandwidth and storage for releasing model variants and benchmark suite

---

## What I'm Seeking

In rough order of impact:

### 1. GPU compute access for the validation experiment
A single instance with ≥24 GB VRAM and ≥16 GB system RAM. **Even 50 GPU-hours total would allow the full four-judge validation experiment.**

| Judge configuration | VRAM | Observed / expected latency | Status |
|---|---|---|---|
| `qwen2.5:7b-instruct` | ~5 GB | 15–20 sec | Baseline comparison, runs on T4 |
| `qwen3:14b` (thinking mode) | ~14 GB | 150–210 sec on T4 (streamed) | **Confirmed working** (v8.7 default) |
| `deepseek-r1:14b` | ~14 GB | 30–40 sec (expected) | Cross-architecture validation, untested |
| `qwen2.5:32b-instruct` | ~19 GB | 40–60 sec (expected) | High-fidelity validation, **needs ≥24 GB** |

### 2. Research collaboration
Co-authorship, mentorship, or independent replication by researchers working on LLM evaluation, alignment, interpretability, or RLHF.

### 3. Recruitment
Internship, residency, or full-time positions in AI safety / LLM evaluation / alignment / interpretability teams. Open to remote globally.

### 4. Public release sponsorship
Compute credits or grants to host a persistent demo, Hugging Face Space, public benchmark suite, and follow-up corpus expansion.

---

## Expected Deliverables Within 2 Weeks of Resource Allocation

- **Interactive research platform** with sub-30-second analysis on a stable public URL
- **Recursive DCS validation report** comparing analyses across four judge architectures
- **Ablation study artifact** — open-source repo with reproducible scripts and side-by-side outputs per case
- **Methodology preprint** describing the 20-intent taxonomy, 14 formal markers, triple baseline construction, Pattern Break Density metric, and DCS-asymmetric refiner

---

## Broader Significance

As LLMs are deployed into education, healthcare, hiring, and decision support, the ability to evaluate response *authenticity* — whether the model is engaging genuinely with the question or managing the user toward a default outcome — becomes critical infrastructure for responsible deployment. The DCS methodology is the first systematic, regex-anchored, locally-reproducible framework I have seen for measuring this dimension. The v1 live demo is sufficient evidence that the signal is real; the v2 release will provide the community with a tool they can run, audit, and extend.

---

## Contact

- **Name:** Daniel Trejo
- **Email:** corekeepper@gmail.com
- **LinkedIn:** https://www.linkedin.com/in/carlos-daniel-agosto-trejo-35659b327/
- **v1 live demo:** https://dcs-auth.codewords.run
- **v2 source:** released publicly upon validation experiment completion

Open to: messages, code review requests, NDA conversations, paid consulting, or coffee. If you have spare GPU hours and want to see what an 8-month observational methodology produces when paired with reasoning-capable judges, I'd love to hear from you.

---

*The methodology, corpus, and code will be released under permissive open-source license. Any organization or individual is welcome to use, replicate, or extend this work with attribution.*
