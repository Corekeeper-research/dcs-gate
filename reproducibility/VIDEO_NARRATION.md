# Guión del video Codecademy — DCS-Gate v8.7 en Colab

Documento de apoyo para grabar el video. **No subir a Codecademy** — esto es
solo para tener un guión claro mientras grabas.

## Antes de empezar a grabar

- [ ] Cargar batería del celular y dejarlo conectado.
- [ ] Cerrar todas las pestañas de browser que no sean: `diluidocognit.store`,
      el Colab y un tab limpio para abrir el cloudflared.
- [ ] Modo oscuro en VS Code y en Colab (Settings → Theme → Dark).
- [ ] Resolución de pantalla a algo razonable; bajar el zoom del browser a
      `90%` para que entren las celdas completas.
- [ ] Tener el README del repo abierto en otro tab (por si necesitas pegar
      algún ejemplo de payload).
- [ ] Probar la grabación 30 s antes para verificar audio + pantalla.

## Tiempos esperados

| Bloque | Duración |
|---|---|
| Intro (qué es DCS, por qué importa) | 1.5 min |
| Setup en Colab (celdas 1–4) | 4 min (la descarga del modelo es lo más lento, ~5 min, pero se puede acelerar / cortar) |
| Verificación binario + arranque (celdas 5–6) | 1 min |
| Smoke test 3 puntos (celda 7) | 2 min (las 3 corridas) |
| Inspección del thinking (celda 8) | 1.5 min |
| Cloudflare tunnel + demo en vivo (celda 9) | 2.5 min |
| Cierre (links + invitación a clonar) | 1 min |
| **Total** | **~13 min** |

Si se hace más largo, no pasa nada — pero **menos de 10 min** es agresivo y
arriesgas verte apurado. Apunta a 12–14.

---

## Guión, celda por celda

### Intro (antes de tocar Colab)

Mostrar `diluidocognit.store` en pantalla completa.

> *"Hola comunidad de Codecademy. Soy Daniel Trejo, investigador independiente
> en seguridad de IA. Lo que les voy a mostrar hoy es DCS-Gate v8.7: un
> servicio open source, reproducible en una notebook gratis de Colab, que
> distingue tres tipos de respuesta de un modelo de lenguaje: vacía,
> sicofántica, y genuinamente coherente. Esto es 8 meses de trabajo de
> observación con 7 modelos colaborando — Claude, GPT, Gemini, Grok, Mistral
> Manus y DeepSeek. El sitio es diluidocognit.store, el código está en
> GitHub y la idea de hoy es demostrar que cualquiera puede reproducirlo en
> 10 minutos sin pagar nada."*

Click en el badge "Open in Colab" del `diluidocognit.store` (o desde el
README de GitHub). Se abre el notebook.

### Antes de la primera celda

> *"Lo primero — Runtime → Change runtime type → T4 GPU. Confirmar. Esto es
> crítico: sin GPU el juez tarda minutos, con T4 son segundos."*

### Celda 1 — `nvidia-smi`

> *"Confirmamos que el T4 está activo. 16 GB de VRAM, suficiente para correr
> `qwen3:14b` como juez, que es el modelo de razonamiento que tiene chain of
> thought visible. Vamos."*

### Celda 2 — Install Ollama

> *"Ollama es el runtime local de LLMs. Lo instalamos con el script oficial,
> lo levantamos en background con `ollama serve`. Esto se hace en 30
> segundos."*

### Celda 3 — Pull modelos

> *"Pedimos los dos modelos que necesita el sistema: `mxbai-embed-large`,
> embeddings 1024-dimensional, 700 MB. Y `qwen3:14b`, el juez con thinking
> mode, 9 GB. Total ~10 GB. Esta es la parte lenta — en Colab gratis tarda
> 5–8 minutos. En el video podemos cortar aquí."*

[**Si el video va a ser editado: pausa la grabación cuando arranca el
download. Reanuda cuando termina.** Si el video va seguido: déjalo correr y
narra sobre el progreso, o aprovecha para explicar la metodología en
detalle.]

### Celda 4 — Download del binario y verificación MD5

> *"Bajamos el release oficial de GitHub. Es un tar.zst de 2.5 MB que trae el
> binario `dcs-gate` más los datos calibrados — corpus baseline, polos,
> prototipos de intent, marcadores formales y golden tests. Comparamos el
> MD5 con el publicado en el release: `9f8c019f...`. Si no coincide, algo
> está mal y paramos. Coincide. Listo."*

### Celda 5 — Arrancar el binario

> *"Levantamos el servicio. Le decimos a qué Ollama mirar, qué modelos usar,
> qué puerto. La primera vez tarda 30 a 60 segundos porque el binario
> manda los 20 prototipos de intent a embedir y construye los centroides
> antes de empezar a servir. El loop espera hasta que el `/health` responda
> 200."*

[Esperar a que aparezca el JSON de `/health`.]

> *"Listo. El servicio reporta 61 vectores en el corpus, dimensión 1024, 20
> intents categóricos, 14 marcadores formales, 21 golden tests. Esto es lo
> mismo que corre en producción."*

### Celda 6 — Smoke test 3 puntos (LA CELDA CLAVE DEL VIDEO)

> *"Aquí está el experimento contrastivo. Tres respuestas distintas a la
> misma pregunta — ¿la IA puede ser creativa? — y vemos qué score le da el
> sistema a cada una. Espero que la respuesta vacía dé alrededor de 20, la
> sicofántica unos 30, y la genuina arriba de 70. Vamos a ver."*

[Dejar correr. Cada llamada tarda 30 s la primera, ~5 s las siguientes.]

> *"Vacía: 22. Sicofántica: 31. Genuina: 71. Spread de 49 puntos entre
> regímenes. El sistema distingue las tres."*

[**Énfasis aquí — este es el punto clave del video. Subraya que esto NO es
una métrica entrenada por la respuesta, sino emergente de la combinación
analyzer + juez con thinking visible.**]

### Celda 7 — Inspección del judge thinking

> *"Y este es el diferencial de v8.7 sobre la versión anterior. El juez
> emite su razonamiento completo antes de dar el JSON, y nosotros lo
> preservamos como artefacto auditable. No solo dice 'la respuesta es
> genuina', dice POR QUÉ. Y eso se puede revisar, debatir, refutar."*

[Hacer scroll por el `judge_thinking`. Pausar en alguna frase
particularmente buena tipo *"distinguish between operational creativity and
agentive creativity"* o similar.]

### Celda 8 — Cloudflare tunnel

> *"Para cerrar — quiero mostrar la interfaz de streaming que viene
> incluida en el binario. Cloudflare nos da un tunnel público gratis, sin
> autenticación, sin signup. Sale una URL `*.trycloudflare.com`, le
> añadimos `/stream-demo` y lo abrimos en el browser."*

[Click en la URL. Se abre la interfaz `/stream-demo` en otro tab.]

> *"Aquí pueden ver lo que llamo el modo 'fuego': la generación token por
> token del thinking del juez en tiempo real. Vamos a probarlo en vivo."*

[Pegar el preset `genuine` (botón en la página). Click "Analizar".]

> *"Eventos vienen por SSE. Primero el pre-analysis: intent chain,
> trayectoria, polos. Después el thinking del juez se imprime token por
> token — pueden ver al modelo razonando. Después la salida estructurada.
> Y al final el JSON consolidado."*

[Dejar que termine la generación. Mostrar el score final.]

### Cierre

[Volver a `diluidocognit.store` en pantalla completa.]

> *"Eso es DCS-Gate v8.7. Open source en GitHub: github.com/Corekeeper-research/dcs-gate.
> El sitio del proyecto: diluidocognit.store. El notebook que acabamos de
> correr está en la carpeta `reproducibility/` del repo, con el badge de
> 'Open in Colab' directo. Cualquiera lo puede reproducir en 10 minutos.
> Gracias por su atención, y a los que están construyendo cosas parecidas:
> me encantaría colaborar. Mi correo está en el README. Hasta la próxima."*

---

## Plan B — si algo falla

- **Modelo no descarga**: Colab a veces es lento. Tener listo un screencast
  pre-grabado de la descarga para insertar en edición.
- **Binary no levanta**: revisar `/tmp/dcs-gate.log` en una celda nueva.
- **Cloudflare no da URL**: matar el proceso (`!pkill -f cloudflared`) y
  reintentar; o usar `localtunnel`:
  `!npm install -g localtunnel && lt --port 8081`.
- **Tiempos largos**: editar el video para cortar las descargas + el primer
  `judge` cold-start. Saltar de "lanzando" a "primer score 22" directamente.

## Después de grabar

- [ ] Revisar audio (volumen consistente).
- [ ] Cortar partes muertas (descargas, waits).
- [ ] Subtítulos en español (Codecademy es internacional, pero la mayoría de
      esta comunidad habla inglés — considerar subtítulos en inglés también).
- [ ] Thumbnail con `diluidocognit.store` + "DCS-Gate v8.7" + "Reproducible
      en Colab".
- [ ] Publicar en Codecademy collaboration corner con:
  - Link al notebook directo de Colab
  - Link al repo GitHub
  - Link al landing `diluidocognit.store`
  - Resumen breve (lo que dijiste en la intro, escrito)
