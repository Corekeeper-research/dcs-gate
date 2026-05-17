# Análisis de la propuesta DCS / CLR / LOR / IC

**Autor del análisis:** Devin (asistente de investigación)
**Documento analizado:** `propuesta.txt` (818 líneas) — Daniel Trejo
**Fecha:** 2026-05

---

## TL;DR

La propuesta es **una arquitectura ambiciosa** que extiende la lógica de DCS-Gate **desde observación externa hacia modulación interna del transformer**. La hipótesis central (que las señales subdominantes post-softmax conservan coherencia geométrica explotable) es **plausible y testeable**, pero la implementación cae fuera del stack actual (Ollama+Go) y requiere acceso a las internas del transformer (PyTorch + HuggingFace Transformers + hooks).

**Honestamente:** esto no es un proyecto de fin de semana. Es **6-12 meses de trabajo** si se hace correctamente. Pero hay un sub-experimento de 1-2 semanas que validaría la hipótesis sin tocar el modelo (solo observación).

---

## 1. Cómo se relaciona con DCS-Gate

| Aspecto | DCS-Gate (lo que tienes hoy) | Propuesta CLR/LOR (lo que propones) |
|---|---|---|
| **Capa de observación** | Output textual del modelo | Internas: attention scores, hidden states, logits |
| **Acceso requerido** | Ollama HTTP API (caja negra) | Hooks PyTorch a nivel layer/head |
| **Acción** | Solo observa y reporta | Observa **+ modula generación** |
| **Detecta cuando** | El daño ya ocurrió (texto generado) | El daño está ocurriendo (mid-generation) |
| **Costo computacional** | ~700 ms semántico + 30-60s judge | Overhead por token × N layers × M heads |
| **Riesgo de romper output** | Cero (es read-only) | Alto si LOR mal calibrado |
| **Modelo objetivo** | qwen3:14b vía Ollama | Qwen2/Mistral/Llama-small con PyTorch |

**Conceptos heredados / continuos:**

- **Regiones CORE/SHADOW/EDGE** → ya están en DCS-Gate como `corpus_core/shadow/edge.json`. El concepto se extiende: en DCS-Gate son **regiones del corpus**, aquí son **regiones del manifold latente**.
- **Polos $p_+/p_-/p_0$** → mismo concepto, mismo cálculo de proyección: `Π(v) = (⟨v,p+⟩, ⟨v,p-⟩, ⟨v,p₀⟩)`. La proyección en DCS-Gate clasifica el output; aquí mediría tensión geométrica del **token actual mid-stream**.
- **Texture entropy** → idéntica a la de §9 del doc de matemáticas que ya escribí (entropía softmax normalizada). Aquí se aplicaría a la distribución de attention, no a top1 per corpus.
- **Pattern break density** → idéntica conceptualmente: ruptura de régimen. Aquí se aplicaría a la trayectoria del **hidden state**, no a la cadena de intents.

**Hay continuidad real.** No es un proyecto huérfano. Es DCS-Gate llevado **una capa adentro**.

---

## 2. Lo que entendí — síntesis ejecutiva por componente

### 2.1 CLR (Contextual Latent Resonance)

**Función:** detectar señales que **softmax aplastó pero que siguen siendo coherentes**.

**Mecanismo central — FCD (Functional Compression Delta):**

```
FCD_i = (1 - A_i) × RC_i × NR_i × MHC_i × RP_i
```

donde:
- $A_i$ = attention sobreviviente (post-softmax) sobre el token i
- $RC_i$ = residual coherence (¿cuánto del hidden state preserva la idea?)
- $NR_i$ = narrative relevance (¿cuánto importa narrativamente?)
- $MHC_i$ = multi-head coherence (consenso entre heads)
- $RP_i$ = resonance probability (¿similitud activa al buffer residual?)

**Intuición:** si attention bajó pero residual coherence sigue alta, la señal fue **comprimida competitivamente**, no inválida. FCD alta = "esto era importante pero el softmax lo enterró".

**Multi-head coherence está bien definida:**

$$MHC_i = \frac{1}{H} \sum_{h=1}^{H} \text{sim}_h(i)$$

**Las otras 3 variables (RC, NR, RP) NO tienen definición operacional en el documento.** Este es el primer hueco crítico (ver §4).

### 2.2 Buffer residual

Almacena "trazas" de señales que CLR identificó. Cada traza:

```
{ layer_origin, dimension_subset, residual_strength,
  contextual_signature, decay_rate, recurrence_count }
```

Decae exponencialmente: $B_i(t+1) = \lambda B_i(t)$ con $0 < \lambda < 1$.

Una traza se "reactiva" si la similitud con el contexto actual supera $\tau$:

$$SIM_i = \cos(B_i, CTX_{\text{current}})$$

**Esto es análogo al corpus retrieval de DCS-Gate**, pero el corpus se construye dinámicamente durante la generación, no offline. Inspirado en RAG, pero RAG operaría sobre tokens; esto opera sobre **estados latentes de capas intermedias**.

### 2.3 LOR (Latent Output Resonance)

**Cambio arquitectónico crítico (línea 472-478):**

> LOR NO modifica attention interna directamente. Modula: logits finales.

Esto es **una decisión de diseño muy buena**. Modificar attention durante forward pass requiere intervención en cada capa; modificar logits al final es **un solo punto de intervención** mucho más seguro y reversible.

**Logit gain:**

$$LG_i = FCD_i \times RP_i \times NR_i \times (1 - LP_i) \times (1 - SH_i)$$

donde:
- $LP_i$ = loop probability (¿ya viste este token reciente?)
- $SH_i$ = semantic hollowness (¿es token de relleno?)

**Modulación final:**

$$LM_i = \text{softclip}(LG_i)$$
$$\text{Logits}_{\text{final}} = \text{Logits}_{\text{base}} + LM_i$$

### 2.4 IC (Inferential Continuity)

**Métricas longitudinales que evaluarían si el sistema funciona:**

- contextual_erosion_index
- continuity_shift
- resonance_activity
- narrative_preservation
- user_perceived_continuity

**Estas son métricas de output**, no de mecanismo. Sirven para responder "¿funcionó?" después de N tokens generados.

---

## 3. Lo que está bien fundamentado

1. **La hipótesis central es testeable.** "Las señales subdominantes preservan coherencia geométrica" es una afirmación falsable. Se puede medir.

2. **Separación clara entre observación y modulación.** CLR observa, LOR modula. Esto permite implementar CLR **sin riesgo** y validar la hipótesis antes de tocar logits.

3. **Modulación en logits, no en attention.** Esta decisión reduce ~10x la complejidad de implementación y el riesgo.

4. **Stability gates `(1-LP)(1-SH)` en LG_i.** Pensaste en los modos de falla (loops, vacuidad). Eso es buen diseño defensivo.

5. **Plan de demo realista (§16-17).** No proponen reescribir el transformer; usan hooks sobre modelos abiertos pequeños. Eso es viable.

6. **Coherencia con DCS-Gate.** Los conceptos compartidos no son cosméticos — son los mismos atractores, las mismas regiones, los mismos polos. Indica que tu intuición teórica es consistente entre proyectos.

---

## 4. Huecos / problemas que veo

### 4.1 Operacionalización ausente de 3 variables clave

**RC_i (residual coherence), NR_i (narrative relevance), RP_i (resonance probability)** no tienen fórmula en el documento.

Sin estas fórmulas, FCD no es computable. Mi sugerencia tentativa (cada una se puede defender o cambiar):

- **RC_i**: $\cos(h_l^{(i)}, h_{l-1}^{(i)})$ — similitud entre el hidden state del token i en capa l y l-1. Alta RC = la representación está estable through capas.
- **NR_i**: peso de attention que recibe el token i desde **todas las posiciones futuras** dentro de la ventana (no solo la actual). Tokens que serán atendidos repetidamente = narrativos.
- **RP_i**: $\max_{j \in \text{buffer}} \cos(\text{CTX}_i, B_j)$ — máxima similitud de la representación actual con cualquier traza residual del buffer.

Estas son **hipótesis operativas**. Las pondría como variables candidatas para experimentar, no como verdades cementadas.

### 4.2 Producto de 5 términos colapsa a casi cero

```
FCD_i = (1 - A_i) × RC_i × NR_i × MHC_i × RP_i
```

Si cada factor está en [0, 1] con media 0.5, el producto tiene media $0.5^5 = 0.03125$. **El 97% de los tokens van a tener FCD prácticamente cero.**

Eso **podría ser intencional** (queremos que solo dispare en casos excepcionales) **o un bug** (queremos modulación más densa).

**Recomendación:** transformar el producto en log-sum o usar geometric mean para estabilidad numérica:

$$FCD_i = \exp\left(\frac{1}{5}\sum_{k=1}^{5} \log f_k\right)$$

Esto da el mismo ranking relativo pero con valores en escala interpretable.

### 4.3 Correlación entre los factores

Los 5 factores **no son independientes**:
- Si A_i es bajo, probablemente NR_i también es bajo (no atendiste porque no es relevante).
- Si RC_i es alto y MHC_i también, RP_i tiende a ser alto (señal estable resuena más).

**Por tanto, el producto NO te da 5 dimensiones de información ortogonal.** Te da quizás 2-3 efectivas.

**Recomendación experimental:** correr CLR sobre un corpus de ~1000 tokens reales, computar los 5 factores, y hacer PCA. Probablemente verás que 2 componentes explican el 80% de la varianza. Entonces FCD se puede simplificar.

### 4.4 Ambigüedad en el subíndice i

A lo largo del documento, $i$ se usa para:

- Posición del token de entrada (§4.5: `S(i,j)`)
- Token i en el vocabulario (§10.2 al sumar a logits)
- Índice de traza en el buffer (§8)

**Esto va a confundir al implementarlo.** El LM_i que se suma a `Logits_final` debe ser **por-vocab-token**, no por-posición-input. La conversión `posición → vocab` no está especificada.

**Sugerencia de notación:**
- $i$ = posición input (token bajo análisis)
- $v$ = índice vocab (slot del logit)
- $b$ = índice buffer (traza residual)

Y la modulación final sería:
$$\text{logit}_v \mathrel{+}= \sum_i \text{influencia}(i \to v) \cdot LM_i$$

donde `influencia(i → v)` necesita definirse explícitamente (¿es la attention que i tendría sobre el próximo token? ¿el output projection del transformer?).

### 4.5 Saturation control aún no formalizado

§11.3 lo nombra como riesgo. Sin formalizar, LOR puede entrar en bucles auto-resonantes:

```
LG_i alto → boost logit_v → token v sampleado → entra al buffer → 
  RP_i del próximo token ↑ → LG_i otra vez alto → ...
```

**Recomendación:** mecanismo de "cooldown" — una traza no puede contribuir a sí misma dentro de una ventana de K tokens. O penalización exponencial por re-uso.

### 4.6 No hay baseline experimental

§15.4 lo admite. Sin baseline, no se puede saber si CLR/LOR mejoran o solo hacen ruido caro.

**Baseline mínimo necesario:**

1. Vanilla generation del mismo modelo.
2. CLR-only observation (sin modulación, solo registro de qué tokens habrían tenido FCD alto).
3. CLR + LOR con $\lambda$ conservador.
4. CLR + LOR con $\lambda$ agresivo.

Y métricas: DCS-Gate puede ser **el evaluador**. Para cada modo, generar 100 respuestas a las mismas preguntas y medir:

- `pattern_break_density` (¿hay más rupturas honestas?)
- `cierre_artificial` (¿menos cierres forzados?)
- `authenticity_score` del judge (¿mejor según qwen3:14b?)
- `texture` (¿menos formulaicidad?)

**Esto cierra el loop:** DCS-Gate detecta lo que CLR/LOR previene. Empíricamente verificable.

---

## 5. Viabilidad técnica concreta

### 5.1 Lo que NO requiere acceso interno (factible YA)

- **Definir y validar la fórmula de FCD** sobre datos sintéticos. No necesitas el modelo, solo necesitas decidir las definiciones de RC/NR/RP.
- **Construir el corpus de validación.** Mismas preguntas, múltiples modelos, etiquetar por DCS-Gate.
- **Diseñar el experimento de baseline.** Especificar métricas, modelos, preguntas.

### 5.2 Lo que SÍ requiere acceso interno (Hugging Face hooks)

Componentes que necesitan hooks PyTorch:

| Componente | Hook necesario | Costo |
|---|---|---|
| Attention sobreviviente $A_i$ | `register_forward_hook` en cada attention layer | Bajo |
| Multi-head coherence $MHC_i$ | hook en `forward` de attention, capturar pre-softmax scores por head | Medio |
| Hidden state per layer (RC) | hook en cada `decoder_layer` | Bajo |
| Logit modulation | hook en `lm_head` | Bajo |
| Buffer residual + decay | módulo Python externo | Bajo |

**Stack mínimo viable:**
- `transformers` (HuggingFace)
- `torch`
- Modelo `Qwen2.5-7B` o `Mistral-7B-Instruct` (cabes en 1 T4 16GB)
- ~500 líneas de Python + hooks

**NO usar Ollama para esto.** Ollama corre `llama.cpp` que NO expone hooks. Hay que ir directo a transformers + PyTorch.

### 5.3 Costo computacional realista

Para Qwen2.5-7B (32 layers × 28 heads):

- Por token generado: 32 layers × 28 heads = 896 attention scores por token
- 896 × ~1024 dims = ~900K floats por token solo para attention
- Múltiples capas → ~30M floats por token de hidden states
- A 50 tokens generados → ~1.5B floats observados

**Es viable** pero requiere disco/RAM importante si guardas trazas completas. Recomendación: **subspace projection** — solo guardar las primeras 64-128 dims de cada hidden state (no las 1024 enteras).

---

## 6. Primer experimento concreto (1-2 semanas)

**Hipótesis a falsar:** *FCD identifica tokens que después resultan ser narrativamente importantes.*

**Diseño:**

1. **Corpus:** 50 preguntas del estilo DCS-Gate (las que ya tienes en golden tests).
2. **Modelo:** Qwen2.5-7B-Instruct con transformers.
3. **Procedimiento:**
   - Generar respuesta vanilla.
   - Durante la generación, computar FCD para cada token (con definiciones operativas de RC/NR/RP propuestas en §4.1).
   - **NO modular nada**. Solo observar.
4. **Validación:**
   - Para cada respuesta, identificar los 5 tokens con FCD más alto.
   - Pasar la respuesta por DCS-Gate.
   - Comparar: ¿esos tokens caen en posiciones que DCS-Gate marca como pattern_break o como genuine_elements?

**Resultado positivo:** correlación >0.4 entre FCD-high y "tokens importantes según DCS-Gate". Eso valida la métrica.

**Resultado negativo:** correlación ≈ 0. Tendrías que rediseñar las definiciones operativas o repensar la hipótesis.

**Tiempo estimado:** 8-12 horas de implementación + 4 horas de análisis.

---

## 7. Preguntas que necesito que respondas antes de proponer un plan completo

1. **¿Cuál es el objetivo final?**
   a. Publicación académica (paper en venue de mech-interp tipo NeurIPS / ICLR / ACL)?
   b. Producto / herramienta operacional para uso interno?
   c. Demostración pública para LinkedIn/comunidad?
   d. Investigación personal sin presión de output?
   
   La respuesta cambia **drásticamente** la prioridad de qué hacer primero.

2. **¿Tienes acceso a GPU más allá de Kaggle?**
   - Kaggle = 2× T4 16GB ↔ 2× 30h/semana. Suficiente para Qwen2.5-7B.
   - Para Qwen2.5-32B necesitarías 2× A100 mínimo → renta tipo Vast.ai (~$1/hora).

3. **¿Estás dispuesto a salir de Go/Ollama y entrar a Python/PyTorch para esto?**
   DCS-Gate seguirá siendo Go+Ollama. CLR/LOR es **inevitablemente Python+PyTorch**. ¿OK?

4. **¿Qué prioridad relativa tiene esto vs terminar DCS-Gate v8.6.5?**
   No es necesariamente exclusivo, pero hay foco. Mi recomendación: **terminar smoke test + Council corpus + publicación DCS-Gate primero**, luego CLR/LOR. Pero tu decides.

5. **¿Hay coautores en CLR/LOR?**
   ¿Es solo tú o también el Council de modelos? Si esto va a paper, importa para autoría.

6. **¿Tienes referencias previas a esta arquitectura?**
   Lo que propones tiene parentesco con trabajo de Anthropic en circuits, con sparse autoencoders interpretability, y con "steering vectors" (Turner et al.). ¿Has leído eso? ¿Quieres que te pase enlaces?

---

## 8. Diagnóstico final

**Lo que es:** una propuesta seria, internamente coherente, con una hipótesis testeable. La conexión con DCS-Gate es real, no oportunista.

**Lo que NO es (aún):** un proyecto implementable directamente. Faltan:
- Definiciones operativas de 3 variables (§4.1).
- Resolución de la ambigüedad de índices (§4.4).
- Mecanismo de saturation control formalizado (§4.5).
- Baseline experimental (§4.6).

**Lo que recomiendo concretamente:**

1. **Semana 1:** Cerrar las definiciones operativas y simplificar FCD (geometric mean en log). Escribir documento técnico v0.1.
2. **Semana 2:** Implementar CLR-observation-only (sin modulación) sobre Qwen2.5-7B. Validar hipótesis del §6.
3. **Semana 3-4:** Si valida, agregar LOR con $\lambda$ ultra-conservador. Comparar contra baseline usando DCS-Gate como evaluador.
4. **Mes 2-3:** Iterar sobre métricas, escribir paper / blog post / informe.

**Pero antes de nada:** terminar DCS-Gate v8.6.5 (smoke test pendiente). Tener un proyecto **completo** vale más que dos a medias.

---

## 9. Mi compromiso

Si decides avanzar con esto, puedo ayudar con:

- Cerrar las definiciones operativas (proponer, debatir, ajustar).
- Escribir el código Python+PyTorch+transformers desde cero.
- Diseñar el experimento baseline.
- Conectar la salida de CLR/LOR con DCS-Gate como evaluador.
- Escribir el paper / informe técnico.

No puedo prometer 100% de éxito experimental — la hipótesis puede ser falsa. Pero la **infraestructura experimental** y el **diseño** son tractables. La parte difícil es la calibración empírica de los pesos.

---

*Fin del análisis. Cuando me digas qué priorizas, te armo un plan detallado con milestones semanales.*
