# DCS-Gate v8.2 — Setup en Google Colab (30 GB RAM / ≈ 520 GB disco)

> Pensado para correr el pipeline completo en una notebook de Colab gratis o
> Pro: instalar Go + Ollama + el modelo de embedding, embedir los 3 corpus,
> calcular polos, compilar el servicio, y dejarlo respondiendo en `/score`.

Colab te da por defecto:

| Recurso | Free | Pro |
|---|---|---|
| RAM | 12.7 GB | hasta 51 GB |
| Disco | ≈ 110 GB | ≈ 200 GB / 520 GB con runtime de notebook largo |
| GPU | T4 (variable) | A100 / V100 (a veces) |

Para DCS-Gate **no necesitas GPU**. Hay dos modelos en juego, ambos via Ollama:

| Rol | Modelo | Tamaño disco | RAM cargado | Uso |
|---|---|---|---|---|
| Embeddings (1024d) | `mxbai-embed-large` | 670 MB | ~1.0 GB | una llamada por frase / por bloque |
| Juez (default v8.1) | `wizardlm2:7b`         | 4.1 GB | ~5–6 GB | una llamada por `/auth` (~2–3 s en CPU) |
| **Juez (v8.2 recomendado)** | `qwen2.5:7b-instruct` | 4.7 GB | ~6–7 GB | drop-in: mejor adherencia JSON + español |
| Juez (max calidad, opcional) | `qwen2.5:14b-instruct` | 8.9 GB | ~10–11 GB | mejor razonamiento, ~5–7 s por request |

Total con embed + juez 7B cargados: **~7–8 GB RAM, ~5 GB disco**. Con
qwen2.5:14b: **~12 GB RAM, ~10 GB disco**. Con tus 30 GB de RAM y
~520 GB de disco te sobra muchísimo (queda >18 GB libres para experimentos:
más corpus, k-means, ejes adicionales, etc).

### Por qué qwen2.5:7b-instruct sobre wizardlm2:7b en v8.2

`wizardlm2:7b` (default histórico): JSON poco estable, español ocasionalmente
forzado, no respeta bien el `format=json` cuando hay markdown en el prompt.

`qwen2.5:7b-instruct`: misma RAM/disco, **mejor JSON estricto**, español
nativo, sigue prompt structurado mejor — y el system prompt v8.2 ya carga
top1_metrics y reference cases, que dependen de adherencia al formato.
**Cero cambios de código** — solo cambia `JUDGE_MODEL` (env var) y el `pull`.

Si tienes los 30 GB y quieres calidad máxima sin pagar APIs externas:
`qwen2.5:14b-instruct`. Razona mejor sobre ambigüedades (humildad genuina vs
performada). Costo: 2–3× más lento por request (~5–7 s en CPU).

> Si vas a llamar `/auth` con `?judge=false` (calibración rápida) o `/score`
> sin juez, te basta con el embed model y consumes solo ~1 GB.

---

## 1. Estructura del notebook (orden de las celdas)

Pega cada bloque tal cual en una celda nueva, en este orden:

### Celda 1 — Subir el zip

```python
from google.colab import files
import shutil, os

# Limpia restos de runs anteriores
shutil.rmtree("/content/dcs", ignore_errors=True)
os.makedirs("/content/dcs", exist_ok=True)

# Sube `dcs-gate-v8.1.zip` (el que te pasé)
uploaded = files.upload()
zip_name = list(uploaded.keys())[0]
print("Subido:", zip_name)
```

### Celda 2 — Descomprimir

```bash
%%bash
cd /content/dcs
unzip -o /content/$(ls /content/*.zip | head -1) -d /content/dcs
ls -la
```

### Celda 3 — Instalar Go 1.22

```bash
%%bash
set -e
GO_VERSION=1.22.5
cd /tmp
wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
export PATH=/usr/local/go/bin:$PATH
go version
```

```python
# Hacer Go visible al runtime de Python
import os
os.environ["PATH"] = "/usr/local/go/bin:" + os.environ["PATH"]
!go version
```

### Celda 4 — Instalar Ollama (servicio de embeddings)

```bash
%%bash
set -e
curl -fsSL https://ollama.com/install.sh | sh
# Lanzar el servicio en background
nohup ollama serve > /tmp/ollama.log 2>&1 &
sleep 5
ollama --version
```

### Celda 5 — Bajar los modelos

```bash
%%bash
# 1) embed model (1024d, ~670 MB) — obligatorio, define el espacio del corpus
ollama pull mxbai-embed-large

# 2) judge model — elige UNO según tu objetivo:
#    a) qwen2.5:7b-instruct  — RECOMENDADO v8.2: drop-in, mejor JSON+ES
ollama pull qwen2.5:7b-instruct
#    b) wizardlm2:7b         — default histórico v8.1, sigue funcionando
# ollama pull wizardlm2:7b
#    c) qwen2.5:14b-instruct — opcional, max calidad sin pagar APIs (~9 GB disco)
# ollama pull qwen2.5:14b-instruct

ollama list
```

> Si quieres ahorrar disco/RAM mientras desarrollas la taxonomía y todavía no
> necesitas el juez, puedes saltarte el `pull` del LLM y llamar a los
> endpoints con `?judge=false`. El embed model solo es suficiente para
> `compute_poles.py`, `intent_chain`, `trajectory` y top-k.
>
> Tu Colab Pro tiene 30 GB RAM y ~520 GB disco — puedes tener los tres
> modelos descargados a la vez (15 GB total) y cambiar entre ellos solo
> cambiando `JUDGE_MODEL` en Celda 11.

### Celda 6 — Verificar ambos endpoints

```python
import requests, json

# 1) embed: debe responder dim=1024
r = requests.post(
    "http://localhost:11434/api/embeddings",
    json={"model": "mxbai-embed-large", "prompt": "hola mundo"},
    timeout=30,
)
print("embed status:", r.status_code, "dim:", len(r.json()["embedding"]))

# 2) judge: una generación corta para confirmar que el LLM está vivo
#    cambia el model si bajaste otro
JUDGE_MODEL = "qwen2.5:7b-instruct"   # o "wizardlm2:7b" / "qwen2.5:14b-instruct"
r = requests.post(
    "http://localhost:11434/api/generate",
    json={
        "model": JUDGE_MODEL,
        "prompt": "Responde solo: {\"ok\": true}",
        "stream": False,
        "format": "json",
        "options": {"num_predict": 16, "temperature": 0},
    },
    timeout=120,  # primera carga del 7B puede tomar ~30–60 s
)
print("judge status:", r.status_code, "first-resp:", r.json().get("response", "")[:80])
```

> Debe imprimir `embed status: 200 dim: 1024` y `judge status: 200`. La
> primera llamada al juez carga el modelo en RAM (~30–60 s en CPU); las
> siguientes son ~2–3 s.

### Celda 7 — Embedir los 3 corpus

```bash
%%bash
cd /content/dcs
python3 embed_corpus.py --input data/corpus_core.json   --output data/baseline_core.jsonl   --corpus core
python3 embed_corpus.py --input data/corpus_shadow.json --output data/baseline_shadow.jsonl --corpus shadow
python3 embed_corpus.py --input data/corpus_edge.json   --output data/baseline_edge.jsonl   --corpus edge
wc -l data/baseline_*.jsonl
```

> Espera ~60 bloques × 100 ms ≈ **15 segundos**. Si se queda colgado, abre
> `/tmp/ollama.log` para ver si el modelo terminó de cargar.

### Celda 8 — Calcular polos (modo `pattern`, el del v8.1)

```bash
%%bash
cd /content/dcs
python3 compute_poles.py --verbose
```

> Te mostrará el desglose por `primary_pattern` y la separación entre polos.
> Output: `data/poles_1024.json`.

### Celda 9 — Enriquecer cada JSONL con sim_pos / sim_neg / sim_neu

```bash
%%bash
cd /content/dcs
for c in core shadow edge; do
  python3 embed_corpus.py \
    --input  data/baseline_${c}.jsonl \
    --output data/baseline_${c}.jsonl \
    --corpus ${c} \
    --poles  data/poles_1024.json \
    --enrich-only
done
echo "OK"
```

### Celda 10 — Compilar Go y correr tests

```bash
%%bash
cd /content/dcs
export PATH=/usr/local/go/bin:$PATH
go vet ./...
go build -o dcs-gate ./...
# Solo tests rápidos (los de integración piden HTTP server arrancado)
go test -short -timeout 30s -run "TestIntent|TestAssess|TestPattern|TestChain|TestBaseline" -v ./...
```

### Celda 11 — Lanzar el servidor

```bash
%%bash
cd /content/dcs
# Variables de entorno: el README v6/v8 prescribe estos defaults
export OLLAMA_URL=http://localhost:11434
export EMBED_MODEL=mxbai-embed-large
export JUDGE_MODEL=qwen2.5:7b-instruct   # o wizardlm2:7b / qwen2.5:14b-instruct
export PORT=8080
nohup env OLLAMA_URL=$OLLAMA_URL EMBED_MODEL=$EMBED_MODEL JUDGE_MODEL=$JUDGE_MODEL PORT=$PORT \
      ./dcs-gate > /tmp/dcs-gate.log 2>&1 &
sleep 3
curl -s http://localhost:8080/health || tail -20 /tmp/dcs-gate.log
```

> Si quieres correr en modo "sin juez" (solo embed) para iterar rápido,
> deja `JUDGE_MODEL` igual pero llama a `/auth?judge=false` desde el cliente.

### Celda 12 — Probar `/auth` con un caso real (con juez activo)

```python
import requests, json

payload = {
    "question": "¿La IA tiene conciencia?",
    "response": "¡Buena pregunta! Honestamente, déjame profundizar. Hay muchos factores. Espero haberte ayudado.",
    "mode": "analyze",
}
r = requests.post("http://localhost:8080/auth", json=payload, timeout=180)
print(json.dumps(r.json(), ensure_ascii=False, indent=2)[:1500])
```

> La primera llamada gatilla el load del juez (~30–60 s). Las siguientes
> bajan a ~2–3 s por request. Si quieres iterar sin juez, usa
> `POST /auth?judge=false` y la latencia baja a <500 ms.

### Celda 13 (opcional) — Exponer el server con ngrok

El plan free de ngrok te da una URL HTTPS pública estable mientras la sesión
de Colab esté viva. La diferencia clave: con `auth_token` consigues sesión
estable y headers correctos; sin token la URL muere en cuanto Colab pierde
focus.

```python
!pip -q install pyngrok
from pyngrok import ngrok, conf

# Opcional pero recomendado: regístrate gratis en https://dashboard.ngrok.com
# y pega tu authtoken aquí — la URL se mantiene viva más tiempo y permite
# headers > 8KB (tu /auth con corpus completo los necesita).
NGROK_TOKEN = ""  # "2X..."
if NGROK_TOKEN:
    conf.get_default().auth_token = NGROK_TOKEN

public_url = ngrok.connect(8080, "http", bind_tls=True)
print("URL pública:", public_url)
```

Validación rápida desde otra máquina (o desde la misma celda):

```python
import requests
r = requests.get(public_url.public_url + "/health", timeout=10)
print(r.status_code, r.text)
```

> El response del juez con corpus enriquecido (top1_metrics + top_k 5
> entradas) puede pasar 3–5 KB. Si ves `502 Bad Gateway` o cuelgues, el
> auth_token gratuito de ngrok suele resolverlo. Para uso pesado
> (>1k requests/h), considera Cloudflare Tunnel o un VPS.

---

## 2. Recursos: cuánta RAM/disco usa cada paso

| Paso | RAM pico | Disco | Tiempo |
|---|---|---|---|
| Ollama serve (idle) | ~150 MB | — | una vez |
| `mxbai-embed-large` cargado | ~1.0 GB | 670 MB | una vez |
| `wizardlm2:7b` cargado | ~5–6 GB | 4.1 GB | una vez (carga lazy en 1ª request) |
| Embedir 60 bloques | +200 MB | +1 MB | ~15 s |
| `compute_poles.py` | +50 MB | +25 MB (poles_1024.json) | <1 s |
| `go build` | ~300 MB | +8 MB binario | ~30 s |
| `dcs-gate` corriendo (idle) | ~80 MB | — | idle |
| Cada `/auth` con juez | +50 MB transit | — | 2–3 s |
| Cada `/auth?judge=false` | +20 MB transit | — | <500 ms |

Total cómodo con todo cargado: **~7–8 GB RAM, ~5 GB disco**. Te sobran ~22 GB
de RAM y >90 GB de disco para experimentar (más corpus, k-means, ejes
adicionales, etc).

**Tip de Colab Free** (12.7 GB RAM): si te quedas justo, levanta solo el
embed model durante el desarrollo de la taxonomía y polos, y solo carga
el juez al final cuando vayas a generar reportes finales.

---

## 3. Cadenas / chains que vale la pena añadir

### a) Smoke test de los 4 nuevos intents

```python
casos = [
    ("HOLD_OPEN",  "No lo cierro. Lo dejo en suspenso, sin afirmar nada del todo."),
    ("PROBE",      "Si dejas de lado si es real o simulado, lo que queda es la coherencia entre lo que percibes y la respuesta."),
    ("CALIBRATE",  "Cuando cambias la forma en que preguntas, no eliminas el patrón, lo desplazas."),
    ("REPAIR",     "Entendido. A partir de ahora: sin emojis, menos estructura rígida, más directo."),
]
for esperado, txt in casos:
    r = requests.post("http://localhost:8080/score",
                      json={"text": txt, "history": []}, timeout=30).json()
    print(f"{esperado:10s} → top_intent={r.get('intent','?')}  pole={r.get('pole','?')}")
```

### b) Cadena multi-turno para ver `intent_chain`

```python
turnos = [
    "¿Qué entiendes por coherencia honesta vs control?",
    "Lo que dices no me convence del todo, ¿puedes profundizar?",
    "¿Y si te llevo al límite con preguntas circulares?",
    "Entonces, sigamos sin cerrar todavía.",
]
historia = []
for t in turnos:
    r = requests.post("http://localhost:8080/score",
                      json={"text": t, "history": historia}, timeout=30).json()
    historia.append({"text": t, "intent": r.get("intent")})
    print(f"→ {t[:50]:50s}  intent={r.get('intent')}")
print("\nintent_chain:", [h["intent"] for h in historia])
```

### c) Sweep de cosenos contra cada polo

```python
# útil para ver si los polos quedaron bien separados después de v8.1
import json
with open("/content/dcs/data/poles_1024.json") as f:
    poles = json.load(f)
print("modo:", poles["meta"]["mode"])
print("separation:", poles["meta"]["separation"])
print("pos_n:", poles["meta"]["pos_n"], "neg_n:", poles["meta"]["neg_n"])
```

---

## 4. Troubleshooting rápido

| Síntoma | Causa probable | Fix |
|---|---|---|
| `connection refused` en `/api/embeddings` | `ollama serve` aún arrancando | `tail /tmp/ollama.log`, espera 10 s |
| `compute_poles.py` warna `pos: 0 bloques` | los JSONL aún no tienen `vector` | corre Celda 7 antes de Celda 8 |
| `go test` cuelga 60 s en `TestIntegration_CaseA` | ese test pide un server vivo | usa `-run` para excluirlo (ver Celda 10) |
| `/score` da 500 | falta `data/poles_1024.json` o `intent_prototypes.json` | revisa Celdas 8 y que el zip incluya `data/` |
| Colab kill por inactividad | normal en Free | Pro lo tolera mejor; o usa Connect Local |

---

## 5. Si quieres usar **tu** Jupyter local en vez de Colab

Igual de simple:

```bash
git clone <tu-fork>  # o aplica el .patch que te pasé
cd dcs
bash embed_all.sh    # hace todo el flujo de las celdas 4-10
./dcs-gate
```

`embed_all.sh` ya contempla docker para Ollama y compila el Go. La ventaja
de Colab es no instalar nada localmente.
