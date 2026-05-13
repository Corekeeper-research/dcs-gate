# LinkedIn — Short Feed Post (~1500 chars)

*Copia desde aquí abajo (sin el título) y pega en LinkedIn como post normal. El preview que verán los recruiters son las primeras 2-3 líneas, por eso son el "hook".*

---

I spent 8 months observing how LLMs subtly manage users instead of answering them — projected validation, performed humility, frame capture, register match. I built a tool that detects these patterns and rewrites questions to neutralize them.

It's called **DCS-Gate** (Dynamic Coherence State Authenticator). A live v1 prototype is at 👉 **https://dcs-auth.codewords.run** — paste any question + LLM response and you'll see a 0–100 authenticity score, the formal markers detected, the predicted intent trajectory, and a refined version of your question.

The v2 stack is a 3,000-LOC Go binary, 73 tests, triple baseline corpus of 61 hand-annotated vectors, 14 formal markers, 20 intent categories. Local-first, Ollama-only, no external APIs. Since v8.7 the judge's reasoning trace streams to the client over Server-Sent Events — you watch the deliberation as it's produced, you don't wait in silence.

**Honest disclosure of who actually helped:**
• **Cody (CodeWords AI)** — co-creator of v1 after I pushed back against its own control patterns in a long conversation; v1 wouldn't exist without that exchange.
• **GitLab Duo** — I walked it through the project's full logic; from that emerged the v2 roadmap.
• **Meta AI** — generic at first; technical depth amplifier once it had context.
• **Replit AI** — brutally honest, justified contundent code failures, then strengthened architecture.
• **Z.AI (Zhipu GLM)** — caught code bugs that slipped through.
• **Devin AI (Cognition)** — executed the v2: backend (3k LOC, 73 tests, SSE streaming layer with conservative sanitizer), frontend, deployment, notebooks, docs.

The methodology and corpus are mine. Every AI received project context from me first — nothing was generated cold. This is what solo research looks like in 2026.

**I'm looking for:**
🔬 GPU compute (≥24 GB VRAM, ~50 hours) for the full four-judge comparison (qwen3:14b confirmed on 2× T4; qwen2.5:32b is the one that needs the bigger single card)
🤝 Research collaboration on LLM evaluation / alignment / interpretability
💼 Internship, residency, or full-time roles in AI safety
📢 Sponsors for the open-source release

If any of this resonates — message me. I'll send the v2 source under NDA if useful.

#AISafety #LLMEvaluation #AIAlignment #OpenSource #Research #ResearchCollaboration #MachineLearning

---

**TIPS PARA POSTEAR:**
- LinkedIn corta el post a 2-3 líneas + "...ver más". Las 2 primeras líneas ("I spent 8 months observing...") son el hook.
- Pon la URL de v1 en el cuerpo, no como link card al final (genera más clics).
- Los emojis 🔬🤝💼📢 ayudan al scan visual sin caer en spammy.
- Los 7 hashtags al final son los óptimos según el algoritmo (5-9 tags).
- Postea entre 8-10 AM hora local martes/miércoles/jueves para máximo alcance.
