# DCS-Gate — Propuesta de Migración Selectiva a Zig (vía cgo)

> "Matemáticamente Zig es un dios y en algunas cosas gana por precisión y bajo nivel"
> — donde realmente paga la mudanza, y donde **NO** vale la pena tocar nada.

Esta nota responde dos preguntas:

1. **¿Qué partes del DCS-Gate v8 son candidatas a migrar a Zig?**
2. **¿Cómo se conecta Zig con Go vía cgo sin romper el build actual?**

No es un plan de "reescribir todo en Zig". Es un mapa quirúrgico: cuatro
funciones calientes con tracking claro de ROI, tres descartadas con razón.

---

## 0. Análisis del hot path actual (v8.2, leyendo el código)

Antes de decidir si migrar, midamos dónde está el costo real por cada `/auth`:

| Operación | Qué hace | Coste real | ¿Zig ayuda? |
|---|---|---:|---|
| **Dot products en TopK** | 8957 vectores × 1024d × ~5 frases × 4 pools = **180M multiplicaciones** | 30–80 ms | **Sí, 4–8×** con `@Vector(8, f32)` AVX2 |
| **Llamada al juez (LLM)** | wizardlm2:7b o qwen2.5:7b en Ollama | 1500–3000 ms | **NO** (es I/O y compute del LLM, no nuestro) |
| **Llamada al embed (mxbai)** | 1024d por frase | 50–150 ms | **NO** (proceso aparte en Ollama) |
| **Regex de markers formales** | 14 patterns sobre ~5–15 frases | 1–3 ms | No (negligible) |
| **AnalyzeIntra n-grams** | unigram/bigram entropy sobre tokens UTF-8 | 1–5 ms | No (string handling, Go más expresivo) |
| **Reporting markdown** | concat strings | <1 ms | No |

**Conclusión cuantitativa**: latencia total típica de `/auth` = ~1700–3300 ms.
De eso, ~30–80 ms es nuestro cómputo (dot products + heap). Si migramos eso a
Zig con SIMD obtenemos 5×, lo que baja a ~6–16 ms. **Mejora total observable
en p50 latency: ~12% (de 2000ms a 1760ms).** Para un usuario haciendo análisis
manual, indistinguible.

**Donde Zig SÍ rompe velocidad**:
- **Modo batch** (evaluar 100 respuestas en paralelo sin tocar al juez):
  los 30–80 ms × 100 = 3–8 s se van a 0.5–1.5 s. Speedup real **5–8×**.
- **Corpus a 50k+ vectores**: con 8957 actuales el costo es lineal pero
  pequeño. A 50k vectores los 30–80 ms se vuelven 200–500 ms → ahí Zig
  baja a 25–60 ms. Speedup **4–8× con impacto real**.

**Donde Zig NO ayuda**:
- En el hot path con juez activo, el LLM domina ~94% de la latencia.
- En `/score` o `/auth?judge=false` el ahorro es de 30 ms sobre 200 ms total
  → ~15% que ningún usuario nota.

**Recomendación honesta**: con corpus actual (8957 vectores) y uso interactivo
con juez, **no migres todavía**. Implementa primero las 3 optimizaciones Go
puras (sección 7) y deja la infraestructura Zig (Fases A + B sin sustitución)
lista para el día que (a) hagas evaluación batch, o (b) el corpus crezca a
50k+. Esa es la decisión defendible al instante.

---

## 1. Tabla de candidatos

| # | Símbolo (Go) | Archivo | Tipo de carga | ¿Vale la pena Zig? | Speedup esperado | Riesgo |
|---|---|---|---|---|---|---|
| 1 | `cosineSim` / loops de similitud | `baseline.go`, `embed.go` | dot product en vectores `float32` de 1024 dims, batch grande | **Sí (alto ROI)** | 4–8× con `@Vector(8, f32)` AVX2 / 2–3× sin SIMD | Bajo |
| 2 | `BuildCentroids` (suma + normalización) | `baseline.go` | media de N vectores 1024d + normalize | **Sí (medio)** | 3–5× | Bajo |
| 3 | `topKFromEntries` | `baseline.go` | similitud + heap top-k | **Sí (medio)** | 2–4× si va junto con (1) | Medio (heap en cgo) |
| 4 | `Pole` (3 cosenos por entrada) | `baseline.go` | 3 dot products + clasificación | **Sí (bajo, pero gratis)** si ya migraste (1) | 2× | Bajo |
| 5 | `chainMatchRatio` (LCS) | `intents.go` | DP O(n·m) sobre cadenas cortas (≤30) | **No** | <1.2× | Alto en mantenibilidad |
| 6 | `AnalyzeIntra` (n-grams, entropía) | `analyzer.go` | strings UTF-8, manipulación textual | **No** | nulo o negativo | Alto (Unicode en Zig es manual) |
| 7 | `embed_corpus.py` / `compute_poles.py` | Python | I/O + JSON + 1 cosine | **No** | I/O bound | N/A |

> Regla de oro: **migra solo lo que hace dot products masivos en `float32`.**
> Lo demás (parsers, strings, JSON, control flow del analizador) se queda en Go.

---

## 2. Plan en orden de ROI decreciente

### Fase A — `dcs_math.zig` (vector kernels)

Una sola unidad de compilación Zig que expone, vía `export "C"`, cuatro
funciones planas y libres de allocator dinámico:

```zig
// dcs_math.zig (build target: -dynamic, -fPIC, -O ReleaseFast)

const std = @import("std");

// Producto punto puro float32, vectorizado
export fn dcs_dot_f32(a: [*]const f32, b: [*]const f32, n: usize) f32 {
    const V = @Vector(8, f32);
    var i: usize = 0;
    var acc: V = @splat(0);
    while (i + 8 <= n) : (i += 8) {
        const va: V = a[i..][0..8].*;
        const vb: V = b[i..][0..8].*;
        acc += va * vb;
    }
    var sum: f32 = 0;
    inline for (0..8) |k| sum += acc[k];
    while (i < n) : (i += 1) sum += a[i] * b[i];
    return sum;
}

// Cosine batch: query (1×dim) contra base (nblocks×dim), out (nblocks)
export fn dcs_cosine_batch_f32(
    query: [*]const f32,
    base:  [*]const f32,
    nblocks: usize,
    dim:   usize,
    qnorm: f32,            // norma precomputada del query
    bnorms: [*]const f32,  // normas precomputadas de cada bloque base
    out:   [*]f32,
) void {
    var b: usize = 0;
    while (b < nblocks) : (b += 1) {
        const dot = dcs_dot_f32(query, base + b * dim, dim);
        const denom = qnorm * bnorms[b];
        out[b] = if (denom > 1e-12) dot / denom else 0;
    }
}

// Centroide normalizado: media de nblocks vectores dim → out (dim)
export fn dcs_centroid_f32(
    base: [*]const f32,
    nblocks: usize,
    dim: usize,
    out: [*]f32,
) void {
    @memset(out[0..dim], 0);
    var b: usize = 0;
    while (b < nblocks) : (b += 1) {
        var i: usize = 0;
        while (i < dim) : (i += 1) out[i] += base[b * dim + i];
    }
    if (nblocks == 0) return;
    const inv: f32 = 1.0 / @as(f32, @floatFromInt(nblocks));
    var i: usize = 0;
    var sumsq: f32 = 0;
    while (i < dim) : (i += 1) {
        out[i] *= inv;
        sumsq += out[i] * out[i];
    }
    const norm = @sqrt(sumsq);
    if (norm > 1e-12) {
        i = 0;
        while (i < dim) : (i += 1) out[i] /= norm;
    }
}

// Top-K: selecciona los K mayores de scores[n], devuelve idx + sim
export fn dcs_topk_f32(
    scores: [*]const f32,
    n: usize,
    k: usize,
    out_idx: [*]i32,
    out_sim: [*]f32,
) void {
    // ... mini-heap de tamaño K (omitido por brevedad; ~40 líneas)
}
```

**Build:**

```bash
zig build-lib dcs_math.zig -dynamic -fPIC -O ReleaseFast \
    -femit-bin=libdcs_math.so
```

### Fase B — Wrapper Go con cgo

```go
// dcs_math_zig.go  (build tag: !no_zig)
//go:build !no_zig

package main

/*
#cgo CFLAGS: -I${SRCDIR}/zig
#cgo LDFLAGS: -L${SRCDIR}/zig -ldcs_math -Wl,-rpath,${SRCDIR}/zig

#include <stdint.h>
extern float dcs_dot_f32(const float* a, const float* b, size_t n);
extern void  dcs_cosine_batch_f32(const float* q, const float* base,
                                  size_t nb, size_t dim, float qnorm,
                                  const float* bnorms, float* out);
extern void  dcs_centroid_f32(const float* base, size_t nb, size_t dim, float* out);
extern void  dcs_topk_f32(const float* scores, size_t n, size_t k,
                          int32_t* out_idx, float* out_sim);
*/
import "C"

import "unsafe"

func zigDot(a, b []float32) float32 {
    if len(a) == 0 || len(a) != len(b) {
        return 0
    }
    return float32(C.dcs_dot_f32(
        (*C.float)(unsafe.Pointer(&a[0])),
        (*C.float)(unsafe.Pointer(&b[0])),
        C.size_t(len(a)),
    ))
}

func zigCosineBatch(query []float32, base [][]float32, qnorm float32, bnorms []float32) []float32 {
    n, dim := len(base), len(query)
    if n == 0 {
        return nil
    }
    flat := make([]float32, n*dim)        // pack a contiguous slab
    for i, v := range base { copy(flat[i*dim:], v) }
    out := make([]float32, n)
    C.dcs_cosine_batch_f32(
        (*C.float)(unsafe.Pointer(&query[0])),
        (*C.float)(unsafe.Pointer(&flat[0])),
        C.size_t(n), C.size_t(dim),
        C.float(qnorm),
        (*C.float)(unsafe.Pointer(&bnorms[0])),
        (*C.float)(unsafe.Pointer(&out[0])),
    )
    return out
}
```

Y un fallback puro Go con build tag opuesto:

```go
//go:build no_zig

package main
func zigDot(a, b []float32) float32 { /* implementación Go actual */ }
// idem para los otros
```

Así `go build -tags no_zig` sigue compilando sin Zig instalado (CI mínimo,
máquinas sin la lib).

### Fase C — Sustitución en call sites

Solo dos archivos cambian:

1. `baseline.go` — `Pole()` y `topKFromEntries()` llaman a `zigCosineBatch` + `zigTopK`
2. `embed.go` (si existe el batch) — usa `zigCosineBatch` cuando hay >100 vectores

El resto del código (intents, analyzer, judge, report, formal) **no se toca**.

### Fase D — Empaquetado

- `zig/dcs_math.zig` + `zig/build.sh` en el repo
- Binary `libdcs_math.so` a generarse en CI antes del `go build`
- README añade un párrafo: "Zig 0.13+ requerido (opcional, fallback con `-tags no_zig`)"
- `embed_all.sh` añade un step `bash zig/build.sh` justo antes del `go build`

---

## 3. Por qué NO migrar otras cosas

| Función | Razón |
|---|---|
| `chainMatchRatio` (LCS) | Cadenas de ≤30 strings cortos. Go ya hace 0.5µs por chain. Zig no compensa el costo cgo (~150ns por call). |
| `AnalyzeIntra` n-grams | Manipulación de strings UTF-8 con `strings.Fields`, `unicode.IsPunct`. Zig no tiene libstd Unicode equivalente. Reescribir a mano sería peor que el código Go. |
| Parser JSON (`encoding/json`) | I/O bound y altamente optimizado en Go. Zig no gana nada. |
| HTTP server (`evaluate.go`, `main.go`) | Idem — el bottleneck es la red, no el CPU. |
| Python (`embed_corpus`, `compute_poles`) | I/O bound + un solo cosine grande al final. Si quisieras acelerar `compute_poles.py`, mejor un binding Python→`libdcs_math.so` con ctypes que migrar todo. |

---

## 4. Trade-offs honestos

**Ganamos**
- 4–8× en el hot path de polos y top-k (lo que se llama por cada `/score`)
- Precisión idéntica (Zig usa IEEE 754 igual que Go)
- Tamaño de binario menor para la lib `.so` que la rama equivalente Go con assembly inline

**Pagamos**
- Build chain con dependencia opcional de Zig 0.13+
- ~200 líneas extra de Zig + ~80 de wrapper Go
- cgo introduce un overhead fijo (~80–150ns) por llamada — por eso solo migramos funciones BATCH, no `dcs_dot` aislado para una pareja

**No tocamos** (y por qué sí queda en Go)
- La lógica de coherencia (intents, transiciones, judge) es semántica, no aritmética
- El análisis textual mete UTF-8 y heurísticas de cierre/ruptura, donde Go es más expresivo
- El reporting es markdown — no hay nada que ganar

---

## 5. Plan ejecución sugerido (3 PRs incrementales)

1. **PR-zig-1**: Añade `zig/dcs_math.zig` + `zig/build.sh` + wrappers cgo + tests de paridad numérica (compara `zigDot` vs `goDot` con tolerancia 1e-6). El binario actual no usa nada todavía. Build tag `!no_zig` activado por defecto.
2. **PR-zig-2**: Sustituye `Pole()` y `BuildCentroids` por las versiones zig. Benchmark comparativo en el commit.
3. **PR-zig-3**: Sustituye `topKFromEntries`. Benchmark.

Cada PR puede revertirse sin romper ningún otro componente.

---

## 6. Decisión recomendada

**Hacerlo solo si:**
- Vas a tener corpus >5k bloques (con 60 bloques el speedup es invisible)
- Vas a llamar `/score` más de 10 req/s sostenidos
- Te importa la latencia p99 del endpoint, no el throughput agregado

**No hacerlo (todavía) si:**
- Sigues iterando rápido en la taxonomía y los corpus
- Solo es una herramienta de análisis manual con cargas puntuales

Mi recomendación con el corpus actual (60 bloques): **mantén el plan descrito
en este documento como `ZIG_MIGRATION.md` en el repo**, pero implementa solo
**Fase A + Fase B** (kernels + wrapper, sin sustitución todavía). Eso te deja
la infraestructura lista, los benchmarks comparativos publicados, y la
sustitución la haces el día que el corpus llegue a la escala donde duele.
