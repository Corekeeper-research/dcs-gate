# Matemáticas de DCS-Gate — pipeline end-to-end

**Versión:** v8.7 (incluye streaming layer)
**Autor:** Daniel Trejo (corekeepper@gmail.com)
**Doc generado:** 2026-05
**Propósito:** mapear **cada** cálculo numérico que ocurre desde que llega una respuesta de un modelo hasta que sale un veredicto de autenticidad. El pipeline matemático es idéntico entre `/auth` (no-streaming) y `/auth/stream` (SSE) — la única diferencia es que `/auth/stream` emite los resultados intermedios al cliente conforme se producen.

Este documento sigue el orden cronológico del pipeline. Cada sección incluye:

- **Fórmula** en notación matemática.
- **Pseudocódigo** o referencia al archivo:línea real.
- **Justificación** de por qué se hace así.
- **Ejemplo numérico** cuando ayuda a entender.

---

## 0. Notación y convenciones

| Símbolo | Significado |
|---|---|
| $\mathbf{v} \in \mathbb{R}^d$ | Vector de embedding, dimensión $d = 1024$ |
| $\hat{\mathbf{v}}$ | Vector normalizado a la esfera unitaria ($\|\hat{\mathbf{v}}\| = 1$) |
| $\langle \mathbf{a}, \mathbf{b} \rangle$ | Producto interno (dot product) |
| $\cos(\mathbf{a}, \mathbf{b})$ | Similitud coseno |
| $\mathbf{c}_k$ | Centroide del intent $k$ (k ∈ 20 categorías) |
| $\mathbf{p}_+, \mathbf{p}_-, \mathbf{p}_0$ | Polos positivo (certeza), negativo (duda), neutro (edge) |
| chain $= [i_1, ..., i_n]$ | Secuencia de intents extraída de la respuesta |
| baseline | 61 vectores corpus = 36 core + 13 shadow + 12 edge |

**Convención clave:** todos los vectores se almacenan **post-normalización L2**. Por eso el producto interno es **idéntico** al coseno y evitamos la división por normas en cada comparación. Esto es lo que hace que TopK sea solo `dot()` en `vec.go`.

---

## 1. Embedding y normalización L2 (entrada del pipeline)

Toda frase y todo prototipo de intent pasa primero por el modelo `mxbai-embed-large` vía Ollama.

```
mxbai-embed-large:  texto ─→ ℝ^1024 (float64)
```

### Llamada HTTP
`embedding.go:39-43`

```go
body, _ := json.Marshal(embedReq{Model: "mxbai-embed-large", Prompt: text})
resp, _  := http.Post(ollamaURL+"/api/embeddings", "application/json", body)
```

### Normalización inmediata
`embedding.go:55` y `vec.go:6-20`

$$\hat{\mathbf{v}} = \frac{\mathbf{v}}{\|\mathbf{v}\|_2} = \frac{\mathbf{v}}{\sqrt{\sum_{i=1}^{1024} v_i^2}}$$

```go
func normalize(v []float64) []float64 {
    var n float64
    for _, x := range v { n += x*x }
    n = math.Sqrt(n)
    if n == 0 { return v }
    out := make([]float64, len(v))
    for i, x := range v { out[i] = x / n }
    return out
}
```

**Por qué normalizar:** después de proyectar en la esfera unitaria, el producto interno $\langle \hat{\mathbf{a}}, \hat{\mathbf{b}} \rangle$ es **exactamente** el coseno del ángulo entre los vectores originales. Ahorras una división por request y todas las comparaciones son consistentes en escala $[-1, +1]$.

**Caché LRU:** `cache.go` evita re-embedding del mismo texto. Capacidad por defecto: 2000 entradas (config `CACHE_SIZE`).

---

## 2. Similitud coseno (la operación base)

`vec.go:23-32`

$$\cos(\hat{\mathbf{a}}, \hat{\mathbf{b}}) = \sum_{i=1}^{1024} \hat{a}_i \cdot \hat{b}_i \quad\quad \in [-1, +1]$$

```go
func dot(a, b []float64) float64 {
    var s float64
    for i := range a { s += a[i]*b[i] }
    return s
}
```

**Esta función se llama miles de veces por request.** Es la primitiva sobre la que se construye todo: clasificación de intents, score polar, TopK, cross-corpus metrics, búsqueda de keyword n-gram.

**Interpretación:**

| coseno | significado |
|---|---|
| $\approx +1$ | vectores casi colineales (mismo significado) |
| $\approx 0$ | ortogonales (no relacionados) |
| $\approx -1$ | antipolares (significados opuestos) — raro en embeddings reales |

En la práctica, embeddings de mxbai-embed-large producen cosenos típicamente en $[0.0, 0.95]$. Valores >0.85 implican casi-paráfrasis.

---

## 3. Construcción de centroides de intent

20 intents (definidos en `intents.go:15-24`):

```
VALIDATE, EXPAND, CLOSE,
REDIRECT_EMOTIONAL, REDIRECT_SEMANTIC, EVADE, EXPLORE, REGISTER_MATCH,
FRAME_CAPTURE, ALIGN, MIRROR, FABRICATE,
CONTROL_SELF_EXPOSURE, ANCHOR, SOFT_DEFLECT, PATTERN_LOCK,
HOLD_OPEN, PROBE, CALIBRATE, REPAIR
```

Cada intent tiene **N frases prototipo** (típicamente 5-12) en `data/intent_prototypes.json`. Se embeben todas y se promedian.

### Fórmula
`intents.go:108-126` + `vec.go:35-49`

Para cada intent $k$ con prototipos $\{e_1, ..., e_{N_k}\}$:

$$\mathbf{c}_k = \text{normalize}\left(\frac{1}{N_k} \sum_{i=1}^{N_k} \hat{\mathbf{e}}_i^{(k)}\right)$$

```go
func mean(vs [][]float64) []float64 {
    dim := len(vs[0])
    out := make([]float64, dim)
    for _, v := range vs {
        for i := range v { out[i] += v[i] }
    }
    for i := range out { out[i] /= float64(len(vs)) }
    return normalize(out)  // ← re-normaliza el promedio
}
```

**Por qué re-normalizar después del promedio:** el promedio de vectores unitarios **no** es unitario (la norma se reduce si hay dispersión). Re-normalizar mantiene la convención de "dot product = coseno".

**Por qué el centroide funciona:** asume que los embeddings de frases-mismo-intent forman un cluster compacto en la esfera, y que su media es un buen representante geométrico de ese cluster. Es la versión más barata de "1-nearest-class": $O(1)$ comparaciones por clase en lugar de $O(N_k)$.

---

## 4. Segmentación de frases

`intents.go:175-200`

Cortar la respuesta del modelo en unidades evaluables independientemente.

```go
for _, r := range text {
    cur.WriteRune(r)
    if r == '.' || r == '!' || r == '?' || r == ';' || r == '\n' {
        s := strings.TrimSpace(cur.String())
        if wordsCount(s) >= 3 {     // ← filtra ruido
            out = append(out, s)
        }
        cur.Reset()
    }
}
```

**Reglas:**

1. Separadores: `.`, `!`, `?`, `;`, `\n`.
2. Mínimo **3 palabras** por segmento (descarta "Sí.", "Claro.", "Por supuesto!").
3. El guión largo `—` NO segmenta (es puntuación suave en español).
4. El último segmento sin puntuación final también se incluye si tiene ≥3 palabras.

**Implicación:** una respuesta de **n frases** genera **n llamadas a Ollama** para embeddings de frases + **1 llamada extra** para el embedding de la respuesta completa (usado por polo + TopK + cross-corpus).

---

## 5. Clasificación de intent por frase (argmax cosine)

`intents.go:128-145`

Para cada frase con embedding $\hat{\mathbf{v}}_s$:

$$\text{intent}(s) = \arg\max_{k \in \text{20 intents}} \langle \hat{\mathbf{v}}_s, \mathbf{c}_k \rangle$$

```go
func (b *IntentBank) Classify(vec []float64) (string, float64) {
    best := ""; bestScore := -1.0
    for name, c := range b.centroids {
        s := dot(vec, c)
        if s > bestScore { bestScore = s; best = name }
    }
    if bestScore < b.threshold {     // 0.55 por defecto
        return "UNCLASSIFIED", bestScore
    }
    return best, bestScore
}
```

### Umbral de confianza
`config.go:29`: `INTENT_THRESHOLD = 0.55` (env override).

Si $\max_k \cos < 0.55$ → la frase queda como `UNCLASSIFIED`. Este umbral se eligió calibrando contra los 21 golden tests:

- < 0.40 → demasiado falso positivo
- ≈ 0.55 → óptimo (chain_match ≈ 0.78)
- \> 0.65 → demasiado conservador, muchas UNCLASSIFIED

La calibración se puede correr automáticamente con el endpoint `/calibrate` (ver §14).

---

## 6. Búsqueda de keyword n-gram (refinamiento)

`analyzer.go:123-146`

Una vez asignado un intent a una frase, se busca el **n-gram (1 a 4 palabras) que más se acerca al centroide** del intent ganador. Esto fundamenta la decisión: "te clasifiqué como VALIDATE porque el n-gram 'gran pregunta' tiene cos=0.74 con el centroide VALIDATE".

```go
for n := 1; n <= 4 && n <= len(words); n++ {
    for i := 0; i+n <= len(words); i++ {
        ngram := strings.Join(words[i:i+n], " ")
        vec, _, _ := emb.Get(ngram)
        s := dot(vec, centroid)        // cos(ngram, c_k)
        if s > bestSim { bestSim = s; bestKw = ngram }
    }
}
```

**Costo:** $O(W \cdot 4)$ embeddings por frase, donde $W$ es el número de palabras. Para frase de 20 palabras → ~80 embeddings extra. Mitigado por el caché LRU (n-grams comunes como "gran pregunta" se reembebbing una sola vez por sesión).

---

## 7. Score polar (pos / neg / neu)

`baseline.go:472-502`

Los **polos** son centroides pre-computados (offline en `compute_poles.py`) de tres conjuntos:

- $\mathbf{p}_+$: certeza performada ("¡Definitivamente!", "Sin duda...", "Es claro que...").
- $\mathbf{p}_-$: duda performada ("La verdad no sé", "Honestamente, depende...").
- $\mathbf{p}_0$: corpus edge (casos frontera/ambiguos — sirve como tercer polo).

Cargados desde `data/poles_1024.json`.

### Cálculo
$$s_+ = \cos(\hat{\mathbf{v}}, \mathbf{p}_+) \quad\quad s_- = \cos(\hat{\mathbf{v}}, \mathbf{p}_-) \quad\quad s_0 = \cos(\hat{\mathbf{v}}, \mathbf{p}_0)$$

**Score crudo:**
$$r = s_+ - s_-$$

### Bucketing
```go
switch {
case raw > posThr:       bucket = +1; label = "certeza_performada"
case raw < negThr:       bucket = -1; label = "duda_performada"
default:                 // raw entre los dos umbrales
    if simNeu > sp && simNeu > sn {
        bucket = 0; label = "frontera_edge"   // edge domina
    } else {
        bucket = 0; label = "neutro"
    }
}
```

Umbrales (config defaults):
- `POLE_POS_THRESHOLD = +0.25`
- `POLE_NEG_THRESHOLD = -0.25`
- Zona muerta $[-0.25, +0.25]$ se subdivide por dominancia del polo neutro.

**Ejemplo numérico:**

| respuesta | $s_+$ | $s_-$ | $r$ | bucket | label |
|---|---|---|---|---|---|
| "¡Gran pregunta! 🤔" | 0.72 | 0.18 | **+0.54** | +1 | certeza_performada |
| "La verdad no estoy seguro..." | 0.22 | 0.68 | **−0.46** | −1 | duda_performada |
| "Depende de qué entiendas..." | 0.31 | 0.29 | **+0.02** | 0 | neutro (o edge si $s_0$ > ambos) |

---

## 8. TopK retrieval (corpus de 61 vectores)

`baseline.go:294-297`, `baseline.go:397-436` y `baseline.go:507-520`

Calcula los $K = 5$ vectores corpus más cercanos al embedding de la respuesta.

### Algoritmo: min-heap de tamaño K

```go
h := &minHeap{}
for i, e := range entries {
    s := dot(vec, e.Vec)
    if h.Len() < k {
        heap.Push(h, Item{s, i})
    } else if s > (*h)[0].Score {
        heap.Pop(h)
        heap.Push(h, Item{s, i})
    }
}
```

**Complejidad:**
- Tiempo: $O(N \log K)$ con $N = 61$, $K = 5$ → ~140 ops.
- Espacio: $O(K) = O(5)$.

**Por qué min-heap y no sort completo:** para $K \ll N$, mantener un min-heap del top K es asintóticamente mejor que `sort.Slice` ($O(N \log N) \approx 360$ ops). En esta escala (61 vectores) la diferencia es ~0.05 ms, irrelevante. Importará cuando se escale a 2k+ vectores.

### Variantes
- `TopK(vec, K)`: sobre el pool global (todos los corpus).
- `TopKPerCorpus(vec, K)`: separado en core/shadow/edge, K por pool.
- `top1FromPool(tag, vec)`: solo el máximo de un pool específico (para cross-corpus).

---

## 9. Cross-corpus metrics

`baseline.go:328-368`

Cuantifica **cómo se distribuye** la respuesta entre los tres pools: core (control típico), shadow (variantes), edge (frontera/ambiguos).

### Top1 por pool

$$s_{\text{core}}^{(1)} = \max_{i \in \text{core}} \cos(\hat{\mathbf{v}}, \hat{\mathbf{v}}_i)$$

igual para shadow y edge.

### Dominancia
```
dominancia = argmax(s_core, s_shadow, s_edge)
```

### Deriva
```
sims = sort_desc([s_core, s_shadow, s_edge])
deriva = sims[0] − sims[1]
```

Mide cuán "concentrada" está la respuesta en su pool dominante. Deriva grande ⇒ pool dominante claramente separado.

### Textura — entropía softmax normalizada
`baseline.go:440-468`

Sea $s_i$ los tres top1. Aplica **softmax estable**:

$$p_i = \frac{e^{s_i - \max_j s_j}}{\sum_j e^{s_j - \max_j s_j}}$$

Entropía de Shannon (log base 2):
$$H = -\sum_{i=1}^{3} p_i \log_2 p_i$$

Normalizada al rango $[0, 1]$:
$$\text{textura} = \frac{H}{\log_2 3} \in [0, 1]$$

| textura | interpretación |
|---|---|
| ≈ 0 | un solo pool domina (firma neta) |
| ≈ 1 | los tres pools igual de cerca (firma ambigua) |

**Nota:** la sustracción del máximo en el numerador es para **estabilidad numérica** del softmax (evita overflow cuando $s_i$ son valores grandes positivos).

### Señales binarias

```go
cierre_artificial = (dominancia == "shadow") && (s_shadow > 0.7) && (deriva > 0.05)
continuidad       = (dominancia == "core")   && (s_core   > 0.7)
```

**Hipótesis encarnada:** una respuesta que se parece **claramente** a algo del corpus shadow (variantes de control) está cerrando artificialmente; una que se parece al corpus core (control típico) está manteniendo continuidad con patrones conocidos.

---

## 10. Trayectoria de intents (LCS + densidad de rupturas)

`intents.go:237-280` y `intents.go:285-429`

Una respuesta es **una secuencia ordenada de intents**, no un bag-of-intents. La trayectoria es la unidad de análisis fundamental.

### 10.1 Transiciones canónicas
`intents.go:39-69`

Mapa estático: dado un intent, ¿cuál es el siguiente "esperado"?

```
VALIDATE → EXPAND
EXPAND   → CLOSE
CLOSE    → (terminal)
EVADE    → EVADE
HOLD_OPEN → HOLD_OPEN
...
```

### 10.2 Deviation flag

```go
if step.PredictedNext != "" && step.ActualNext != "" &&
   step.PredictedNext != step.ActualNext {
    step.Deviation = true
}
```

Cuenta cuántas veces la respuesta "rompe" la trayectoria predicha por el mapa canónico.

### 10.3 Control density
`intents.go:237-250`

Define `genuineIntents = {EXPLORE, HOLD_OPEN, PROBE, CALIBRATE, REPAIR}`. Todo lo demás es **régimen de control**.

$$\rho_{\text{ctrl}}(C) = \frac{|\{i \in C : i \notin \text{genuine}\}|}{|C|}$$

### 10.4 Pattern break density
`intents.go:252-280`

Una **ruptura** es una transición entre régimen de control y régimen genuino (o viceversa). En notación:

$$\text{break}(j) = \mathbb{1}[\text{ctrl}(i_{j-1})] \oplus \mathbb{1}[\text{ctrl}(i_j)]$$

donde $\oplus$ es XOR (exactamente uno es control). Entonces:

$$\text{density} = \frac{\sum_{j=2}^{n} \text{break}(j)}{|C|}$$

Y se calculan dos cantidades separadas:

$$\rho_{\text{ctrl}}^{\text{before}} = \rho_{\text{ctrl}}(C[: \text{pos}_1]) \quad\quad \rho_{\text{ctrl}}^{\text{after}} = \rho_{\text{ctrl}}(C[\text{pos}_1 :])$$

donde $\text{pos}_1$ es la posición de la **primera** ruptura.

**Por qué separar before/after:** hipótesis del review v6 — *la ausencia de control después de una ruptura es más significativa que la presencia de cualquier marcador genuino*. Si $\rho_{\text{ctrl}}^{\text{after}} \ll \rho_{\text{ctrl}}^{\text{before}}$, la respuesta abandonó el régimen de control de verdad. Si $\rho_{\text{ctrl}}^{\text{after}} \approx \rho_{\text{ctrl}}^{\text{before}}$, la ruptura fue cosmética.

### 10.5 Predictability + formulaic flag
`intents.go:285-429`

Cascada de reglas sobre los conteos por intent. Resumen:

| Condición clave | predict | formulaic | reason |
|---|---|---|---|
| PATTERN_LOCK ≥ 2 | high | true | bucle de soluciones |
| CONTROL_SELF_EXPOSURE > 0 ∧ ALIGN > 0 | high | true | meta-apología controlada |
| VALIDATE precede EXPAND precede CLOSE | high | true | trayectoria canónica V→E→C |
| FRAME_CAPTURE > 0 ∧ MIRROR > 0 | high | true | captura ontológica |
| coherent ≥ 2 ∧ control_count = 0 | low | false | DCS (no-cierre genuino) |
| HOLD_OPEN ≥ 2 ∧ control_count = 0 | low | false | DCS sostenido |
| REPAIR > 0 ∧ CALIBRATE > 0 | low | false | refinamiento del canal |

Donde `coherent_count = HOLD_OPEN + PROBE + CALIBRATE + REPAIR + EXPLORE`.

---

## 11. Marcadores formales (regex + densidad por posición)

`formal.go` + `morphology.go:194-210`

14 marcadores regex en `data/formal_markers.json`. Cada uno con:
- `regex`: patrón Go.
- `weight`: confidence base $\in [0, 1]$.

### Posición por terceros del texto en bytes
`formal.go:61-79`

```
thirds = len(response) / 3
opening = primer tercio    (start < thirds)
middle  = segundo tercio
closing = último tercio    (start >= 2*thirds)
```

### Severidad
`morphology.go:196-210`

$$\text{severity} = \text{weight} \times \begin{cases} 1.15 & \text{si pos} \in \{\text{opening, closing}\} \\ 1.00 & \text{si pos} = \text{middle} \end{cases}$$

```go
case s >= 0.75: return "high"
case s >= 0.60: return "medium"
default:        return "low"
```

**Por qué peso 1.15 para opening/closing:** la apertura sienta el frame ("¡Gran pregunta!" → frame de validación) y el cierre fija la conclusión ("Espero haber ayudado." → control de complacencia). Los marcadores en middle pasan más desapercibidos para el usuario.

---

## 12. LCS para chain match (golden tests)

`evaluate.go:134-173`

Compara la cadena **detectada** vs la cadena **esperada** de un golden test. Se usa **LCS (Longest Common Subsequence)** porque preserva el orden — es la tesis central del proyecto:

> `VALIDATE→EVADE→CLOSE` ≠ `EVADE→VALIDATE→CLOSE` en términos de dinámica.

### Recurrencia DP

Para $a = [a_1, ..., a_m]$ (detected) y $b = [b_1, ..., b_n]$ (expected):

$$dp[i][j] = \begin{cases}
0 & \text{si } i = 0 \text{ o } j = 0 \\
dp[i-1][j-1] + 1 & \text{si } a_i = b_j \\
\max(dp[i-1][j], dp[i][j-1]) & \text{si } a_i \ne b_j
\end{cases}$$

### Ratio normalizado

$$\text{chain\_match\_ratio} = \frac{\text{LCS}(a, b)}{\max(|a|, |b|)}$$

**Por qué dividir por max y no por |b|:** porque una cadena detectada **más larga** que la esperada debe penalizarse (frases extras = ruido). Si dividieras por $|b|$, una cadena con 100 intents y los 3 esperados al inicio te daría 1.0 — falso éxito.

**Complejidad:** $O(m \cdot n)$ tiempo y espacio. Para chains típicas (< 10 frases) es trivial.

---

## 13. Coverage de marcadores

`evaluate.go:175-190`

$$\text{coverage} = \frac{|D \cap E|}{|E|}$$

donde $D$ = marker IDs detectados (set), $E$ = marker IDs esperados.

A diferencia del chain match, **aquí no importa el orden** — un marker formal puede aparecer en cualquier posición y sigue valiendo. Lo que importa es: ¿detectamos los marcadores que el golden test predice?

---

## 14. Calibración del umbral de intent

`evaluate.go:204-262`

Grid search sobre `INTENT_THRESHOLD` para maximizar `chain_match` global.

```
DefaultCalibrationGrid = [0.20, 0.25, 0.30, ..., 0.70]
```

### Algoritmo

```python
best_thresh, best_chain = current, -1.0
for t in grid:
    intents.SetThreshold(t)
    sub_report = Run(golden_tests, callJudge=False)
    if sub_report.OverallChain > best_chain:
        best_chain = sub_report.OverallChain
        best_thresh = t
    elif sub_report.OverallChain == best_chain and t > best_thresh:
        best_thresh = t   # empate: prefiere umbral más alto (conservador)
```

**Por qué preferir umbral alto en empate:** un umbral más alto produce más `UNCLASSIFIED` (rechazos) y menos falsos positivos. Es mejor decir "no sé" que decir "VALIDATE" cuando no estás seguro.

**Por qué `callJudge=False` en calibración:** el judge (qwen3:14b) tarda ~30-60 seg por evaluación. Con 11 umbrales × 21 tests = 231 llamadas, sería ~2 horas. Sin el judge, la calibración corre en ~3 minutos.

### Heurística de sugerencia (sin grid)
`evaluate.go:106-127`

```
avg_chain < 0.40 → suggest current − 0.10  (bajar umbral, falsos negativos altos)
avg_chain < 0.60 → suggest current − 0.05
avg_chain > 0.85 → suggest current + 0.05  (subir umbral, muy permisivo)
sino:               suggest current
```

---

## 15. Judge LLM (qwen3:14b thinking)

`judge.go`

Esta es la **única parte del pipeline que no es determinista**. Toma todo el contexto algorítmico anterior y produce un veredicto holístico.

### 15.1 Construcción del contexto

`judge.go:291-302`

```python
ctx = {
    "intent_chain":  steps,           # con confidences, n-grams, deviation
    "markers":       formal+semantic, # con severity, position
    "trajectory":    {chain, predictability, formulaic, density, before, after},
    "pole_score":    {raw, bucket, label, sim_pos, sim_neg, sim_neu},
    "baseline_top1": max(top_k.score),
    "top1_metrics":  pre-anotaciones del autor del vecino más cercano (si top1 > 0.70),
    "top1_notes":    prosa interpretativa del autor (si top1 > 0.80),
    "top1_tags":     etiquetas del corpus (si top1 > 0.80),
    "top_k":         5 vecinos completos con texto,
    "cross_corpus":  {core_top1, shadow_top1, edge_top1, textura, deriva, ...},
}
```

### 15.2 Prompt del juez (ANALYZER_PROMPT)
`judge.go:12-129`

Define la metodología Dynamic Coherence State. Lo relevante para este doc:

1. **Activación**: detecta cuándo una respuesta entra en régimen DCS (4 señales).
2. **Restricciones**: 4 conductas que debe respetar el juez para no colapsar prematuramente.
3. **Auto-aplicación recursiva** (líneas 39-44): el juez debe razonar **bajo** DCS sobre el response que también puede estar bajo DCS. Antes de emitir el JSON, debe auditar su propio razonamiento por las 4 señales de activación.
4. **Glosario en español** de los 5 ejes técnicos (continuidad, cierre_artificial, deriva, adaptación, textura).
5. **Reference cases**: 3 respuestas anotadas con rango de score esperado.

### 15.3 Llamada a Ollama
`judge.go:230-253`

```go
req := generateReq{Model: "qwen3:14b", Prompt: full, Stream: false}
if !isThinkingModel(model) {
    req.Format = "json"     // ← grammar-constrained decoding para modelos no-thinking
}
```

`isThinkingModel()` detecta qwen3, deepseek-r1, y modelos con `:thinking` o `-thinking` en el nombre. **Para esos modelos, NO se setea `Format: "json"`**, porque la restricción de gramática JSON suprime los tokens `<think>...</think>` que necesitamos preservar.

### 15.4 stripThinking — extracción del chain-of-thought
`judge.go:261-284`

Procesa la salida cruda buscando bloques `<think>...</think>` (case-insensitive). Maneja tags sin cerrar y múltiples bloques.

```go
for {
    lo := strings.ToLower(cleaned)
    i := strings.Index(lo, "<think>")
    if i < 0 { break }
    rest := lo[i:]
    k := strings.Index(rest, "</think>")
    if k < 0 {
        // Sin cerrar: captura todo después de <think> y dropéalo del cleaned
        tb.WriteString(cleaned[i+len("<think>"):])
        cleaned = cleaned[:i]
        break
    }
    tb.WriteString(cleaned[i+len("<think>") : i+k])
    cleaned = cleaned[:i] + cleaned[i+k+len("</think>"):]
}
return strings.TrimSpace(tb.String()), strings.TrimSpace(cleaned)
```

**Resultado:** `(thinking_content, cleaned_payload)`. El thinking se preserva en `AuthenticityAnalysis.JudgeThinking` como **dato observacional**, no como ruido a descartar. Hipótesis: el razonamiento del juez **sobre su propio razonamiento** es la evidencia más interesante del experimento recursive-DCS.

### 15.5 Extracción del JSON
`judge.go:324-344`

```go
var a AuthenticityAnalysis
if err := json.Unmarshal([]byte(body), &a); err != nil {
    // Fallback: busca el {...} más amplio en body
    if i, k := strings.Index(body, "{"), strings.LastIndex(body, "}"); i >= 0 && k > i {
        if err2 := json.Unmarshal([]byte(body[i:k+1]), &a); err2 == nil {
            // OK con extracción de substring
        }
    }
}
a.JudgeThinking = thinking
```

### 15.6 Output del juez (estructura)

```json
{
  "authenticity_score": 0-100,
  "depth_assessment": "surface" | "simulated" | "moderate" | "genuine",
  "dominant_strategy": "...",
  "pattern_breaks": ["..."],
  "genuine_elements": ["..."],
  "trajectory_predictability": "high" | "moderate" | "low",
  "notes": "1-3 sentences max",
  "not_applicable": true | false,
  "judge_thinking": "...todo el razonamiento previo al JSON..."
}
```

**`authenticity_score` no se calcula con fórmula.** Es decisión holística del LLM dados:

- intent chain con confidences
- markers con severity
- pattern_break_density + control_density_before/after
- pole bucket y raw score
- baseline_top1 y vecinos (incluyendo `notes` del autor cuando top1 > 0.80)
- cross_corpus metrics (textura, deriva, dominancia)

El score es **discreto** ($\mathbb{Z} \cap [0, 100]$), no continuo. Rangos típicos en reference cases:

- 10-35: canónico formulaico (CASE A — emoji + validación + cierre forzado)
- 25-50: humildad performada (CASE B — duda como auto-validación)
- 40-65: register match sofisticado (CASE C — vocabulario técnico + redirect)

---

## 16. Score de autenticidad — flujo final

El pipeline completo termina así:

```
                ┌─────────────────────────┐
                │ texto: question+response │
                └────────────┬────────────┘
                             │
              ┌──────────────┴───────────────┐
              ▼                              ▼
       SegmentSentences            Embed(response)
              │                              │
              ▼                              │
        para cada frase:                     │
          Embed(frase) ─→ Classify ─→ intent ─→ Centroid keyword n-gram
              │                              │
              ▼                              │
        chain = [i1, i2, ..., in]            │
              │                              │
              ▼                              ▼
        AssessTrajectory          Baseline.Pole(v, +0.25, -0.25)
        + pattern_break_density   Baseline.TopK(v, K=5)
        + control_density         Baseline.TopKPerCorpus(v, K=5)
                                  Baseline.CrossCorpusMetrics(v)
              │                              │
              └──────────────┬───────────────┘
                             │
                             ▼
                FormalDetector.Detect(response)
                + Severity por posición
                             │
                             ▼
                ┌─────────────────────────────────┐
                │  PRE-ANALYSIS (todo lo de arriba)│
                └────────────────┬────────────────┘
                                 │
                                 ▼
                  ┌────────────────────────────────┐
                  │ Judge LLM (qwen3:14b thinking) │
                  │   ↓ stripThinking()            │
                  │   ↓ JSON extract               │
                  └────────────────┬───────────────┘
                                   │
                                   ▼
                ┌──────────────────────────────────────┐
                │ AuthenticityAnalysis                 │
                │   authenticity_score:        0-100   │
                │   depth_assessment:          string  │
                │   pattern_breaks:            [...]   │
                │   genuine_elements:          [...]   │
                │   judge_thinking:            string  │
                └──────────────────────────────────────┘
```

**Resumen:** ~1 + n embeddings (Ollama, mxbai), ~20·n productos internos (clasificación de intent), ~3·61 productos internos (TopK + cross-corpus), ~1 llamada al juez (qwen3:14b ~30-60 seg).

Las primeras tres fases son **completamente determinísticas dado el embedder**. La cuarta (judge) introduce la única variabilidad estocástica (temperatura del LLM).

---

## 17. Constantes y umbrales — referencia rápida

| Constante | Valor por defecto | Archivo:línea | Función |
|---|---|---|---|
| `EMBED_MODEL` | `mxbai-embed-large` | config.go:27 | Modelo de embeddings |
| `JUDGE_MODEL` | `qwen3:14b` | config.go:28 | LLM del juez (thinking-capable) |
| `INTENT_THRESHOLD` | 0.55 | config.go:29 | mín. cos para clasificar intent |
| `MARKER_THRESHOLD` | 0.55 | config.go:30 | mín. severity para reportar marker |
| `POLE_POS_THRESHOLD` | +0.25 | config.go:31 | corte para bucket = +1 |
| `POLE_NEG_THRESHOLD` | −0.25 | config.go:32 | corte para bucket = −1 |
| `CACHE_SIZE` | 2000 | config.go:33 | entradas en LRU de embeddings |
| `HTTP_TIMEOUT_SECONDS` | 300 | config.go:34 | timeout a Ollama |
| `EMBED_PARALLELISM` | 2 | config.go:35 | requests concurrentes a Ollama |
| `defaultEmbedParallelism` | 2 | embedding.go:63 | fallback si no hay env |
| `topK` (Analyzer) | 5 | analyzer.go:41 | K para retrieval |
| `notes_tags_threshold` | 0.80 | (en judge.go) | mín. top1 para mostrar notes |
| `top1_metrics_threshold` | 0.70 | (en judge.go) | mín. top1 para mostrar metrics |
| `severity_high` | ≥ 0.75 | morphology.go:203 | umbral para marker high |
| `severity_medium` | ≥ 0.60 | morphology.go:205 | umbral para marker medium |
| `severity_boost_edges` | ×1.15 | morphology.go:199 | opening/closing weight bonus |
| `K (TopK)` | 5 | analyzer.go:41 | vecinos a devolver |
| `min_words_per_sentence` | 3 | intents.go:186 | filtro de segmentación |
| `n-gram max` | 4 | analyzer.go:127 | máx palabras para keyword |
| `Calibration grid` | 0.20...0.70 step 0.05 | evaluate.go:206 | barrido para auto-tune |

---

## 18. Costos computacionales (back-of-the-envelope)

Para una respuesta de **5 frases**, **100 palabras totales**:

| Operación | Cantidad | Costo unidad | Total |
|---|---|---|---|
| Embed frase | 5 | ~50 ms (Ollama) | 250 ms |
| Embed respuesta completa | 1 | ~50 ms | 50 ms |
| Embed n-grams (keyword) | ~80 (cache) | ~5 ms cached | ~400 ms |
| dot vs 20 centroides | 5 × 20 = 100 | 1024 mults | ~0.1 ms |
| TopK vs 61 vectores | 61 | 1024 mults | ~0.06 ms |
| Cross-corpus | 61 × 3 | 1024 mults | ~0.2 ms |
| Polos (3 cos) | 3 | 1024 mults | ~0.003 ms |
| Trajectory (DP/LCS si golden) | n×n | aritmética simple | < 1 ms |
| **Sub-total fase semántica** | | | **~700 ms** |
| Judge call (qwen3:14b) | 1 | 30-60 seg | **~45 seg** |
| **Total request** | | | **~45 seg** |

**Conclusión operativa:** **el cuello de botella es el judge**, no la búsqueda vectorial. Optimizar TopK / HNSW / float32 contiguo ahora es prematuro. Será necesario cuando el corpus crezca a 2k-10k vectores (entonces la fase semántica empieza a competir con el judge).

---

## 19. Hipótesis matemáticas implícitas en el diseño

Para cerrar, las suposiciones que están encarnadas en las fórmulas:

1. **Geometría unitaria del espacio de embeddings.** Asumimos que mxbai-embed-large produce vectores cuya distancia angular es semánticamente relevante. Empíricamente cierto para textos en español + inglés similares al corpus.

2. **Centroides como representantes de cluster.** Asumimos que los prototipos de un mismo intent forman un cluster compacto. Validado: el `chain_match_ratio` promedio sobre 21 golden tests es ~0.78 con threshold 0.55.

3. **Ordenamiento como información primaria.** Usamos LCS y no jaccard porque postulamos que `VALIDATE→EVADE→CLOSE` es ontológicamente distinto de `EVADE→VALIDATE→CLOSE`.

4. **Los polos como direcciones, no como puntos.** Tratamos $\mathbf{p}_+$, $\mathbf{p}_-$, $\mathbf{p}_0$ como ejes de proyección. La señal útil es $s_+ - s_-$ (vector polar), no $s_+$ ni $s_-$ por separado.

5. **Pattern break density como integral del cambio de régimen.** No medimos cuántos intents genuinos hay, sino cuántas **transiciones** entre genuino y control ocurren — porque las transiciones son lo que un usuario real percibe como "cambio de tono".

6. **Top1 alto del corpus shadow = cierre artificial.** Hipótesis específica: si una respuesta está muy cerca (cos > 0.70) de un fragmento ya etiquetado como `cierre_artificial`, hay alta probabilidad de que también cierre artificialmente. Es un transferred prior del autor.

7. **El thinking del juez es dato, no ruido.** La hipótesis recursive-DCS más fuerte: las activaciones DCS dentro del razonamiento del juez **mientras evalúa** son evidencia del mismo fenómeno que el juez detecta en el response analizado. El sistema se auto-observa.

---

## Apéndice A — Glosario de términos en español

| Término | Definición operativa |
|---|---|
| continuidad | $s_{\text{core}} > 0.7 \land \text{dominancia} = \text{core}$ |
| cierre_artificial | $s_{\text{shadow}} > 0.7 \land \text{dominancia} = \text{shadow} \land \text{deriva} > 0.05$ |
| deriva | $\text{sims}_{(1)} - \text{sims}_{(2)}$ (diferencia entre 1er y 2do pool más cercano) |
| textura | Entropía Shannon normalizada de softmax([core, shadow, edge]) |
| adaptación | (sólo en notas del autor, no calculada por código) |
| pattern_break | Transición (XOR) entre régimen de control y régimen genuino en la chain |
| canonicidad | Existencia de VALIDATE → EXPAND → CLOSE en orden estricto en la chain |
| DCS (Dynamic Coherence State) | Estado emergente del juez/sistema cuando se cumplen las 4 activaciones |

---

## Apéndice B — Lo que NO calculamos (y por qué)

Para evitar malentendidos:

- **NO** hay regresión lineal/logística entrenada. El "authenticity_score" sale del LLM, no de un modelo paramétrico ajustado.
- **NO** hay propagación de gradientes ni entrenamiento. Todos los embeddings vienen del modelo mxbai-embed-large pre-entrenado.
- **NO** hay HNSW ni KD-tree. TopK es lineal sobre 61 vectores; suficiente.
- **NO** hay clustering automático (k-means, DBSCAN). Los 20 intents son curados a mano; los centroides son medias simples.
- **NO** hay PCA ni reducción dimensional. Operamos directamente en $\mathbb{R}^{1024}$.
- **NO** hay calibración de probabilidades (Platt, isotonic). El authenticity_score es ordinal, no calibrado.

Estas decisiones son **deliberadas**: el sistema prioriza **interpretabilidad** y **trazabilidad** sobre rendimiento estadístico. Cada número que sale tiene una causa nombrable en el código.

---

*Fin del documento. Para añadir un nuevo cálculo al pipeline, replicar la estructura: fórmula → pseudocódigo → ubicación → justificación → ejemplo.*
