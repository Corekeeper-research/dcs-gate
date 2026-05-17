package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

type Embedder struct {
	cfg    Config
	client *http.Client
	cache  *LRU
}

func NewEmbedder(cfg Config) *Embedder {
	return &Embedder{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
		cache:  newLRU(cfg.CacheSize),
	}
}

type embedReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResp struct {
	Embedding []float64 `json:"embedding"`
}

// Get devuelve el embedding normalizado y si vino del cache.
func (e *Embedder) Get(text string) ([]float64, bool, error) {
	if v, ok := e.cache.Get(text); ok {
		return v, true, nil
	}
	body, _ := json.Marshal(embedReq{Model: e.cfg.EmbedModel, Prompt: text})
	resp, err := e.client.Post(e.cfg.OllamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, false, errors.New("ollama embed status " + resp.Status)
	}
	var er embedResp
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, false, err
	}
	if len(er.Embedding) == 0 {
		return nil, false, errors.New("ollama returned empty embedding")
	}
	vec := normalize(er.Embedding)
	e.cache.Put(text, vec)
	return vec, false, nil
}

// defaultEmbedParallelism: requests concurrentes a Ollama por defecto.
// Alineado con OLLAMA_NUM_PARALLEL=2 recomendado para CPU de 2 cores.
// Override vía env EMBED_PARALLELISM.
const defaultEmbedParallelism = 2

// GetMany pide varios embeddings en paralelo con límite de concurrencia.
// Errores individuales se reflejan como nil en la posición correspondiente,
// preservando el índice original: orden de entrada = orden de salida.
func (e *Embedder) GetMany(texts []string) [][]float64 {
	out := make([][]float64, len(texts))
	if len(texts) == 0 {
		return out
	}

	limit := defaultEmbedParallelism
	if e.cfg.EmbedParallelism > 0 {
		limit = e.cfg.EmbedParallelism
	}
	if limit > len(texts) {
		limit = len(texts)
	}

	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup

	for i, t := range texts {
		wg.Add(1)
		go func(idx int, text string) {
			defer wg.Done()
			sem <- struct{}{}        // adquiere slot
			defer func() { <-sem }() // libera slot
			vec, _, err := e.Get(text)
			if err == nil {
				out[idx] = vec
			}
		}(i, t)
	}

	wg.Wait()
	return out
}
