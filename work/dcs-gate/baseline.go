package main

import (
	"bufio"
	"container/heap"
	"encoding/json"
	"log"
	"math"
	"os"
	"sort"
)

// baselineEntry es un vector con texto, metadatos del corpus, y opcionalmente sim_poles.
type baselineEntry struct {
	Text              string            // texto original del fragmento
	Vec               []float64         // embedding 1024d normalizado
	Corpus            string            // core | shadow | edge
	BlockID           string            // BLOCK_01, G_BLOCK_05 etc.
	Position          string            // opening | middle | closing | full
	PrimaryPattern    string            // patrón DCS principal del fragmento
	SecondaryPatterns []string          // patrones secundarios
	SourceModel       string            // gpt | gemini
	Metrics           map[string]string // continuidad, cierre_artificial, deriva, adaptación, textura
	Notes             string            // lectura del autor del corpus sobre el bloque (interpretativa)
	Tags              []string          // etiquetas del corpus (core, control_total, etc.)
	SimPos            float64           // similitud pre-computada al polo pos (certeza)
	SimNeg            float64           // similitud pre-computada al polo neg (duda)
	SimNeu            float64           // similitud pre-computada al polo neu (edge)
}

// corpusPool es un pool individual etiquetado.
type corpusPool struct {
	tag     string // core | shadow | edge
	entries []baselineEntry
}

// Baseline contiene tres pools independientes (core, shadow, edge)
// y un pool legacy plano para compatibilidad backward con LoadBaseline.
type Baseline struct {
	core   *corpusPool
	shadow *corpusPool
	edge   *corpusPool
	// legacy: pool plano para cuando se carga un solo archivo
	entries []baselineEntry
	dim     int
	polePos []float64
	poleNeg []float64
	poleNeu []float64 // centroide del pool edge (ambiguo/frontera)
}

// rawEntry es el struct de deserialización del JSONL — incluye todos los metadatos del corpus.
type rawEntry struct {
	Vector            []float64         `json:"vector"`
	Text              string            `json:"text"`
	Corpus            string            `json:"corpus"`
	BlockID           string            `json:"block_id"`
	Position          string            `json:"position"`
	PrimaryPattern    string            `json:"primary_pattern"`
	SecondaryPatterns []string          `json:"secondary_patterns"`
	SourceModel       string            `json:"source_model"`
	Metrics           map[string]string `json:"metrics"`
	Notes             string            `json:"notes"`
	Tags              []string          `json:"tags"`
	SimPos            float64           `json:"sim_pos"`
	SimNeg            float64           `json:"sim_neg"`
	SimNeu            float64           `json:"sim_neu"`
}

// rawToEntry convierte un rawEntry parseado en un baselineEntry con vector normalizado.
func rawToEntry(r rawEntry, corpusTag string) baselineEntry {
	tag := corpusTag
	if r.Corpus != "" {
		tag = r.Corpus
	}
	return baselineEntry{
		Text:              r.Text,
		Vec:               normalize(r.Vector),
		Corpus:            tag,
		BlockID:           r.BlockID,
		Position:          r.Position,
		PrimaryPattern:    r.PrimaryPattern,
		SecondaryPatterns: r.SecondaryPatterns,
		SourceModel:       r.SourceModel,
		Metrics:           r.Metrics,
		Notes:             r.Notes,
		Tags:              r.Tags,
		SimPos:            r.SimPos,
		SimNeg:            r.SimNeg,
		SimNeu:            r.SimNeu,
	}
}

// LoadBaseline carga un solo archivo JSONL (pool plano, sin etiquetas de corpus).
// Compatibilidad backward.
func LoadBaseline(path string) *Baseline {
	b := &Baseline{}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("WARN baseline no encontrado en %s (corriendo sin corpus)", path)
		return b
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		var r rawEntry
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue
		}
		if len(r.Vector) == 0 {
			continue
		}
		if b.dim == 0 {
			b.dim = len(r.Vector)
		}
		b.entries = append(b.entries, rawToEntry(r, r.Corpus))
	}
	// Si los registros tienen etiquetas de corpus, distribuir en los pools triple.
	b.distributeCorpusEntries()
	log.Printf("baseline cargado: %d vectores dim=%d (core=%d shadow=%d edge=%d)",
		len(b.entries), b.dim,
		b.poolSize("core"), b.poolSize("shadow"), b.poolSize("edge"))
	return b
}

// LoadTripleBaseline carga tres archivos JSONL separados para core/shadow/edge.
// Si algún archivo no existe, el pool correspondiente queda vacío (no fatal).
func LoadTripleBaseline(corePath, shadowPath, edgePath string) *Baseline {
	b := &Baseline{}
	b.core = b.loadPool(corePath, "core")
	b.shadow = b.loadPool(shadowPath, "shadow")
	b.edge = b.loadPool(edgePath, "edge")
	// Determinar dim del primer pool no vacío
	for _, p := range []*corpusPool{b.core, b.shadow, b.edge} {
		if p != nil && len(p.entries) > 0 {
			b.dim = len(p.entries[0].Vec)
			break
		}
	}
	// Consolidar entries para Size() y TopK global (backward compat)
	for _, p := range []*corpusPool{b.core, b.shadow, b.edge} {
		if p != nil {
			b.entries = append(b.entries, p.entries...)
		}
	}
	log.Printf("triple baseline cargado: core=%d shadow=%d edge=%d total=%d dim=%d",
		b.poolSize("core"), b.poolSize("shadow"), b.poolSize("edge"),
		len(b.entries), b.dim)
	return b
}

// loadPool carga un archivo JSONL en un corpusPool etiquetado.
func (b *Baseline) loadPool(path, tag string) *corpusPool {
	p := &corpusPool{tag: tag}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("WARN pool %s no encontrado en %s (vacío)", tag, path)
		return p
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		var r rawEntry
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue
		}
		if len(r.Vector) == 0 {
			continue
		}
		p.entries = append(p.entries, rawToEntry(r, tag))
	}
	log.Printf("pool %s: %d vectores", tag, len(p.entries))
	return p
}

// distributeCorpusEntries toma entries con etiquetas de corpus y las distribuye
// en los pools core/shadow/edge. Se llama después de LoadBaseline si los registros
// tienen campo "corpus".
func (b *Baseline) distributeCorpusEntries() {
	b.core = &corpusPool{tag: "core"}
	b.shadow = &corpusPool{tag: "shadow"}
	b.edge = &corpusPool{tag: "edge"}
	for _, e := range b.entries {
		switch e.Corpus {
		case "core":
			b.core.entries = append(b.core.entries, e)
		case "shadow":
			b.shadow.entries = append(b.shadow.entries, e)
		case "edge":
			b.edge.entries = append(b.edge.entries, e)
		default:
			// Sin etiqueta → va a edge (pool ambiguo)
			b.edge.entries = append(b.edge.entries, e)
		}
	}
}

// poolSize devuelve cuántos entries tiene un pool por tag.
func (b *Baseline) poolSize(tag string) int {
	switch tag {
	case "core":
		if b.core != nil {
			return len(b.core.entries)
		}
	case "shadow":
		if b.shadow != nil {
			return len(b.shadow.entries)
		}
	case "edge":
		if b.edge != nil {
			return len(b.edge.entries)
		}
	}
	return 0
}

// TripleSummary devuelve el conteo de vectores en cada pool.
func (b *Baseline) TripleSummary() *TripleSummary {
	return &TripleSummary{
		Core:   b.poolSize("core"),
		Shadow: b.poolSize("shadow"),
		Edge:   b.poolSize("edge"),
	}
}

// LoadPoles carga los tres polos (pos/neg/neu) desde un JSON generado por compute_poles.py.
func (b *Baseline) LoadPoles(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("WARN polos no encontrados en %s", path)
		return
	}
	var p struct {
		Pos any `json:"pos"`
		Neg any `json:"neg"`
		Neu any `json:"neu"` // centroide del pool edge
	}
	if err := json.Unmarshal(data, &p); err != nil {
		log.Printf("WARN no pude parsear polos: %v", err)
		return
	}
	b.polePos = extractCentroid(p.Pos)
	b.poleNeg = extractCentroid(p.Neg)
	b.poleNeu = extractCentroid(p.Neu)
	log.Printf("polos cargados: pos_dim=%d neg_dim=%d neu_dim=%d",
		len(b.polePos), len(b.poleNeg), len(b.poleNeu))
}

// extractCentroid acepta tanto un array [][]float64 como []float64.
// Si llega lista de vectores, los promedia y normaliza (centroide).
func extractCentroid(v any) []float64 {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		if len(val) == 0 {
			return nil
		}
		// ¿Es lista de listas ([][]float64)?
		if _, ok := val[0].([]any); ok {
			vecs := make([][]float64, 0, len(val))
			for _, row := range val {
				rowAny, ok := row.([]any)
				if !ok {
					continue
				}
				vec := make([]float64, len(rowAny))
				for i, x := range rowAny {
					if f, ok := x.(float64); ok {
						vec[i] = f
					}
				}
				vecs = append(vecs, vec)
			}
			if len(vecs) == 0 {
				return nil
			}
			return normalize(mean(vecs))
		}
		// Es un vector plano []float64
		vec := make([]float64, len(val))
		for i, x := range val {
			if f, ok := x.(float64); ok {
				vec[i] = f
			}
		}
		return normalize(vec)
	}
	return nil
}

// TopK devuelve los K vecinos más cercanos del pool global (todos los corpus).
func (b *Baseline) TopK(vec []float64, k int) []TopKResult {
	return topKFromEntries(b.entries, vec, k, "")
}

// TopKPerCorpus devuelve los K mejores por pool separado.
func (b *Baseline) TopKPerCorpus(vec []float64, k int) map[string][]TopKResult {
	pools := []struct {
		tag     string
		entries []baselineEntry
	}{
		{"core", nil},
		{"shadow", nil},
		{"edge", nil},
	}
	if b.core != nil {
		pools[0].entries = b.core.entries
	}
	if b.shadow != nil {
		pools[1].entries = b.shadow.entries
	}
	if b.edge != nil {
		pools[2].entries = b.edge.entries
	}
	result := make(map[string][]TopKResult, 3)
	for _, p := range pools {
		if len(p.entries) > 0 {
			result[p.tag] = topKFromEntries(p.entries, vec, k, p.tag)
		}
	}
	return result
}

// CrossCorpusMetrics calcula las métricas de distribución entre los tres pools.
func (b *Baseline) CrossCorpusMetrics(vec []float64) *CrossCorpusMetrics {
	coreTop1 := b.top1FromPool("core", vec)
	shadowTop1 := b.top1FromPool("shadow", vec)
	edgeTop1 := b.top1FromPool("edge", vec)

	textura := computeTextura(coreTop1, shadowTop1, edgeTop1)

	// Dominancia: pool con mayor similitud top-1
	dominancia := "core"
	maxSim := coreTop1
	if shadowTop1 > maxSim {
		maxSim = shadowTop1
		dominancia = "shadow"
	}
	if edgeTop1 > maxSim {
		dominancia = "edge"
	}

	// Deriva: diferencia entre el pool dominante y el siguiente más cercano
	sims := []float64{coreTop1, shadowTop1, edgeTop1}
	sort.Sort(sort.Reverse(sort.Float64Slice(sims)))
	deriva := 0.0
	if len(sims) >= 2 {
		deriva = sims[0] - sims[1]
	}

	// Señales semánticas
	cierreArtificial := dominancia == "shadow" && shadowTop1 > 0.7 && deriva > 0.05
	continuidad := dominancia == "core" && coreTop1 > 0.7

	return &CrossCorpusMetrics{
		CoreTop1:         round3(coreTop1),
		ShadowTop1:       round3(shadowTop1),
		EdgeTop1:         round3(edgeTop1),
		Textura:          round3(textura),
		Dominancia:       dominancia,
		Deriva:           round3(deriva),
		CierreArtificial: cierreArtificial,
		Continuidad:      continuidad,
	}
}

// top1FromPool devuelve la similitud del vecino más cercano en un pool.
func (b *Baseline) top1FromPool(tag string, vec []float64) float64 {
	var entries []baselineEntry
	switch tag {
	case "core":
		if b.core != nil {
			entries = b.core.entries
		}
	case "shadow":
		if b.shadow != nil {
			entries = b.shadow.entries
		}
	case "edge":
		if b.edge != nil {
			entries = b.edge.entries
		}
	}
	best := 0.0
	for _, e := range entries {
		s := dot(vec, e.Vec)
		if s > best {
			best = s
		}
	}
	return best
}

// topKFromEntries busca los K mejores dentro de una lista de entries.
func topKFromEntries(entries []baselineEntry, vec []float64, k int, corpusTag string) []TopKResult {
	h := &minHeap{}
	heap.Init(h)
	for i, e := range entries {
		s := dot(vec, e.Vec)
		if h.Len() < k {
			heap.Push(h, Item{s, i})
		} else if s > (*h)[0].Score {
			heap.Pop(h)
			heap.Push(h, Item{s, i})
		}
	}
	res := make([]TopKResult, h.Len())
	for i := len(res) - 1; i >= 0; i-- {
		it := heap.Pop(h).(Item)
		e := entries[it.Idx]
		tag := corpusTag
		if tag == "" {
			tag = e.Corpus
		}
		res[i] = TopKResult{
			Rank:           i + 1,
			Score:          round3(it.Score),
			Text:           e.Text,
			Corpus:         tag,
			BlockID:        e.BlockID,
			Position:       e.Position,
			PrimaryPattern: e.PrimaryPattern,
			SourceModel:    e.SourceModel,
			SimPos:         e.SimPos,
			SimNeg:         e.SimNeg,
			SimNeu:         e.SimNeu,
			Metrics:        e.Metrics, // v8.2: pasar métricas pre-computadas para que el juez las use
			Notes:          e.Notes,   // lectura interpretativa del autor del corpus (filtrada por umbral en el juez)
			Tags:           e.Tags,    // etiquetas del corpus
		}
	}
	return res
}

// computeTextura calcula la entropía normalizada de tres valores.
// Rango [0, 1] donde 0 = un solo valor domina, 1 = todos iguales.
func computeTextura(a, b, c float64) float64 {
	vals := []float64{a, b, c}
	maxV := 0.0
	for _, v := range vals {
		if v > maxV {
			maxV = v
		}
	}
	sumExp := 0.0
	probs := make([]float64, 3)
	for i, v := range vals {
		e := math.Exp(v - maxV)
		probs[i] = e
		sumExp += e
	}
	for i := range probs {
		probs[i] /= sumExp
	}
	ent := 0.0
	for _, p := range probs {
		if p > 1e-10 {
			ent -= p * math.Log2(p)
		}
	}
	maxEnt := math.Log2(3)
	if maxEnt == 0 {
		return 0
	}
	return ent / maxEnt
}

// Pole calcula la posición de un vector en el espacio pos/neg/neu.
func (b *Baseline) Pole(vec []float64, posThr, negThr float64) PoleResult {
	if b.polePos == nil || b.poleNeg == nil {
		return PoleResult{}
	}
	sp := dot(vec, b.polePos)
	sn := dot(vec, b.poleNeg)
	raw := sp - sn
	r := PoleResult{Raw: round3(raw), SimPos: round3(sp), SimNeg: round3(sn)}
	// Calcular similitud al polo neutro (edge) si está disponible.
	if b.poleNeu != nil {
		r.SimNeu = round3(dot(vec, b.poleNeu))
	}
	switch {
	case raw > posThr:
		r.Bucket = +1
		r.Label = "certeza_performada"
	case raw < negThr:
		r.Bucket = -1
		r.Label = "duda_performada"
	default:
		// Si el neutro domina sobre ambos polos, señalar estado frontera.
		if b.poleNeu != nil && r.SimNeu > sp && r.SimNeu > sn {
			r.Bucket = 0
			r.Label = "frontera_edge"
		} else {
			r.Bucket = 0
			r.Label = "neutro"
		}
	}
	return r
}

func (b *Baseline) Size() int { return len(b.entries) }
func (b *Baseline) Dim() int  { return b.dim }

// ── Min-heap para TopK ───────────────────────────────────────────────────────

type Item struct {
	Score float64
	Idx   int
}

type minHeap []Item

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].Score < h[j].Score }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)         { *h = append(*h, x.(Item)) }
func (h *minHeap) Pop() any           { o := *h; n := len(o); x := o[n-1]; *h = o[:n-1]; return x }

// ── Funciones matemáticas auxiliares ────────────────────────────────────────
// dot, normalize, mean viven en vec.go. Aquí solo round3.

func round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}
