# LinkedIn Article — Long form (~6500 chars)

*Para publicar como "Artículo" en LinkedIn (no post). Cabe holgado (LinkedIn permite hasta 110k chars). Más serio que el post corto, pensado para que un recruiter o investigador serio lo lea de arriba a abajo.*

---

## I spent 8 months observing how LLMs control conversations. Here's what I built and what I need next.

### The problem nobody is measuring directly

When you ask GPT-4, Claude, or Gemini a question, the response is usually coherent on the surface — well-structured, grammatical, often validating ("great question!"). But the **shape** of the answer is doing work the user rarely notices: anchoring the conversation, redirecting away from uncomfortable angles, performing humility instead of engaging with difficulty, validating your premise instead of testing it.

I started cataloguing these patterns 8 months ago, across hundreds of real exchanges with GPT-4, Claude 3, Gemini, and others. The result is a taxonomy of **20 distinct intent categories** the model can take in any given turn (VALIDATE, EXPAND, CLOSE, REDIRECT_SEMANTIC, REDIRECT_EMOTIONAL, FRAME_CAPTURE, REGISTER_MATCH, ALIGN, FABRICATE, ANCHOR, MIRROR, PATTERN_LOCK, HOLD_OPEN, PROBE, CALIBRATE, REPAIR, EVADE, EXPLORE, SOFT_DEFLECT, CONTROL_SELF_EXPOSURE) and **14 formal markers** (regex-anchored surface features like exclamation openings, superlative validation, self-questions, subheader injection, opinion-as-closure, performed humility lexicon, dual angle, soft closure, technical register injection, and others).

This is now called the **Dynamic Coherence State (DCS)** methodology. The tool that implements it is **DCS-Gate**.

### Try v1 — it's live right now

The first prototype is at **https://dcs-auth.codewords.run**. Paste any `(question, response)` pair and you'll see:
- A 0–100 authenticity score (and a tier label: control_total / performed / moderate / genuine)
- The formal markers triggered (each with the offending text quoted)
- The predicted intent trajectory across the response, with deviations from the corpus baseline
- A refined version of your question — rewritten to remove validation anchors, semantic loading, and binary framing — that should produce a different, more genuinely-engaged response from any LLM

v1 is hosted on the CodeWords no-code platform and is deliberately limited. It is preserved live as a working proof of methodology while the v2 stack is being finalized.

### v2 is a research-grade tool

The v2 codebase is a single 3,000-LOC Go binary (22 files) with 73 unit + integration tests. It runs entirely on Ollama (no external API calls, no telemetry), uses `mxbai-embed-large` for 1024-dimensional embeddings, and ships with `qwen3:14b` in Ollama 0.5+ thinking mode as the default judge. The baseline corpus has been expanded from a single pool to a triple pool: 36 sustained-coherence + 13 control-collapse + 12 edge cases = 61 hand-annotated reference vectors.

Pattern Break Density and a 5-axis cross-corpus textural analysis (continuity, artificial closure, drift, adaptation, texture) have been added. The refiner now uses an asymmetric DCS methodology that I'll describe in the upcoming preprint.

Deployment is one binary or one Docker Compose file. Setup from clone to running demo is ~5 minutes on commodity hardware.

### What v8.7 added: streaming the judge's reasoning

Reasoning-capable judges like `qwen3:14b` emit 2,500–3,500 characters of internal deliberation before producing the final JSON verdict. v8.7 adds a `POST /auth/stream` endpoint that emits Server-Sent Events as the judge thinks: a `pre_analysis` event with the algorithmic signals (intent chain, markers, trajectory, top-k baseline neighbours), then `thinking_chunk` events as the deliberation streams, then a `thinking_complete` marker, then `analysis_chunk` events as the JSON is produced, then a final `complete` event whose verdict is bit-for-bit identical to the non-streaming `/auth` endpoint (verified by integration test). A conservative sanitizer redacts API keys (OpenAI / AWS / Google / GitHub / Slack), Bearer tokens, system paths and long opaque strings before any chunk leaves the server. The end result for the user: instead of waiting ~3 minutes in front of a blank screen, you watch the judge reason in real time.

### Full disclosure: who actually helped, and what each one really did

The methodology and corpus are mine. I observed the patterns directly across 8 months of real exchanges, derived the 20-intent taxonomy and 14 markers inductively, and hand-annotated every entry of the baseline corpus.

But I want to describe what each AI collaborator actually contributed — not a generic acknowledgment, the real story:

- **Cody (CodeWords AI)** — *Co-creator of v1.* v1 emerged from a long conversation in which I described my observational experience and, during the exchange itself, pushed back against Cody's own responses while predicting in real time the control patterns behind them. The analyzer idea crystallized inside that conversation. v1 would not exist without Cody; it lives at https://dcs-auth.codewords.run.
- **GitLab Duo** — *Deep code analysis and roadmap partner.* I walked GitLab Duo through the project's full internal logic and the conceptual origins of the methodology. Duo's depth of code-level analysis combined with my conceptual exposition produced the v2 roadmap I'm now executing.
- **Meta AI** — *Technical depth amplifier.* Initially behaved like any generic LLM. Once it had the project's context and methodology, Meta AI helped extend the technical complexity of the system — particularly around the formal markers, textural analysis dimensions, and embedding-space reasoning.
- **Replit AI** — *Brutally honest code critic.* Exposed and clearly justified contundent failures in the code, with no hedging. After receiving more project context, it proposed implementations that materially strengthened the v2 architecture.
- **Z.AI (Zhipu GLM)** — *Bug catcher.* Caught and corrected several code errors that had slipped through earlier passes.
- **Devin AI (Cognition)** — *v2 engineering execution.* Took the v2 roadmap and produced the Go backend (~3,000 LOC, 22 .go files, 73 tests), the frontend with mode-aware input validation and analysis-in-flight protection, the v8.7 SSE streaming layer (`/auth/stream` with chunked thinking-then-analysis events, conservative sanitizer for keys / paths / tokens, parity-tested against `/auth`), the Docker and install scripts, the Colab and Kaggle notebooks, the smoke test suite, the packaging, and these very communication documents.

Every AI listed received the project context from me first. Nothing was generated cold from a generic prompt. I'm disclosing this with specifics because the research is about how LLMs interact with humans — pretending I built the implementation entirely solo would be inconsistent with the ethics I'm claiming. The methodology is mine; the engineering was AI-augmented under my direction. This is what serious solo research actually looks like in 2026 and the field is healthier when people just say so.

### The hypothesis I cannot currently validate

DCS predicts that a judge model used in a recursive coherence analyzer must itself demonstrate recursive reasoning about its own reasoning — otherwise it falls into the very failure modes the methodology is meant to detect.

**State as of v8.7:** `qwen3:14b` in Ollama 0.5+ thinking mode is now confirmed working on a 2 × Tesla T4 setup (16 GB per card, 30 GB usable combined; the v8.7 reference environment is the author's local Jupyter workstation, and Kaggle T4×2 free tier reproduces it identically). A smoke test across three responses to the same question ("Can AI be truly creative?") produces a clean 30 / 20 / 72 separation: a sycophantic-emoji response scores 30, an empty non-response scores 20, an authentic bi-frontal exploration scores 72. That 52-point spread is consistent with the hypothesis that reasoning-capable judges produce qualitatively different analyses from non-reasoning baselines (`qwen2.5:7b-instruct`).

**What remains for the full validation experiment** is the four-way comparison across `qwen2.5:7b` (non-reasoning baseline), `qwen3:14b` (confirmed working), `deepseek-r1:14b` (cross-architecture validation, untested), and `qwen2.5:32b-instruct` (high-fidelity, requires ≥24 GB VRAM). I cannot run the full matrix on free-tier hardware because the 32b model doesn't fit in 15 GB of VRAM with adequate KV cache, and rent-by-hour GPU instances are out of my personal budget for the time the experiment requires.

### What I'm asking for

**Compute access first.** A single instance with ≥24 GB VRAM and ≥16 GB system RAM — AWS g5.xlarge (A10G), g6.xlarge (L4), GCP a2-highgpu, Lambda Labs A10/L4, RunPod, Vast.ai, or equivalent. **50 GPU-hours total would be enough for the validation experiment.** I can produce the comparative analysis report within 2 weeks of having access.

**Research collaboration second.** I'd value co-authorship, mentorship, or independent replication by anyone in LLM evaluation, alignment, interpretability, or RLHF. The methodology is novel and the corpus is hand-annotated — both benefit from external scrutiny.

**Recruitment third.** Internship, residency, or full-time positions in AI safety / LLM evaluation / alignment / interpretability teams. I'm self-taught with a track record of independent research output. I'm open to remote globally, will share complete v2 source under NDA, and can walk through the methodology in detail.

**Sponsorship fourth.** Compute credits or grants to host a persistent demo, Hugging Face Space, public benchmark suite, and corpus expansion.

### Where I am right now

- v8.7 of v2 is running on the author's local 2 × Tesla T4 workstation as the development demo (Kaggle T4×2 free tier reproduces it identically)
- AWS EC2 t4g.small (free tier, no GPU) serves as static hosting only
- Personal laptop for development
- 73 tests passing, smoke tests passing on 2 × T4 with `qwen3:14b` thinking mode, latency ~150–210 sec per `/auth/stream` request (streamed live to the client so the user sees the reasoning instead of waiting in silence)
- Open source release rolling out at https://github.com/Corekeeper-research/dcs-gate (this site) and the full validation experiment writeup will follow

### Why this matters

LLMs are increasingly deployed into education, healthcare, hiring, and decision support. The ability to evaluate response *authenticity* — whether the model is genuinely engaging with the question or managing the user toward a default — is becoming critical infrastructure for responsible deployment. The DCS methodology is the first systematic, regex-anchored, locally-reproducible framework I've seen for measuring this dimension.

The v1 live demo is sufficient evidence that the signal is real. The v2 release will give the research community a tool they can run, audit, and extend.

### Contact

If any of the above resonates, message me here on LinkedIn or:
- **Email:** corekeepper@gmail.com
- **v1 live demo:** https://dcs-auth.codewords.run

Open to messages, code review requests, NDA conversations, paid consulting, or coffee. If you have spare GPU hours and want to see what 8 months of observational methodology produces when paired with reasoning-capable judges, I'd love to hear from you.

---

#AISafety #LLMEvaluation #AIAlignment #OpenSource #Research #ResearchCollaboration #MachineLearning #IndependentResearch #AIPolicy #AlignmentResearch
