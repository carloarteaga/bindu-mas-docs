Por favor estudia en detalles todos los tutoriales que existen en public/console/docs. 
Estudia todos los documentos referidos por public/console/docs/AGENTS.md
Esto que te pongo a continuación es una idea de como crear un tutorial que abarque el standard de RFID EPC incluyendo:
LLRP protocol simulator (simula un rfid reader) rfid-reader-simulator
Simula epc encoding tags (11 standards de EPC)
Simula la información de la s antenas como read power etc (dando la capacidad de demostrar cómo evaluar el funcionamiento de las antennas)
Simula 4 antennas
Un Bindu BPR plugin bpr-rfid-epc, que implementa:
 El protocol LLRP para hablar con el simulador
La decodificación/decoding de los EPC tags simulados por el rfid-reader-simulator
Genera un message-type por cada decoding soportado.
El objetivo de esta tarea es producir el tutorial-plan.md , toda la documentación en inglés.
Los comentarios y sugerencias siguientes son solo sugerencias , la decisión del plan final es tuya. Crea el tutorial en el directorio public/console/docs/rfid-epc-tutorial.


=======

Actúa como Arquitecto de Sistemas y Desarrollador Go Core para Bindu.
Tu tarea es diseñar, implementar y empaquetar un nuevo BPR Plugin nativo llamado `bindu-bpr-llrp-reader`. Este plugin actuará como un puente para integrar hardware RFID EPC Global (RAIN RFID) al ecosistema Bindu utilizando el estándar LLRP (Low Level Reader Protocol).

**Objetivo de la Solución:**
El plugin debe permitir a Bindu conectarse a un lector RFID físico, procesar el flujo continuo de lecturas (tags) y emitir eventos semánticos puros al mesh. Para facilitar el desarrollo y las pruebas, la solución debe incluir un Simulador LLRP integrado.

**Requisitos Arquitectónicos:**

1. **El Simulador LLRP (Mock Reader):**
   - Implementa un servidor TCP ligero en Go que escuche en el puerto estándar LLRP (5084).
   - Debe ser capaz de aceptar una conexión, responder a comandos básicos (`ADD_ROSPEC`, `ENABLE_ROSPEC`) y comenzar a inyectar tramas binarias ficticias de tipo `RO_ACCESS_REPORT` con códigos EPC (cadenas HEX).
   - El simulador se puede ejecutar mediante una bandera en el binario (ej. `--mode=simulator`) o como un comando auxiliar, para que los desarrolladores validen la topología antes de tocar el metal.

2. **El BPR Plugin (LLRP Client):**
   - Debe operar como un proceso aislado (BPR) comunicándose con Bindu vía `stdin`/`stdout`.
   - Se conectará vía TCP a la IP/Puerto del lector LLRP (físico o al simulador).
   - Manejará la concurrencia nativa (Goroutines) para escuchar el streaming de red sin bloquearse.
   - Traducirá las tramas LLRP (`RO_ACCESS_REPORT`) extrayendo el EPC y la antena, emitiendo un payload JSON validado (MessageType, ej. `RfidTagRead`) hacia `stdout`.

3. **Referencias de Ecosistema Go:**
   - Para el parsing y codificación binaria de LLRP, analiza implementaciones maduras en el ecosistema Go. Toma como referencia arquitectónica el manejo de red y estructuras de proyectos como el de EdgeX Foundry (`github.com/edgexfoundry/device-rfid-llrp-go`) u otros parsers LLRP open-source en Go.
   - Extrae solo lo estrictamente necesario para decodificar el EPC, manteniendo el binario del plugin ultraligero y libre de dependencias masivas.

**Entregables Esperados:**
1. Un Plan de Implementación paso a paso.
2. La definición del esquema JSON (MessageType) que emitirá el plugin.
3. El código fuente en Go tanto para el cliente (BPR plugin) como para el simulador TCP.
4. Instrucciones para compilar y ejecutar el simulador localmente para probar el flujo completo hacia Bindu.

Por favor, asegúrate de optimizar el manejo de memoria (evitar recolección de basura excesiva en el bucle de lectura), ya que los lectores RFID pueden emitir miles de tags por segundo en el Edge.


definir tipos de mensajes distintos (ej. EpcSgtinRead, EpcSsccRead, EpcGlnRead)

el plugin debe tener una función DecodeEPC(hexString string) que haga este desempaquetado a nivel de bits. De esta manera, el flujo dentro de tu BPR Plugin queda perfecto:

El paquete LLRP recibe la lectura y extrae el EPC en hexadecimal bruto.

El decodificador bit a bit de Agy lee el header y reconoce que es un SGTIN.

El plugin empaqueta los datos decodificados en el esquema JSON EpcSgtinRead

========
Tomo exactamente 46 minuitos a Agy (AI Developre Agent) terminar la solucion completa.


=====
Respuesta de Agy al terminar

e ha completado con éxito el diseño, desarrollo, validación y empaquetado del nuevo BPR Plugin nativo (bindu-rfid-epc-
  plugin), el Simulador LLRP / Mock Reader (simulate_dock_portal.go), y el paquete tutorial completo de Sistema Multi-Agente
  (MAS) bajo el estándar RFID EPC Global (RAIN RFID) y LLRP, ubicado en .
  ──────
  ### 📦 Resumen de Artefactos Producidos (The Rule of 6 Artifacts)
    public/console/docs/rfid-epc-tutorial/
    ├── 🔌 bindu-rfid-epc-plugin/          # [Artefacto 5] BPR Plugin nativo en Go puro (Zero-Alloc)
    │   ├── bitreader.go                  # Decodificador bit-a-bit de los 11 estándares EPC
    │   ├── llrp.go                       # Framing binario LLRP, TLV Parser y comandos ROSpec
    │   ├── main.go                       # Cliente LLRP, loop de reconexión y emisor de eventos SDK
    │   ├── plugin.json                   # Manifiesto BPR v1 con capacidades y tipos in/out
    │   └── README.md
    ├── 📜 poc_message_types.json          # [Artefacto 1] 15 esquemas JSON Schema declarativos
    ├── 🧠 poc_semantic_reactor_config.json # [Artefacto 2] AISemanticReactorSet (SLM Manifest Reconciler)
    ├── ⚡ poc_cel_reactor_config.json      # [Artefacto 3] CELReactorSet (Cinemática espacial 4-antenas & Noise Gate)
    ├── 📊 poc_dashboards.json             # [Artefacto 4] 3 DashboardDefinitions (Transit, EPC Stream, RF Matrix)
    ├── 🚀 rfid_mas_solution.json          # [Artefacto 6] Manifiesto MASSolutionSet y OTA Deploy script
    ├── 🛠️ deploy_mas_solution.sh          # Protocolo universal de despliegue y validación preflight
    ├── 📡 simulate_dock_portal.go         # Servidor TCP LLRP en puerto 5084 con 4 antenas y 11 EPCs
    ├── 🏃 simulate_dock_portal.sh         # Script de arranque del simulador
    ├── 📋 tutorial-plan.md                # Especificación técnica maestra en inglés
    └── 📖 tutorial_guide.md               # Guía completa paso a paso en inglés
  ──────
  ### 🔬 1. Decodificación Bit a Bit de los 11 Estándares EPC

  El plugin implementa un BitReader sin asignación dinámica de memoria en el hot path, que extrae y traduce directamente a
  MessageTypes fuertemente tipados:

  1. SGTIN-96 (0x30): Cajas y productos unitarios retail → EpcSgtinRead
  2. SSCC-96 (0x31): Pallets logísticos y contenedores → EpcSsccRead
  3. SGLN-96 (0x32): Ubicación de puertas dock y muelles → EpcSglnRead
  4. GRAI-96 (0x33): Totes, cajas plásticas y pallets retornables → EpcGraiRead
  5. GIAI-96 (0x34): Flota de montacargas y maquinaria fija → EpcGiaiRead
  6. GSRN-96 (0x35): Carnets de operadores y conductores → EpcGsrnRead
  7. GDTI-96 (0x3C): Manifiestos de aduana y certificados → EpcGdtiRead
  8. CPI-96 (0x3D): Componentes automotrices y aviónicos → EpcCpiRead
  9. SGCN-96 (0x3E): Cupones y vales de alto valor → EpcSgcnRead
  10. ITIP-96 (0x38): Rollos textiles y madera fraccionada → EpcItipRead
  11. GID-96 (0x35/GID): Piezas industriales en bruto → EpcGidRead
  ──────
  ### 📡 2. Simulador LLRP del Portal de 4 Antenas (simulate_dock_portal.go)

  • Servidor TCP RFC-compliant en puerto estándar 5084.
  • Maneja el ciclo completo de comandos LLRP: GET_READER_CAPABILITIES, ADD_ROSPEC, ENABLE_ROSPEC, START_ROSPEC, STOP_ROSPEC,
  KEEPALIVE.
  • Simulación física de tránsito:
      • INBOUND: Antenas 1 y 2 leen primero; Antenas 3 y 4 leen 350 ms después con alta potencia.
      • OUTBOUND: Antenas 3 y 4 leen primero; Antenas 1 y 2 leen después.
      • STRAY: Lecturas estáticas de baja potencia en el área de preparación.

  ──────
  ### ✅ 3. Verificación y Validación Offline

  Todos los componentes superaron el portal de compilación y validación sin advertencias:

  • CEL Reactor: go run ./cmd/bindu-studio cel-reactor validate → OK
  • Semantic AI Reactor: go run ./cmd/bindu-studio semantic-ai-reactor validate → OK (0 warnings)
  • Dashboards: go run ./cmd/bindu-studio dashboard validate → OK (3 dashboards válidos)
  • Solution Manifest: go run ./cmd/bindu-studio solution validate --strict → OK (15 schemas, 3 CEL, 1 SLM, 3 Dashboards, 1
  Plugin)
  • Test Suite: go test ./... -short → OK
  • HTMX Validation Scripts: validate-htmx-migration.sh & validate-message-types-htmx.sh → OK
  ──────
  ### 🚀 Instrucciones de Ejecución Local

  1. Iniciar el Simulador LLRP (Terminal 1):
    cd public/console/docs/rfid-epc-tutorial
    ./simulate_dock_portal.sh

  2. Desplegar la Solución MAS (Terminal 2):
    cd public/console/docs/rfid-epc-tutorial
    ./deploy_mas_solution.sh --yes

  3. Explorar en la Consola Web:
  Navegar a http://localhost:8080/console/, seleccionar el filtro MAS "Industrial RFID LLRP Dock Portal & 11-Standard EPC
  Digital Twin" y ver los 3 nuevos dashboards en tiempo real.
