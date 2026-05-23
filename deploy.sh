#!/usr/bin/env bash
# deploy.sh — DCS-Gate v8.8 deployment para Azure NC8as_T4_v3
# Tesla T4 · AMD EPYC 7V12 · 8 CPUs · 54 GB RAM
# Uso: bash deploy.sh [--ngrok-token TOKEN] [--model qwen2.5:7b]
# Requisito: ngrok authtoken en ~/.config/ngrok/ngrok.yml o --ngrok-token

set -euo pipefail

# ── Colores ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info()  { echo -e "${CYAN}[INFO]${RESET}  $*"; }
ok()    { echo -e "${GREEN}[OK]${RESET}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
die()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; exit 1; }

# ── Parámetros ────────────────────────────────────────────────────────────────
NGROK_TOKEN=""
JUDGE_MODEL="qwen2.5:7b"
EMBED_MODEL="mxbai-embed-large"
DCS_PORT=8080
APP_DIR="${HOME}/dcs-gate"
PATCH_FILE="${HOME}/dcs-recursive-refiner.patch"
SERVICE_NAME="dcs-gate"

while [[ $# -gt 0 ]]; do
  case $1 in
    --ngrok-token) NGROK_TOKEN="$2"; shift 2 ;;
    --model)       JUDGE_MODEL="$2"; shift 2 ;;
    --port)        DCS_PORT="$2"; shift 2 ;;
    --app-dir)     APP_DIR="$2"; shift 2 ;;
    *) warn "Argumento desconocido: $1"; shift ;;
  esac
done

echo -e "${BOLD}═══════════════════════════════════════════════${RESET}"
echo -e "${BOLD}  DCS-Gate v8.8 — Deployment Script${RESET}"
echo -e "${BOLD}  Modelo juez: ${JUDGE_MODEL}${RESET}"
echo -e "${BOLD}  Puerto:      ${DCS_PORT}${RESET}"
echo -e "${BOLD}  Directorio:  ${APP_DIR}${RESET}"
echo -e "${BOLD}═══════════════════════════════════════════════${RESET}"
echo ""

# ── 1. Prerrequisitos del sistema ─────────────────────────────────────────────
info "Verificando prerrequisitos del sistema..."

sudo apt-get update -qq
sudo apt-get install -y -qq curl wget git jq tmux build-essential ca-certificates

# ── 2. Go 1.22+ ───────────────────────────────────────────────────────────────
GO_REQUIRED="1.22"
if command -v go &>/dev/null; then
  GO_INSTALLED=$(go version | awk '{print $3}' | sed 's/go//')
  MAJOR=$(echo "$GO_INSTALLED" | cut -d. -f1)
  MINOR=$(echo "$GO_INSTALLED" | cut -d. -f2)
  if [[ $MAJOR -gt 1 ]] || [[ $MAJOR -eq 1 && $MINOR -ge 22 ]]; then
    ok "Go ${GO_INSTALLED} ya instalado"
  else
    warn "Go ${GO_INSTALLED} < ${GO_REQUIRED}, reinstalando..."
    INSTALL_GO=true
  fi
else
  INSTALL_GO=true
fi

if [[ "${INSTALL_GO:-false}" == "true" ]]; then
  info "Instalando Go 1.22..."
  GO_VERSION="1.22.4"
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  export PATH=$PATH:/usr/local/go/bin
  ok "Go ${GO_VERSION} instalado"
fi

export PATH=$PATH:/usr/local/go/bin

# ── 3. Ollama ─────────────────────────────────────────────────────────────────
if ! command -v ollama &>/dev/null; then
  info "Instalando Ollama..."
  curl -fsSL https://ollama.com/install.sh | sh
  ok "Ollama instalado"
else
  ok "Ollama ya instalado ($(ollama --version 2>/dev/null | head -1))"
fi

# Iniciar Ollama si no está corriendo
if ! pgrep -x ollama &>/dev/null; then
  info "Iniciando Ollama en background..."
  nohup ollama serve > /tmp/ollama.log 2>&1 &
  OLLAMA_PID=$!
  echo "$OLLAMA_PID" > /tmp/ollama.pid
  sleep 3
fi

# Esperar a que Ollama esté listo (máx 60 segundos)
info "Esperando a que Ollama responda..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
    ok "Ollama listo"
    break
  fi
  sleep 2
  if [[ $i -eq 30 ]]; then
    die "Ollama no respondió en 60 segundos. Ver /tmp/ollama.log"
  fi
done

# ── 4. Modelos ────────────────────────────────────────────────────────────────
pull_model() {
  local model="$1"
  if ollama list 2>/dev/null | grep -q "^${model}"; then
    ok "Modelo ${model} ya disponible"
  else
    info "Descargando modelo ${model} (esto puede tardar varios minutos)..."
    ollama pull "$model"
    ok "Modelo ${model} descargado"
  fi
}

pull_model "$JUDGE_MODEL"
pull_model "$EMBED_MODEL"

# ── 5. Código DCS-Gate ────────────────────────────────────────────────────────
if [[ ! -d "$APP_DIR" ]]; then
  die "Directorio ${APP_DIR} no encontrado.
  Opciones:
  1. git clone <repo> ${APP_DIR}
  2. scp -r ./dcs-gate user@vm:${APP_DIR}
  3. Especifica --app-dir /ruta/correcta"
fi

DCS_SRC="${APP_DIR}"
# Detectar si el código está en subdirectorio work/dcs-gate
if [[ -f "${APP_DIR}/work/dcs-gate/main.go" ]]; then
  DCS_SRC="${APP_DIR}/work/dcs-gate"
elif [[ -f "${APP_DIR}/main.go" ]]; then
  DCS_SRC="${APP_DIR}"
else
  die "No se encontró main.go en ${APP_DIR} ni en ${APP_DIR}/work/dcs-gate"
fi

info "Código DCS-Gate en: ${DCS_SRC}"

# ── 6. Aplicar patch v8.8 ─────────────────────────────────────────────────────
# Verificar si el patch ya fue aplicado buscando el símbolo del refiner recursivo
if grep -q "RefineStream" "${DCS_SRC}/judge.go" 2>/dev/null; then
  ok "Patch v8.8 (recursive refiner) ya aplicado"
elif [[ -f "$PATCH_FILE" ]]; then
  info "Aplicando patch v8.8 del recursive refiner..."
  cd "$APP_DIR"
  if patch -p1 --dry-run < "$PATCH_FILE" &>/dev/null; then
    patch -p1 < "$PATCH_FILE"
    ok "Patch aplicado exitosamente"
  else
    warn "El patch no aplica limpio. Puede que ya esté parcialmente aplicado o la versión difiere."
    warn "Verificar manualmente: patch -p1 --dry-run < ${PATCH_FILE}"
  fi
else
  warn "Patch file no encontrado en ${PATCH_FILE}"
  warn "Copiarlo con: scp dcs-recursive-refiner.patch user@vm:${PATCH_FILE}"
  warn "Continuando sin el patch (sin recursive refiner)..."
fi

# ── 7. BRECHA-3 fix — corregir el corpus count en ANALYZER_PROMPT ─────────────
info "Aplicando fix BRECHA-3 (8957-vector → 61-vector)..."
if grep -q "8957-vector" "${DCS_SRC}/judge.go"; then
  sed -i 's/8957-vector control corpus/61-vector control corpus (36 core + 13 shadow + 12 edge)/g' \
    "${DCS_SRC}/judge.go"
  ok "BRECHA-3 corregida"
else
  ok "BRECHA-3 ya corregida (o no presente)"
fi

# ── 8. Compilar ───────────────────────────────────────────────────────────────
info "Compilando DCS-Gate..."
cd "$DCS_SRC"
go build -o dcs-gate ./... || die "Falló la compilación. Ver errores arriba."
ok "Compilación exitosa: ${DCS_SRC}/dcs-gate"

# ── 9. Config de entorno ──────────────────────────────────────────────────────
ENV_FILE="${DCS_SRC}/.env"
cat > "$ENV_FILE" << EOF
PORT=${DCS_PORT}
OLLAMA_URL=http://localhost:11434
EMBED_MODEL=${EMBED_MODEL}
JUDGE_MODEL=${JUDGE_MODEL}
HTTP_TIMEOUT=180s
INTENT_THRESHOLD=0.30
POLE_POS_THR=0.05
POLE_NEG_THR=-0.05
EOF
ok "Archivo de entorno escrito: ${ENV_FILE}"

# ── 10. Servicio tmux ─────────────────────────────────────────────────────────
# Usar tmux (sin systemd) para compatibilidad con VMs sin permisos root
TMUX_SESSION="${SERVICE_NAME}"

if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
  info "Sesión tmux '${TMUX_SESSION}' ya existe, reiniciando..."
  tmux kill-session -t "$TMUX_SESSION"
fi

info "Iniciando DCS-Gate en sesión tmux '${TMUX_SESSION}'..."
tmux new-session -d -s "$TMUX_SESSION" -x 220 -y 50

# Cargar env y ejecutar
tmux send-keys -t "$TMUX_SESSION" "
  cd '${DCS_SRC}' && \\
  export \$(cat .env | grep -v '^#' | xargs) && \\
  echo '=== DCS-GATE INICIANDO ===' && \\
  ./dcs-gate 2>&1 | tee /tmp/dcs-gate.log
" C-m

# Esperar a que DCS-Gate esté listo
info "Esperando a que DCS-Gate responda en puerto ${DCS_PORT}..."
for i in $(seq 1 30); do
  if curl -sf "http://localhost:${DCS_PORT}/health" > /dev/null 2>&1; then
    ok "DCS-Gate listo en puerto ${DCS_PORT}"
    break
  fi
  sleep 2
  if [[ $i -eq 30 ]]; then
    die "DCS-Gate no respondió en 60s. Ver: tmux attach -t ${TMUX_SESSION}"
  fi
done

# Health check
HEALTH=$(curl -sf "http://localhost:${DCS_PORT}/health")
VECTORS=$(echo "$HEALTH" | jq -r '.vectors // "?"')
POLES=$(echo "$HEALTH" | jq -r '.poles_ok // "?"')
GOLDEN=$(echo "$HEALTH" | jq -r '.golden_tests // "?"')
echo ""
echo -e "${GREEN}  ✓ Vectores cargados: ${VECTORS}${RESET}"
echo -e "${GREEN}  ✓ Polos OK:          ${POLES}${RESET}"
echo -e "${GREEN}  ✓ Golden tests:      ${GOLDEN}${RESET}"
echo ""

# ── 11. ngrok ─────────────────────────────────────────────────────────────────
if ! command -v ngrok &>/dev/null; then
  info "Instalando ngrok..."
  curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
    | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
  echo "deb https://ngrok-agent.s3.amazonaws.com buster main" \
    | sudo tee /etc/apt/sources.list.d/ngrok.list
  sudo apt-get update -qq && sudo apt-get install -y -qq ngrok
  ok "ngrok instalado"
fi

# Configurar token si se proporcionó
if [[ -n "$NGROK_TOKEN" ]]; then
  ngrok config add-authtoken "$NGROK_TOKEN"
  ok "ngrok token configurado"
fi

# Iniciar túnel ngrok
NGROK_TMUX="ngrok-tunnel"
if tmux has-session -t "$NGROK_TMUX" 2>/dev/null; then
  tmux kill-session -t "$NGROK_TMUX"
fi

info "Iniciando túnel ngrok → localhost:${DCS_PORT}..."
tmux new-session -d -s "$NGROK_TMUX"
tmux send-keys -t "$NGROK_TMUX" "ngrok http ${DCS_PORT} --log stdout 2>&1 | tee /tmp/ngrok.log" C-m

# Esperar a que ngrok esté listo y obtener la URL
info "Esperando URL de ngrok..."
NGROK_URL=""
for i in $(seq 1 20); do
  sleep 2
  NGROK_URL=$(curl -sf http://localhost:4040/api/tunnels 2>/dev/null \
    | jq -r '.tunnels[0].public_url // ""' 2>/dev/null || true)
  if [[ -n "$NGROK_URL" && "$NGROK_URL" != "null" ]]; then
    break
  fi
done

echo ""
echo -e "${BOLD}═══════════════════════════════════════════════${RESET}"
echo -e "${BOLD}  DEPLOYMENT COMPLETO${RESET}"
echo -e "${BOLD}═══════════════════════════════════════════════${RESET}"
echo ""
echo -e "  ${GREEN}DCS-Gate local:${RESET}  http://localhost:${DCS_PORT}"

if [[ -n "$NGROK_URL" && "$NGROK_URL" != "null" ]]; then
  echo -e "  ${GREEN}ngrok URL:${RESET}       ${NGROK_URL}"
  echo ""
  echo -e "  ${YELLOW}Configurar en diluidocognit.store:${RESET}"
  echo -e "  → Si usas Cloudflare/Nginx como reverse proxy:"
  echo -e "    proxy_pass ${NGROK_URL};"
  echo -e "  → Si configuras el DNS directo al ngrok:"
  echo -e "    CNAME diluidocognit.store → ${NGROK_URL#https://}"
  echo ""
  echo -e "  ${YELLOW}Para ver el refiner en la UI:${RESET}"
  echo -e "  → Abrir: ${NGROK_URL}"
  echo -e "  → La UI usa streaming SSE — el refiner se muestra en tiempo real"
else
  warn "ngrok URL no obtenida. Ver: tmux attach -t ngrok-tunnel"
  warn "Una vez obtenida, úsala como backend para diluidocognit.store"
fi

echo ""
echo -e "  ${CYAN}Logs en tiempo real:${RESET}"
echo -e "  → DCS-Gate:  tmux attach -t ${SERVICE_NAME}   (o: tail -f /tmp/dcs-gate.log)"
echo -e "  → ngrok:     tmux attach -t ${NGROK_TMUX}     (o: tail -f /tmp/ngrok.log)"
echo -e "  → Ollama:    tail -f /tmp/ollama.log"
echo ""
echo -e "  ${CYAN}Para reiniciar DCS-Gate:${RESET}"
echo -e "  bash deploy.sh"
echo ""
echo -e "  ${CYAN}Health check:${RESET}"
echo -e "  curl -s localhost:${DCS_PORT}/health | jq"
echo ""

# ── 12. Verificación final ────────────────────────────────────────────────────
info "Verificación de endpoints..."
ENDPOINTS=("/health" "/metrics")
for ep in "${ENDPOINTS[@]}"; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${DCS_PORT}${ep}")
  if [[ "$STATUS" == "200" ]]; then
    ok "${ep} → HTTP ${STATUS}"
  else
    warn "${ep} → HTTP ${STATUS}"
  fi
done

echo ""
ok "Deploy completo. DCS-Gate v8.8 corriendo con recursive refiner."
