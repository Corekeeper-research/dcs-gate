#!/usr/bin/env bash
# =============================================================================
# embed_all.sh — DCS-Gate v8: setup completo de embedding pipeline
#
# Hace:
#   1. Mata contenedores Docker activos (Ollama y cualquier basura previa)
#   2. Levanta Ollama via Docker (o usa instancia nativa si ya está corriendo)
#   3. Descarga mxbai-embed-large
#   4. Embede los tres corpus JSON → JSONL con metadatos preservados
#   5. Computa los tres polos (pos/neg/neu)
#   6. Enriquece los JSONL con sim_pos / sim_neg / sim_neu
#   7. Compila el binario Go
#
# Uso:
#   cd /ruta/a/dcs-gate-v8
#   chmod +x embed_all.sh
#   ./embed_all.sh
#
# Opcional — si Ollama ya corre nativo (sin Docker), pasar flag:
#   OLLAMA_NATIVE=1 ./embed_all.sh
# =============================================================================

set -euo pipefail

# ── Colores ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GRN='\033[0;32m'
YLW='\033[1;33m'
BLU='\033[0;34m'
RST='\033[0m'

log()  { echo -e "${BLU}[DCS]${RST} $*"; }
ok()   { echo -e "${GRN}[OK ]${RST} $*"; }
warn() { echo -e "${YLW}[WRN]${RST} $*"; }
die()  { echo -e "${RED}[ERR]${RST} $*" >&2; exit 1; }

# ── Configuración ─────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OLLAMA_MODEL="${OLLAMA_MODEL:-mxbai-embed-large}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
OLLAMA_NATIVE="${OLLAMA_NATIVE:-0}"
DOCKER_CONTAINER_NAME="dcs-ollama"
OLLAMA_IMAGE="ollama/ollama"
WAIT_SECS=15

DATA_DIR="$SCRIPT_DIR/data"
CORE_JSON="$DATA_DIR/corpus_core.json"
SHADOW_JSON="$DATA_DIR/corpus_shadow.json"
EDGE_JSON="$DATA_DIR/corpus_edge.json"
CORE_JSONL="$DATA_DIR/baseline_core.jsonl"
SHADOW_JSONL="$DATA_DIR/baseline_shadow.jsonl"
EDGE_JSONL="$DATA_DIR/baseline_edge.jsonl"
POLES_JSON="$DATA_DIR/poles_1024.json"

# ── Preflight ─────────────────────────────────────────────────────────────────
log "=== DCS-Gate v8 — Embedding Pipeline ==="
log "Directorio: $SCRIPT_DIR"
log "Modelo:     $OLLAMA_MODEL"
log "Ollama URL: $OLLAMA_URL"
echo

cd "$SCRIPT_DIR"

# Verificar Python
python3 --version &>/dev/null || die "Python3 no encontrado"
ok "Python3 disponible"

# Verificar scripts
[[ -f embed_corpus.py ]]  || die "embed_corpus.py no encontrado en $SCRIPT_DIR"
[[ -f compute_poles.py ]] || die "compute_poles.py no encontrado en $SCRIPT_DIR"
ok "Scripts Python presentes"

# Verificar corpus JSON
missing=0
for f in "$CORE_JSON" "$SHADOW_JSON" "$EDGE_JSON"; do
    [[ -f "$f" ]] || { warn "Corpus no encontrado: $f"; missing=1; }
done
[[ $missing -eq 1 ]] && die "Faltan archivos de corpus en $DATA_DIR — coloca corpus_core.json, corpus_shadow.json, corpus_edge.json"
ok "Corpus JSON presentes (core + shadow + edge)"

mkdir -p "$DATA_DIR"

# ── Paso 1: Docker cleanup ────────────────────────────────────────────────────
if [[ "$OLLAMA_NATIVE" == "0" ]]; then
    log "Paso 1: limpiando contenedores Docker previos..."

    if command -v docker &>/dev/null; then
        # Matar y remover cualquier contenedor que use el puerto 11434
        OLD_IDS=$(docker ps -q --filter "publish=11434" 2>/dev/null || true)
        if [[ -n "$OLD_IDS" ]]; then
            log "  Deteniendo contenedores en puerto 11434: $OLD_IDS"
            docker stop $OLD_IDS &>/dev/null || true
            docker rm   $OLD_IDS &>/dev/null || true
        fi

        # Matar el contenedor nombrado específico si existe
        if docker ps -a --format '{{.Names}}' | grep -q "^${DOCKER_CONTAINER_NAME}$"; then
            log "  Removiendo contenedor $DOCKER_CONTAINER_NAME..."
            docker stop "$DOCKER_CONTAINER_NAME" &>/dev/null || true
            docker rm   "$DOCKER_CONTAINER_NAME" &>/dev/null || true
        fi
        ok "Limpieza Docker completada"
    else
        warn "Docker no encontrado — asumiendo Ollama nativo"
        OLLAMA_NATIVE=1
    fi
else
    log "Paso 1: saltado (OLLAMA_NATIVE=1)"
fi

# ── Paso 2: Levantar Ollama ───────────────────────────────────────────────────
log "Paso 2: levantando Ollama..."

if [[ "$OLLAMA_NATIVE" == "0" ]]; then
    # Detectar si hay GPU disponible
    GPU_FLAG=""
    if docker info 2>/dev/null | grep -qi "nvidia"; then
        GPU_FLAG="--gpus all"
        log "  GPU Nvidia detectada — usando aceleración GPU"
    else
        warn "  Sin GPU detectada — corriendo en CPU (más lento)"
    fi

    log "  Iniciando contenedor $DOCKER_CONTAINER_NAME..."
    docker run -d \
        --name "$DOCKER_CONTAINER_NAME" \
        $GPU_FLAG \
        -v ollama_data:/root/.ollama \
        -p 11434:11434 \
        "$OLLAMA_IMAGE" &>/dev/null

    log "  Esperando que Ollama arranque (${WAIT_SECS}s)..."
    sleep "$WAIT_SECS"
else
    # Ollama nativo: verificar si ya corre, si no arrancar en background
    if ! curl -sf "$OLLAMA_URL/api/tags" &>/dev/null; then
        log "  Arrancando Ollama nativo en background..."
        ollama serve &>/tmp/ollama_serve.log &
        OLLAMA_PID=$!
        sleep 5
        log "  PID Ollama: $OLLAMA_PID (logs en /tmp/ollama_serve.log)"
    else
        ok "  Ollama nativo ya está corriendo"
    fi
fi

# Esperar hasta que Ollama responda
log "  Verificando que Ollama responda..."
MAX_WAIT=60
ELAPSED=0
until curl -sf "$OLLAMA_URL/api/tags" &>/dev/null; do
    sleep 2
    ELAPSED=$((ELAPSED + 2))
    [[ $ELAPSED -ge $MAX_WAIT ]] && die "Ollama no respondió en ${MAX_WAIT}s — revisa logs"
done
ok "Ollama responde en $OLLAMA_URL"

# ── Paso 3: Pull del modelo ───────────────────────────────────────────────────
log "Paso 3: descargando modelo $OLLAMA_MODEL..."

if [[ "$OLLAMA_NATIVE" == "0" ]]; then
    docker exec "$DOCKER_CONTAINER_NAME" ollama pull "$OLLAMA_MODEL"
else
    ollama pull "$OLLAMA_MODEL"
fi
ok "Modelo $OLLAMA_MODEL disponible"

# ── Paso 4: Embedding de los tres corpus ─────────────────────────────────────
log "Paso 4: embidiendo corpus..."

embed_one() {
    local input="$1" output="$2" corpus="$3"
    log "  → $corpus ($input)"
    python3 embed_corpus.py \
        --input  "$input" \
        --output "$output" \
        --corpus "$corpus" \
        --ollama "$OLLAMA_URL"
    local n
    n=$(wc -l < "$output" 2>/dev/null || echo 0)
    ok "  $corpus: $n vectores escritos en $output"
}

embed_one "$CORE_JSON"   "$CORE_JSONL"   "core"
embed_one "$SHADOW_JSON" "$SHADOW_JSONL" "shadow"
embed_one "$EDGE_JSON"   "$EDGE_JSONL"   "edge"

# ── Paso 5: Compute poles ─────────────────────────────────────────────────────
log "Paso 5: computando polos..."

python3 compute_poles.py \
    --core   "$CORE_JSONL" \
    --shadow "$SHADOW_JSONL" \
    --edge   "$EDGE_JSONL" \
    --output "$POLES_JSON"

ok "Polos calculados → $POLES_JSON"

# ── Paso 6: Enriquecer JSONL con sim_poles ────────────────────────────────────
log "Paso 6: enriqueciendo JSONL con sim_pos / sim_neg / sim_neu..."

enrich_one() {
    local jsonl="$1" corpus="$2"
    log "  → enriqueciendo $corpus"
    python3 embed_corpus.py \
        --input       "$jsonl" \
        --output      "$jsonl" \
        --corpus      "$corpus" \
        --poles       "$POLES_JSON" \
        --enrich-only
    ok "  $corpus: enriquecido"
}

enrich_one "$CORE_JSONL"   "core"
enrich_one "$SHADOW_JSONL" "shadow"
enrich_one "$EDGE_JSONL"   "edge"

# ── Paso 7: Compilar Go ───────────────────────────────────────────────────────
log "Paso 7: compilando binario Go..."

if command -v go &>/dev/null; then
    go build -o dcs-gate ./...
    ok "Binario compilado: $SCRIPT_DIR/dcs-gate"
else
    warn "Go no encontrado — saltando compilación. Instala go 1.22+ para compilar."
fi

# ── Resumen ───────────────────────────────────────────────────────────────────
echo
echo -e "${GRN}============================================================${RST}"
echo -e "${GRN} DCS-Gate v8 — Pipeline completado${RST}"
echo -e "${GRN}============================================================${RST}"
echo
echo "Archivos generados:"
for f in "$CORE_JSONL" "$SHADOW_JSONL" "$EDGE_JSONL" "$POLES_JSON"; do
    if [[ -f "$f" ]]; then
        n=$(wc -l < "$f" 2>/dev/null || python3 -c "import json; print(len(json.load(open('$f'))))" 2>/dev/null || echo "?")
        echo "  ✓ $(basename $f) — $n líneas"
    fi
done

if [[ -f "$SCRIPT_DIR/dcs-gate" ]]; then
    echo "  ✓ dcs-gate (binario)"
fi
echo
echo "Para correr:"
echo "  ./dcs-gate"
echo
echo "Para parar Ollama Docker (cuando termines):"
echo "  docker stop $DOCKER_CONTAINER_NAME"
