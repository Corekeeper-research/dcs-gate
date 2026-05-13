#!/bin/bash
set -e
echo "[1/5] Verificando .env"
if [ ! -f .env ]; then
  cp .env.example .env
  echo "  -> .env creado. EDITA NGROK_AUTHTOKEN antes de continuar."
  exit 1
fi
echo "[2/5] Verificando datos"
for f in data/core_gpt.jsonl data/shadow_gemini.jsonl data/edge_mixed.jsonl data/poles_1024.json; do
  if [ ! -f "$f" ]; then
    echo "  -> Falta $f"
    exit 1
  fi
done
echo "  -> Todos los archivos de datos presentes."
echo "[3/5] Levantando contenedores"
docker compose up -d --build
echo "[4/5] Verificando modelos Ollama"
pull_model() {
  local model="$1"
  if docker exec ollama ollama list 2>/dev/null | grep -q "^${model}"; then
    echo "  -> $model ya disponible, saltando descarga."
  else
    echo "  -> Descargando $model ..."
    docker exec -it ollama ollama pull "$model"
  fi
}
pull_model "mxbai-embed-large"
pull_model "wizardlm2:7b"
echo "[5/5] Listo. Probando salud:"
sleep 3
curl -s http://localhost:8080/health || echo "  (esperando)"
echo
echo "Frontend: http://localhost:8080/"
echo "API:      curl -X POST http://localhost:8080/auth -H 'Content-Type: application/json' -d '{...}'"
echo "Métricas: curl http://localhost:8080/metrics"
