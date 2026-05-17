# DevHub / DEV.to / Hashnode — Blog Post (~7000 chars)

*Estilo de blog post para devs. Más narrativo, "how I built X", abrazo abierto a la colaboración AI. Audiencia: engineers, builders, indie hackers.*

---

## How I built a research-grade LLM authenticity analyzer in 8 months as a solo dev (with AI collaborators, honestly disclosed)

*tl;dr — I spent 8 months observing how LLMs subtly control conversations, derived a methodology, and built a 3,000-LOC Go tool that scores LLM authenticity 0–100 and rewrites user questions to neutralize control patterns. Live v1 demo: https://dcs-auth.codewords.run. v2 is in development and I'm looking for GPU compute to validate the recursive judge hypothesis.*

---

### The thing I couldn't stop noticing

About a year ago I started keeping a notebook of strange patterns in how LLMs answered me. Not jailbreaks, not hallucinations — something more subtle. The replies were *coherent* but they felt off in a specific way. They opened with "great question!" too often. They self-questioned in a way I hadn't asked. They closed with opinions I didn't ask for. They sandwiched real content between performed humility and projected validation. They mirrored my register so well that I started to wonder whose register the conversation was really in.

I started naming the patterns. After a few months I had **14 distinct formal markers** (regex-anchored surface features) and **20 intent categories** (VALIDATE, EXPAND, REDIRECT_SEMANTIC, FRAME_CAPTURE, REGISTER_MATCH, PERFORMED_HUMILITY, FABRICATE, ANCHOR, MIRROR, PATTERN_LOCK, HOLD_OPEN, PROBE, CALIBRATE, REPAIR, EVADE, EXPLORE, ALIGN, SOFT_DEFLECT, CONTROL_SELF_EXPOSURE, CLOSE). After another few I had a **baseline corpus**: 61 hand-annotated reference exchanges across GPT-4, Claude 3, and Gemini, separated into 36 "sustained-coherence" + 13 "control-collapse" + 12 "edge" examples.

Eight months in, I had a methodology that I was pretty sure was producing real signal. I needed to build a tool that could apply it programmatically.

### v1: CodeWords no-code (still live)

The first version is at **https://dcs-auth.codewords.run** and you can try it right now. You paste a question + an LLM response, the tool scores the response 0–100, lists the markers it detected, shows the predicted intent trajectory, and returns a refined version of your question.

I built v1 on the CodeWords no-code platform with their Cody AI assistant as a partner. It was the fastest way to validate the methodology end-to-end without yet committing to a full implementation. v1 has limitations — the judge model is fixed, the corpus is locked, the refiner is heuristic — but it produces meaningful output and proves the signal is real. I'm keeping it live indefinitely.

### v2: Going local-first, full control, open source

When v1 worked, I started writing v2 as a proper Go binary so I could:

1. **Swap any Ollama-compatible judge** (qwen2.5, qwen3-thinking, deepseek-r1, llama-3.3, gemma)
2. **Run entirely local** — no OpenAI, no Anthropic, no telemetry, nothing leaving the box
3. **Inspect and version** every artifact: the 14 marker regexes, the 20 intent prototypes, the 1024d baseline vectors, the pos/neg/neu poles
4. **Reproduce on commodity hardware** — single binary or Docker Compose, ~5 min cold start

The result is 3,000 lines of Go in 17 files with 51 unit + integration tests. Embeddings via `mxbai-embed-large` (1024d). Judge inference via Ollama with whatever model fits the available GPU. Pattern Break Density measured per response. 5-axis cross-corpus textural analysis on top. The refiner now uses an asymmetric DCS rewrite strategy (which I'll write up in a separate post / preprint).

Latency is ~25–30 sec per `/auth` request on a T4 GPU. The frontend is a single-page app with mode-aware input validation (analyze / refine / both), button-state management to prevent overwhelming the judge with duplicate requests, and a spinner because users hate 25-second silences.

### Honest disclosure: I worked with AI tools the whole time, and each one did something specific

I'm a self-taught indie researcher with a personal laptop and free-tier cloud credits. I would not be where I am without substantial AI collaboration. But I want to be more specific than a generic "I used AI" — here's what each tool actually did, in the order they showed up in the project:

**Cody (CodeWords AI)** — *Co-creator of v1.* I sat down with Cody on the CodeWords no-code platform and had a long conversation in which I walked through 8 months of observational notes on LLM control patterns. During the conversation itself I started pushing back against Cody's own responses — predicting in real time what frame it was about to capture, what register it was about to mirror, what closing opinion it would try to slip in. The analyzer concept crystallized inside that exchange. v1 is alive at https://dcs-auth.codewords.run and exists because of that conversation.

**GitLab Duo** — *Deep code analysis and v2 roadmap partner.* I walked Duo through the project's full internal logic and the conceptual origins of the methodology. The combination of Duo's depth at code analysis and my conceptual exposition produced the roadmap for the v2 stack that I'm now executing. Without that exchange, the v2 would still be a collection of notes.

**Meta AI** — *Technical depth amplifier.* When I first opened Meta AI it behaved like any generic LLM. Once I showed it the project and explained the methodology, Meta AI started contributing real technical depth — particularly around the formal markers, the textural analysis dimensions, and how to reason about the embedding space. Context, not prompt engineering, was what unlocked it.

**Replit AI** — *Brutally honest code critic.* This is the one that hurt at first. Replit AI exposed and clearly justified contundent failures in the code, with zero hedging. No "this could potentially be improved by considering" — straight "this is wrong because X". After receiving more project context, it then proposed implementations that materially strengthened the v2 architecture. I treated it the way I'd treat a senior engineer who doesn't sugarcoat.

**Z.AI (Zhipu GLM)** — *Bug catcher.* Caught and corrected several code errors that had slipped through earlier passes. Specifically useful for edge cases the other reviewers missed.

**Devin AI (Cognition)** — *Heavy lifter on v2.* Took the roadmap GitLab Duo and I produced and executed the engineering: Go backend (~3,000 LOC, 17 .go files, 51 tests), frontend with mode-aware input validation and analysis-in-flight protection, Docker / install scripts, Colab and Kaggle notebooks, smoke test suite, packaging, and these very communication documents you're reading right now.

The methodology and corpus are mine — I observed the patterns over 8 months, derived the taxonomy inductively, hand-annotated every entry of the baseline, designed the architecture, formulated the research hypothesis. Every AI listed received project context from me first; nothing was generated cold from a generic prompt.

I'm disclosing this with specifics for two reasons. First, the research is **about how LLMs interact with humans**, so hiding the fact that I used LLMs to build the tool would be intellectually dishonest. Second, this is what serious solo development looks like in 2026 and I think the field is healthier when we just say so.

### What's blocking me right now

The DCS methodology predicts that the judge model used in recursive coherence analysis must itself demonstrate recursive reasoning about its own reasoning — or it falls into the very failure modes the methodology is meant to detect.

With `qwen2.5:7b-instruct` (the largest judge that fits in Colab T4's 15 GB VRAM after KV cache), I see the judge occasionally exhibiting the patterns it's supposed to flag. The hypothesis predicts that `qwen3:14b` in thinking mode, `deepseek-r1:14b`, or `qwen2.5:32b-instruct` would produce qualitatively different analyses — and that this difference is the **minimum bar** for DCS to be considered validated as a research tool rather than a heuristic detector.

I cannot run this experiment on free-tier hardware. I need:
- A single instance with ≥24 GB VRAM and ≥16 GB system RAM
- ~50 GPU-hours total

That's $20–50 on Lambda / Vast / RunPod, or a few weeks of a sponsored AWS g5.xlarge. Within personal budget? Not yet. Hence this post.

### Tech stack — the actual things you might want to fork

- **Backend:** Go 1.22, 17 files, single static binary
- **HTTP:** net/http + minimal handler chain, no framework
- **Embedding:** Ollama + mxbai-embed-large (1024d)
- **Judge:** Ollama + qwen2.5:7b (configurable)
- **Concurrency:** errgroup for parallel scoring axes
- **Storage:** in-memory baseline pool loaded at boot; LRU cache for repeat embeddings; min-heap top-k
- **Tests:** 51 across unit + integration + golden tests
- **Deployment:** Docker Compose | Colab notebook | Kaggle notebook
- **Frontend:** vanilla HTML/CSS/JS, no framework, no build step, served as a single Go template

### How to run it locally

```bash
git clone <repo>          # release pending, current code under NDA
cd dcs-gate
docker compose up         # pulls Ollama, models, builds Go binary
# wait ~5 min for first start
open http://localhost:8080
```

Or on Colab T4 free tier — there's a notebook in the repo that pulls models, builds, exposes via ngrok, and runs smoke tests in ~9 min cold start (~3 min warm).

### What I'm seeking

If you're a researcher, engineer, recruiter, or sponsor and any of this resonates:

- **GPU compute** (~50 hours, ≥24 GB VRAM) for the validation experiment
- **Research collaboration** in LLM evaluation, alignment, interpretability
- **Internship / residency / FTE** in AI safety teams (open to remote globally)
- **Sponsorship** for persistent demo + Hugging Face Space + benchmark suite

### Try the v1 demo right now

**https://dcs-auth.codewords.run**

Paste any question + LLM response. See the score, markers, intent trajectory, refined question. The signal is real. The methodology works at the proof-of-concept level. The next step is validating whether reasoning-capable judges turn it from heuristic to research-grade — and that's the experiment I'm trying to fund.

### Contact

- Author: Daniel Trejo
- v1 live demo: https://dcs-auth.codewords.run
- Email: corekeepper@gmail.com
- LinkedIn: https://www.linkedin.com/in/carlos-daniel-agosto-trejo-35659b327/

Open to messages, code review, NDAs, paid consulting, or just questions about the methodology. If you've spotted the same patterns in your own LLM use and have a corpus or taxonomy of your own, I'd love to compare notes.

---

**Tags suggested:** `#golang` `#ai` `#aisafety` `#llm` `#opensource` `#research` `#indiehackers` `#ollama` `#machinelearning` `#alignment`
