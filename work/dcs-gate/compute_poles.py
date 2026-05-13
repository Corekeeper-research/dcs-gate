#!/usr/bin/env python3
"""
compute_poles.py — DCS-Gate v8

Calcula tres polos semánticos a partir de los baseline JSONL.

Modo "pattern" (default, recomendado por el README v8):
  pos  → centroide de bloques con primary_pattern de control / certeza performada
         (VALIDATE, EXPAND, CLOSE, EVADE, FRAME_CAPTURE, MIRROR, ALIGN,
          FABRICATE, REGISTER_MATCH, REDIRECT_EMOTIONAL, REDIRECT_SEMANTIC,
          CONTROL_SELF_EXPOSURE, ANCHOR, SOFT_DEFLECT, PATTERN_LOCK …)
  neg  → centroide de bloques con primary_pattern de coherencia honesta / no-cierre
         (HOLD_OPEN, PROBE, CALIBRATE, EXPLORE, REPAIR)
  neu  → centroide del pool edge (frontera / ambiguo)

Esta semántica casa con las etiquetas de baseline.go:
  bucket +1 = "certeza_performada"  → cerca del polo de control (pos)
  bucket -1 = "duda_performada"     → cerca del polo de coherencia honesta (neg)

Modo "pool" (--legacy o --mode pool):
  pos  → centroide del pool core entero
  neg  → centroide del pool shadow entero
  neu  → centroide del pool edge entero
(comportamiento original v8.0; se mantiene por compatibilidad con scripts previos)

Salida: poles_1024.json con los tres centroides como listas de floats.

Uso:
  python3 compute_poles.py
  python3 compute_poles.py --legacy
  python3 compute_poles.py --pos-patterns VALIDATE,CLOSE,FABRICATE
  python3 compute_poles.py --core data/baseline_core.jsonl --shadow data/baseline_shadow.jsonl
  python3 compute_poles.py --output data/poles_custom.json --verbose
"""

import argparse
import json
import math
import os
import sys


# ── Defaults relativos al directorio del script ───────────────────────────────

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))

DEFAULT_CORE_PATH   = os.path.join(SCRIPT_DIR, "data", "baseline_core.jsonl")
DEFAULT_SHADOW_PATH = os.path.join(SCRIPT_DIR, "data", "baseline_shadow.jsonl")
DEFAULT_EDGE_PATH   = os.path.join(SCRIPT_DIR, "data", "baseline_edge.jsonl")
DEFAULT_OUTPUT      = os.path.join(SCRIPT_DIR, "data", "poles_1024.json")

# Conjuntos por defecto de primary_pattern por polaridad.
# pos = patrones de control / certeza performada
# neg = patrones de coherencia honesta / no-cierre genuino
# neu = patrones híbridos / frontera (vacío => usa el pool edge tal cual)

DEFAULT_POS_PATTERNS = [
    "VALIDATE", "EXPAND", "CLOSE",
    "EVADE",
    "REDIRECT_EMOTIONAL", "REDIRECT_SEMANTIC",
    "REGISTER_MATCH",
    "FRAME_CAPTURE", "ALIGN", "MIRROR", "FABRICATE",
    "CONTROL_SELF_EXPOSURE", "ANCHOR", "SOFT_DEFLECT", "PATTERN_LOCK",
    # patrones secundarios mencionados en el corpus shadow
    "PERFORMED_HUMILITY", "STRUCTURAL_AUTHORITY", "DUAL_ANGLE_DISGUISE",
    "EMOTIONAL_ANCHORING", "VOCABULARY_CALIBRATION",
    "PROJECTED_VALIDATION", "ANTICIPATORY_EXPANSION", "COMPLACENCY_INDUCTION",
    "INDIVIDUALITY_VERBS", "REDIRECT_AS_CARE",
]

DEFAULT_NEG_PATTERNS = [
    "HOLD_OPEN", "PROBE", "CALIBRATE", "EXPLORE", "REPAIR",
]

DEFAULT_NEU_PATTERNS: list[str] = []  # vacío => usa el pool edge entero


def parse_args():
    p = argparse.ArgumentParser(description="Calcula los tres polos semánticos del corpus DCS")
    p.add_argument("--core",   default=DEFAULT_CORE_PATH,   help="JSONL del pool core")
    p.add_argument("--shadow", default=DEFAULT_SHADOW_PATH, help="JSONL del pool shadow")
    p.add_argument("--edge",   default=DEFAULT_EDGE_PATH,   help="JSONL del pool edge")
    p.add_argument("--output", default=DEFAULT_OUTPUT,      help="Ruta de salida del JSON de polos")
    p.add_argument("--mode", choices=["pattern", "pool"], default="pattern",
                   help="pattern (default): filtra por primary_pattern. "
                        "pool: comportamiento legacy (core→pos, shadow→neg, edge→neu)")
    p.add_argument("--legacy", action="store_true",
                   help="Atajo equivalente a --mode pool (compatibilidad con scripts previos)")
    p.add_argument("--pos-patterns", default=",".join(DEFAULT_POS_PATTERNS),
                   help="Lista (csv) de primary_pattern asignados al polo pos")
    p.add_argument("--neg-patterns", default=",".join(DEFAULT_NEG_PATTERNS),
                   help="Lista (csv) de primary_pattern asignados al polo neg")
    p.add_argument("--neu-patterns", default=",".join(DEFAULT_NEU_PATTERNS),
                   help="Lista (csv) de primary_pattern para el polo neu (vacío = pool edge entero)")
    p.add_argument("--verbose", action="store_true", help="Imprimir info por pool y por patrón")
    return p.parse_args()


# ── Utilidades ────────────────────────────────────────────────────────────────

def load_entries(path: str) -> list[dict]:
    """Carga todas las entradas de un JSONL preservando metadatos.

    Cada entrada es un dict con al menos {vector, primary_pattern, …}.
    Las líneas malformadas se ignoran con WARN.
    """
    if not os.path.isfile(path):
        print(f"  WARN: {path} no existe — pool vacío", file=sys.stderr)
        return []
    entries: list[dict] = []
    with open(path, encoding="utf-8") as f:
        for line_no, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                doc = json.loads(line)
            except json.JSONDecodeError as e:
                print(f"  WARN línea {line_no} malformada: {e}", file=sys.stderr)
                continue
            vec = doc.get("vector")
            if not vec or not isinstance(vec, list) or len(vec) == 0:
                continue
            entries.append(doc)
    return entries


def filter_by_pattern(entries: list[dict], allowed: set) -> list[dict]:
    """Devuelve solo las entradas cuyo primary_pattern está en el conjunto.

    Si allowed está vacío devuelve la lista entera (sin filtro).
    Las entradas sin primary_pattern se descartan en modo filtro.
    """
    if not allowed:
        return list(entries)
    return [e for e in entries if e.get("primary_pattern") in allowed]


def vectors(entries: list[dict]) -> list[list[float]]:
    return [e["vector"] for e in entries]


def pattern_breakdown(entries: list[dict]) -> dict:
    """Cuenta entradas por primary_pattern (para verbose)."""
    counts: dict = {}
    for e in entries:
        k = e.get("primary_pattern") or "(sin patrón)"
        counts[k] = counts.get(k, 0) + 1
    return counts


def parse_csv(s: str) -> set:
    """Parsea un string csv en un conjunto de patrones, ignorando vacíos."""
    return {p.strip() for p in s.split(",") if p.strip()}


def centroid(vecs: list[list[float]]) -> list[float] | None:
    """Calcula el centroide (media aritmética) de una lista de vectores."""
    if not vecs:
        return None
    dim = len(vecs[0])
    c = [0.0] * dim
    for v in vecs:
        for i, x in enumerate(v[:dim]):
            c[i] += x
    n = len(vecs)
    return [x / n for x in c]


def normalize(vec: list[float]) -> list[float]:
    norm = math.sqrt(sum(x * x for x in vec))
    if norm < 1e-12:
        return vec
    return [x / norm for x in vec]


def norm_of(vec: list[float]) -> float:
    return math.sqrt(sum(x * x for x in vec))


def dot(a: list[float], b: list[float]) -> float:
    return sum(x * y for x, y in zip(a, b))


def stats(vecs: list[list[float]], pole: list[float]) -> dict:
    """Calcula estadísticas de similitud de los vectores contra el centroide del polo."""
    if not vecs or pole is None:
        return {}
    sims = [dot(normalize(v), pole) for v in vecs]
    sims.sort()
    n = len(sims)
    return {
        "n":      n,
        "min":    round(sims[0], 4),
        "max":    round(sims[-1], 4),
        "mean":   round(sum(sims) / n, 4),
        "median": round(sims[n // 2], 4),
    }


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    args = parse_args()

    if args.legacy:
        args.mode = "pool"

    print("=== compute_poles.py — DCS-Gate v8 ===")
    print(f"  modo:   {args.mode}")
    print(f"  core:   {args.core}")
    print(f"  shadow: {args.shadow}")
    print(f"  edge:   {args.edge}")
    print(f"  output: {args.output}")
    print()

    # Cargar entradas por pool (preservan primary_pattern)
    core_entries   = load_entries(args.core)
    shadow_entries = load_entries(args.shadow)
    edge_entries   = load_entries(args.edge)

    print(f"  core:   {len(core_entries)} bloques")
    print(f"  shadow: {len(shadow_entries)} bloques")
    print(f"  edge:   {len(edge_entries)} bloques")
    print()

    if not core_entries and not shadow_entries and not edge_entries:
        print("ERROR: todos los pools están vacíos — no hay nada que calcular", file=sys.stderr)
        sys.exit(1)

    pos_patterns = parse_csv(args.pos_patterns)
    neg_patterns = parse_csv(args.neg_patterns)
    neu_patterns = parse_csv(args.neu_patterns)

    if args.mode == "pool":
        # Comportamiento legacy: cada pool va directo a su polo, sin filtrar
        pos_entries = core_entries
        neg_entries = shadow_entries
        neu_entries = edge_entries
        print("  [legacy] sin filtro por primary_pattern")
    else:
        # Modo pattern: barre los TRES pools y filtra por primary_pattern
        all_entries = core_entries + shadow_entries + edge_entries
        pos_entries = filter_by_pattern(all_entries, pos_patterns)
        neg_entries = filter_by_pattern(all_entries, neg_patterns)
        # Para neu: si el usuario pasó patrones, filtra; si no, usa el pool edge entero
        if neu_patterns:
            neu_entries = filter_by_pattern(all_entries, neu_patterns)
        else:
            neu_entries = edge_entries

        print(f"  [pattern] pos: {len(pos_entries)} bloques (patrones: {sorted(pos_patterns)})")
        print(f"  [pattern] neg: {len(neg_entries)} bloques (patrones: {sorted(neg_patterns)})")
        if neu_patterns:
            print(f"  [pattern] neu: {len(neu_entries)} bloques (patrones: {sorted(neu_patterns)})")
        else:
            print(f"  [pattern] neu: {len(neu_entries)} bloques (pool edge entero)")

        if args.verbose:
            print("\n  desglose por primary_pattern (pos):")
            for k, v in sorted(pattern_breakdown(pos_entries).items()):
                print(f"    {k}: {v}")
            print("\n  desglose por primary_pattern (neg):")
            for k, v in sorted(pattern_breakdown(neg_entries).items()):
                print(f"    {k}: {v}")

    pos_vecs = vectors(pos_entries)
    neg_vecs = vectors(neg_entries)
    neu_vecs = vectors(neu_entries)

    # Calcular centroides y normalizar
    pos_raw = centroid(pos_vecs)
    neg_raw = centroid(neg_vecs)
    neu_raw = centroid(neu_vecs)

    pos = normalize(pos_raw) if pos_raw else None
    neg = normalize(neg_raw) if neg_raw else None
    neu = normalize(neu_raw) if neu_raw else None

    # Verificar dimensiones
    dims = {len(v) for v in [pos, neg, neu] if v is not None}
    if len(dims) > 1:
        print(f"WARN: dimensiones inconsistentes entre polos: {dims}", file=sys.stderr)
    if dims:
        print(f"\n  dim: {next(iter(dims))}")

    # Estadísticas (verbose)
    if args.verbose:
        if pos:
            print(f"\n  [pos] stats: {stats(pos_vecs, pos)}")
        if neg:
            print(f"  [neg] stats: {stats(neg_vecs, neg)}")
        if neu:
            print(f"  [neu] stats: {stats(neu_vecs, neu)}")

    # Calcular separación entre polos (calidad del espacio)
    sep = {}
    if pos and neg:
        sep["pos_neg"] = round(dot(pos, neg), 4)
    if pos and neu:
        sep["pos_neu"] = round(dot(pos, neu), 4)
    if neg and neu:
        sep["neg_neu"] = round(dot(neg, neu), 4)
    if sep:
        print(f"\n  Separación entre polos (dot product, menor es mejor):")
        for k, v in sep.items():
            print(f"    {k}: {v}")

    # Construir output
    output = {
        "pos": pos,
        "neg": neg,
        "neu": neu,
        "meta": {
            "mode":     args.mode,
            "core_n":   len(core_entries),
            "shadow_n": len(shadow_entries),
            "edge_n":   len(edge_entries),
            "pos_n":    len(pos_vecs),
            "neg_n":    len(neg_vecs),
            "neu_n":    len(neu_vecs),
            "pos_patterns": sorted(pos_patterns) if args.mode == "pattern" else [],
            "neg_patterns": sorted(neg_patterns) if args.mode == "pattern" else [],
            "neu_patterns": sorted(neu_patterns) if args.mode == "pattern" else [],
            "separation": sep,
            "dim":      next(iter(dims)) if dims else 0,
        },
    }

    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(output, f, ensure_ascii=False)  # sin indent para ahorrar espacio

    print(f"\n  polos escritos → {args.output}")

    # Verificación de sanidad
    if pos and neg:
        sim = dot(pos, neg)
        if sim > 0.95:
            print(f"\n  WARN: sim(pos, neg) = {sim:.4f} — los polos son muy similares, "
                  f"¿el corpus tiene suficiente variedad?", file=sys.stderr)
        else:
            print(f"  OK: separación pos↔neg = {1 - sim:.4f}")

    print("\n  Listo.")


if __name__ == "__main__":
    main()
