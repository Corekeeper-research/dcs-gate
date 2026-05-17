package main

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	OllamaURL        string
	EmbedModel       string
	JudgeModel       string
	IntentThreshold  float64
	MarkerThreshold  float64
	PolePosThr       float64
	PoleNegThr       float64
	CacheSize        int
	HTTPTimeout      time.Duration
	EmbedParallelism int
}

func loadConfig() Config {
	return Config{
		Port:             getenv("PORT", "8081"),
		OllamaURL:        getenv("OLLAMA_URL", "http://localhost:11434"),
		EmbedModel:       getenv("EMBED_MODEL", "mxbai-embed-large"),
		JudgeModel:       getenv("JUDGE_MODEL", "qwen3:14b"),
		IntentThreshold:  getenvF("INTENT_THRESHOLD", 0.55),
		MarkerThreshold:  getenvF("MARKER_THRESHOLD", 0.55),
		PolePosThr:       getenvF("POLE_POS_THRESHOLD", 0.25),
		PoleNegThr:       getenvF("POLE_NEG_THRESHOLD", -0.25),
		CacheSize:        getenvI("CACHE_SIZE", 2000),
		HTTPTimeout:      time.Duration(getenvI("HTTP_TIMEOUT_SECONDS", 300)) * time.Second,
		EmbedParallelism: getenvI("EMBED_PARALLELISM", 2),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvF(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getenvI(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
