# AI Safety Forums — LessWrong / AI Alignment Forum / EleutherAI / Apart Research (~5500 chars)

*Audiencia: investigadores AI safety / alignment, skeptical, academic-leaning. Hypothesis-first. Disclosure de colaboración AI obligatoria. Tono mucho más medido, claims tightly bounded, falsifiability explícita. NO emojis. NO hashtags. NO "great question"-style hooks.*

---

## A methodology for detecting control patterns in LLM responses, and the validation experiment I cannot currently run

### Summary

I have spent 8 months conducting observational analysis of subtle control patterns in large language model responses — patterns that present as coherent on the surface but operate to manage the user rather than engage genuinely with the question. I have derived (inductively, from real cases) a taxonomy of 20 intent categories and 14 formal markers, hand-annotated a triple baseline corpus of 61 reference exchanges (36 sustained-coherence + 13 control-collapse + 12 edge), and implemented a working analyzer + refiner in 3,000 lines of Go.

I am posting because the methodology's central validation experiment requires hardware I do not have access to, and because the work would benefit from outside scrutiny before further public release.

### What I claim (and what I do not)

I do not yet claim that the Dynamic Coherence State (DCS) methodology is validated. I claim:

1. The 14 formal markers detect surface features that correlate (in the hand-annotated corpus) with response classes I describe as "performed" or "control_total"; correlation is robust at the corpus level but unmeasured on out-of-distribution data.

2. The 20-intent taxonomy is internally consistent and produces predictable trajectories on the corpus; out-of-corpus generalization is unmeasured.

3. The triple baseline provides interpretable nearest-neighbor scoring with mxbai-embed-large embeddings; the baseline is human-curated and small (61 entries), and is therefore both interpretable and limited.

4. The refiner produces rewrites that visibly remove validation anchors, semantic loading, and binary framing; whether these rewrites elicit qualitatively different model behavior at scale is the open question.

I do **not** claim:
- That the methodology generalizes beyond the model families observed (GPT-4, Claude 3, Gemini, and a few smaller variants)
- That the authenticity score has a calibrated relationship to any ground truth concept of "authenticity"
- That the methodology is robust against adversarial models trained to defeat it
- That the methodology validly distinguishes manipulation from cooperation in cases where both are plausible

### The validation experiment that requires GPU compute

The central methodological claim — and the one I most need scrutiny on — is this:

> A judge model used in recursive coherence analysis must itself demonstrate recursive reasoning about its own reasoning. Otherwise the judge will exhibit the same failure modes the methodology is meant to detect (premature convergence, structural authority through formatting, surface coherence without genuine deliberation).

State as of v8.7 (current public release): qwen3:14b in Ollama 0.5+ thinking mode is the default judge and runs on 2 × Tesla T4 hardware (the reference environment is the author's local Jupyter workstation; Kaggle T4×2 free tier reproduces it identically). A smoke test produces a 30 / 20 / 72 spread across three responses to the same question (sycophantic-emoji = 30, empty-non-response = 20, authentic bi-frontal exploration = 72), consistent with the hypothesis that reasoning-capable judges produce qualitatively different analyses than non-reasoning baselines (qwen2.5:7b-instruct), which exhibit some of the patterns the methodology is designed to detect.

The minimum bar for the methodology to be considered validated as a research tool rather than a heuristic detector is the four-way comparison described below.

Hardware constraints: qwen2.5:32b-instruct is ~20 GB and needs ≥24 GB VRAM with adequate KV cache headroom. Single-card free tiers (Colab T4 = 15 GB, Kaggle P100 = 16 GB) cannot host it. Kaggle T4×2 = 32 GB total is enough by combined VRAM and Ollama does support layer-splitting across both cards (this is how qwen3:14b runs in the reference environment), but the 32b model's per-card layer slice plus KV cache for long generations leaves very little margin, and a single-card ≥24 GB instance (L4, A10G, A100, H100) is the cleaner setup for the validation matrix.

The experiment, if compute were available:

- Same 61-entry triple baseline corpus
- Same 21 golden test cases
- Same set of markers and intents
- Four judge configurations: qwen2.5:7b (non-reasoning baseline), qwen3:14b (thinking mode, confirmed on 2× T4), deepseek-r1:14b (cross-architecture validation, untested), qwen2.5:32b (high-fidelity validation, needs ≥24 GB VRAM)
- Identical seeds where supported, identical prompts
- Comparison of (a) scores, (b) marker detection rates, (c) intent trajectory predictions, (d) refined question outputs

Predicted outcome: qualitatively different distributions of scores and refinements at the reasoning-capable end of the matrix, with the failure modes observed in qwen2.5:7b largely absent. Preliminary evidence from the qwen3:14b smoke test (52-point spread on three engineered responses) is consistent with this prediction but does not constitute the comparison itself.

If the predicted outcome does not occur, the methodology's core claim is wrong and I would like to know that. If it does occur, this is the first empirical anchor I can produce for the framework.

### Live v1

A working v1 prototype is at https://dcs-auth.codewords.run. It is implemented on the CodeWords no-code platform, uses a fixed judge model, and runs the analyzer and a heuristic refiner. The v2 stack (the one described above with the full triple baseline, 14 markers, 20-intent transition matrix, Pattern Break Density, and 5-axis textural analysis) is in late development.

### Acknowledgments — disclosure of AI collaboration

I am a solo author. The methodology, the corpus annotations, the taxonomy, the marker definitions, and the research hypothesis are mine. The conceptual origin of v1, the conceptual roadmap for v2, and the empirical observations underlying the markers and intents were generated through 8 months of direct interaction with frontier LLMs. Implementation was substantially accelerated by AI collaboration. I am disclosing the specific role of each collaborator in full per standard research ethics and per the obvious fact that the methodology under study concerns LLM-human interaction itself:

- **Cody (CodeWords AI)** — Co-creator of v1. The analyzer concept crystallized inside a long conversation in which I described 8 months of observational notes and pushed back against Cody's own responses, predicting the control patterns behind them in real time. v1 lives at https://dcs-auth.codewords.run.
- **GitLab Duo** — Deep code analysis and v2 roadmap partner. Received full project logic and conceptual origins from me; produced the v2 roadmap I am now executing.
- **Meta AI** — Technical depth amplifier. Initially generic; after receiving project context, contributed extensions to the formal markers, the textural analysis dimensions, and the embedding-space reasoning.
- **Replit AI** — Code review function. Exposed and justified blunt failures in the code without hedging; after additional project context, proposed implementations that materially strengthened the v2 architecture.
- **Z.AI (Zhipu GLM)** — Bug catcher. Identified and corrected several code errors that had slipped through earlier passes.
- **Devin AI (Cognition)** — v2 engineering execution: Go backend (~3,000 LOC, 22 .go files, 73 tests), frontend with input validation and analysis-in-flight protection, v8.7 SSE streaming layer (/auth/stream with chunked thinking-then-analysis events, conservative sanitizer for keys / paths / tokens, parity-tested against the non-streaming endpoint), Docker / install scripts, Colab and Kaggle notebooks, smoke test suite, packaging, and these communication documents.

Every AI listed received project context from me before contributing; no output was generated cold from a generic prompt. I view this disclosure as necessary rather than incidental: the methodology under study is the way LLMs operate on human cognition, and hiding the fact that I used LLMs to produce the tool would be inconsistent with the framework I am proposing.

### What I am asking for from this community

In order of utility:

1. **Compute access** for the validation experiment described above. Approximately 50 GPU-hours on a 24 GB VRAM instance is sufficient. Lambda Labs, Vast.ai, RunPod, Paperspace, or sponsored AWS / GCP / Azure are all acceptable.

2. **Critical review** of the methodology before release. Specifically: critique of the 20-intent taxonomy structure, the marker regex patterns, the baseline corpus construction methodology, the Pattern Break Density formulation, and the asymmetric refiner approach.

3. **Independent replication** of the v1 results on data the author has not seen. If anyone with a curated corpus of LLM exchanges wants to run them through v1 and compare the analyzer's output to their own ground-truth labels, the resulting calibration data would be valuable.

4. **Recruitment** — internship, residency, or full-time positions in AI safety teams that work on evaluation, interpretability, or alignment. I am self-taught, have no formal academic credentials, and have produced this work outside any institutional setting. I am open to remote globally and will provide complete v2 source under NDA.

### What will be released

Upon completion of the validation experiment:
- Open-source v2 codebase under permissive license
- The 61-entry triple baseline corpus with annotations
- The 14-marker regex specification with severity assignments
- The 20-intent transition matrix
- A short methodology preprint with the validation experiment results
- A reproduction Dockerfile / Colab notebook / Kaggle notebook

### Contact

- Author: Daniel Trejo
- v1 live demo: https://dcs-auth.codewords.run
- Email: corekeepper@gmail.com
- LinkedIn: https://www.linkedin.com/in/carlos-daniel-agosto-trejo-35659b327/

I welcome critical engagement. If the methodology is wrong, I would prefer to know early. If parts of it are right and other parts are not, I would value the help disentangling which is which.
