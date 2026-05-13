#!/usr/bin/env python3
"""
embed_corpus.py — DCS-Gate v8

Embide un corpus JSON o JSONL con mxbai-embed-large via Ollama.
Preserva todos los metadatos del corpus en el JSONL de salida.

Modos:
  Normal:       --input corpus.json   --output baseline.jsonl  (embide desde JSON)
  Enrich-only:  --input baseline.jsonl --output baseline.jsonl --poles poles_1024.json --enrich-only
                (añade sim_pos/sim_neg/sim_neu a un JSONL ya embedido, sin re-embedir)

Uso:
  python3 embed_corpus.py --input data/corpus_core.json --output data/baseline_core.jsonl --corpus core
  python3 embed_corpus.py --input data/baseline_core.jsonl --output data/baseline_core.jsonl \
          --corpus core --poles data/poles_1024.json --enrich-only
"""

import argparse
import json
import math
import os
import sys
import time

import requests

# ── Config por defecto ────────────────────────────────────────────────────────

DEFAULT_MODEL  = "mxbai-embed-large"
DEFAULT_OLLAMA = "http://localhost:11434"


def parse_args():
    p = argparse.ArgumentParser(description="Embide un corpus JSON → JSONL con Ollama")
    p.add_argument("--input",       required=True,  help="Ruta al corpus de entrada (.json o .jsonl)")
    p.add_argument("--output",      required=True,  help="Ruta de salida JSONL")
    p.add_argument("--corpus",      default="",     help="Etiqueta del corpus: core | shadow | edge")
    p.add_argument("--model",       default=DEFAULT_MODEL,  help="Modelo Ollama a usar")
    p.add_argument("--ollama",      default=DEFAULT_OLLAMA, help="URL base de Ollama")
    p.add_argument("--poles",       default="",     help="Ruta al poles_1024.json para enriquecer sim_poles")
    p.add_argument("--enrich-only", action="store_true",
                   help="Solo añadir sim_poles a JSONL existente, no re-embedir")
    p.add_argument("--batch",       type=int, default=1, help="Tamaño de batch (1 = sin batch)")
    p.add_argument("--retry",       type=int, default=3, help="Reintentos por embedding fallido")
    return p.parse_args()


# ── Carga del corpus ──────────────────────────────────────────────────────────

def load_corpus(path: str) -> list[dict]:
    """Carga corpus desde JSON (list of dicts) o JSONL (una línea por dict)."""
    ext = os.path.splitext(path)[1].lower()
    with open(path, encoding="utf-8") as f:
        if ext == ".jsonl":
            docs = []
            for line in f:
                line = line.strip()
                if line:
                    docs.append(json.loads(line))
            return docs
        else:
            data = json.load(f)
            # Acepta tanto {"blocks": [...]} como [...]
            if isinstance(data, list):
                return data
            if "blocks" in data:
                return data["blocks"]
            # Cualquier otro wrapper de primer nivel con lista
            for v in data.values():
                if isinstance(v, list):
                    return v
            return [data]


def extract_text(doc: dict) -> str:
    """Extrae el texto principal de un documento del corpus."""
    for field in ("text", "content", "response", "phrase", "fragment"):
        if field in doc and isinstance(doc[field], str) and doc[field].strip():
            return doc[field].strip()
    return ""


# ── Embedding ─────────────────────────────────────────────────────────────────

def embed_text(text: str, model: str, ollama_url: str, retries: int = 3) -> list[float] | None:
    url = ollama_url.rstrip("/") + "/api/embeddings"
    for attempt in range(retries):
        try:
            r = requests.post(url, json={"model": model, "prompt": text}, timeout=60)
            r.raise_for_status()
            embedding = r.json().get("embedding")
            if embedding and len(embedding) > 0:
                return embedding
        except requests.RequestException as e:
            if attempt == retries - 1:
                print(f"  ERROR embedding tras {retries} intentos: {e}", file=sys.stderr)
            else:
                time.sleep(1.0 * (attempt + 1))
    return None


def normalize(vec: list[float]) -> list[float]:
    norm = math.sqrt(sum(x * x for x in vec))
    if norm < 1e-12:
        return vec
    return [x / norm for x in vec]


def dot(a: list[float], b: list[float]) -> float:
    return sum(x * y for x, y in zip(a, b))


# ── Polos ─────────────────────────────────────────────────────────────────────

def load_poles(path: str) -> dict[str, list[float]] | None:
    """Carga poles_1024.json y extrae los centroides pos/neg/neu normalizados."""
    if not path or not os.path.isfile(path):
        return None
    with open(path, encoding="utf-8") as f:
        raw = json.load(f)
    poles = {}
    for key in ("pos", "neg", "neu"):
        v = raw.get(key)
        if v is None:
            continue
        if isinstance(v, list):
            if len(v) == 0:
                continue
            if isinstance(v[0], list):
                # Lista de vectores → centroide
                n = len(v[0])
                centroid = [0.0] * n
                for vec in v:
                    for i, x in enumerate(vec):
                        centroid[i] += x
                centroid = [x / len(v) for x in centroid]
                poles[key] = normalize(centroid)
            else:
                poles[key] = normalize(v)
    return poles if poles else None


def enrich_with_poles(entry_vec: list[float], poles: dict[str, list[float]]) -> dict[str, float]:
    """Calcula sim_pos, sim_neg, sim_neu para un vector ya normalizado."""
    result = {}
    nv = normalize(entry_vec)
    for key in ("pos", "neg", "neu"):
        if key in poles:
            result[f"sim_{key}"] = round(dot(nv, poles[key]), 6)
    return result


# ── Modos principales ─────────────────────────────────────────────────────────

def run_enrich_only(args):
    """Lee el JSONL ya embedido y añade sim_pos/sim_neg/sim_neu sin re-embedir."""
    poles = load_poles(args.poles)
    if not poles:
        print("ERROR: se requiere --poles con un archivo válido para --enrich-only", file=sys.stderr)
        sys.exit(1)
    docs = load_corpus(args.input)
    print(f"[enrich-only] {len(docs)} entradas · polos disponibles: {list(poles.keys())}")
    enriched = []
    skipped = 0
    for doc in docs:
        vec = doc.get("vector")
        if not vec or not isinstance(vec, list):
            skipped += 1
            enriched.append(doc)
            continue
        sims = enrich_with_poles(vec, poles)
        doc.update(sims)
        enriched.append(doc)
    tmp = args.output + ".tmp"
    with open(tmp, "w", encoding="utf-8") as out:
        for doc in enriched:
            out.write(json.dumps(doc, ensure_ascii=False) + "\n")
    os.replace(tmp, args.output)
    print(f"[enrich-only] escritas {len(enriched)} entradas → {args.output} (skipped sin vector: {skipped})")


def run_embed(args):
    """Embide desde JSON o JSONL, preservando todos los metadatos."""
    poles = load_poles(args.poles) if args.poles else None
    docs = load_corpus(args.input)
    print(f"[embed] {len(docs)} documentos cargados desde {args.input}")
    print(f"[embed] modelo={args.model} · ollama={args.ollama} · corpus={args.corpus or '(sin etiqueta)'}")
    if poles:
        print(f"[embed] enriqueciendo con poles: {list(poles.keys())}")

    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    written = 0
    failed = 0

    with open(args.output, "w", encoding="utf-8") as out:
        for i, doc in enumerate(docs, 1):
            text = extract_text(doc)
            if not text:
                print(f"  [{i}/{len(docs)}] SKIP — sin texto extraíble", file=sys.stderr)
                failed += 1
                continue
            vec = embed_text(text, args.model, args.ollama, args.retry)
            if vec is None:
                print(f"  [{i}/{len(docs)}] FAIL — embedding retornó None", file=sys.stderr)
                failed += 1
                continue
            # Construir la entrada de salida preservando metadatos del corpus
            entry = {
                "vector":  normalize(vec),
                "text":    text,
                "corpus":  doc.get("corpus", args.corpus) or args.corpus,
            }
            # Propagar todos los campos de metadatos del corpus
            for field in (
                "block_id", "position", "primary_pattern", "secondary_patterns",
                "source_model", "notes", "tags", "metrics",
                "section", "label", "intent", "response_fragment",
            ):
                if field in doc:
                    entry[field] = doc[field]
            # Enriquecer con sim_poles si hay polos disponibles
            if poles:
                sims = enrich_with_poles(vec, poles)
                entry.update(sims)
            out.write(json.dumps(entry, ensure_ascii=False) + "\n")
            written += 1
            if i % 10 == 0 or i == len(docs):
                print(f"  [{i}/{len(docs)}] {written} escritos, {failed} fallidos")

    print(f"[embed] completado: {written} vectores escritos, {failed} fallidos → {args.output}")
    if failed > 0:
        sys.exit(1)


def main():
    args = parse_args()
    if args.enrich_only:
        run_enrich_only(args)
    else:
        run_embed(args)


if __name__ == "__main__":
    main()
