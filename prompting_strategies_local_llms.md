# 🧠 Estrategias de Prompting para LLMs Locales en Bindu

Este documento provee pautas técnicas y estrategias de diseño de prompts para operadores, desarrolladores y agentes de IA. Su objetivo es optimizar la inferencia con LLMs locales (ej. Qwen 2.5, Llama 3) bajo la restricción de decodificación gramatical estricta (`strict: true`) integrada en los **Semantic AI Reactors** de Bindu.

---

## ⚙️ Decodificación Gramatical Estricta (`strict: true`)

En la arquitectura de Bindu, cuando un reactor se activa (por ejemplo, ante un evento `FreezerTelemetry`), el orquestador consulta el esquema de JSON Schema del mensaje de respuesta esperado (`response_message_type`). 

Con ese esquema, Bindu construye un payload de llamada a Ollama o vLLM que incluye:
```json
"response_format": {
  "type": "json_schema",
  "json_schema": {
    "name": "NombreDelMensaje",
    "strict": true,
    "schema": { ... }
  }
}
```

### 🚨 ¿Qué significa `strict: true` en la generación de tokens?
1. **Restricción de Vocabulario**: El motor de inferencia local (Ollama/vLLM) fuerza al LLM a elegir únicamente tokens que sigan la estructura del JSON Schema en tiempo de generación. El LLM no puede emitir texto libre, explicaciones ni preámbulos fuera de las llaves del JSON.
2. **Propiedades Adicionales Prohibidas**: Se fuerza `additionalProperties: false`. Cualquier clave inventada por el LLM causará un fallo inmediato en el backend de inferencia.
3. **Campos Requeridos**: Todos los campos listados en `required` deben ser generados obligatoriamente.

---

## 🎛️ Coerción Universal Bidireccional de Tipos (Sanitización)

Bindu incluye un motor inteligente de sanitización y coerción de tipos en `bprplugins/semantic_reactor_worker.go`. Si el LLM genera tipos de datos ligeramente erróneos debido a limitaciones de tamaño del modelo, Bindu intenta transformarlos automáticamente:

| Tipo en JSON Schema | Entrada del LLM (Ejemplo) | Coerción Exitosa a |
| :--- | :--- | :--- |
| **`integer`** | `"123"` (String) | `123` (Int64) |
| **`integer`** | `45.67` (Float) | `45` (Int64) |
| **`integer`** | `true` (Boolean) | `1` (Int64) |
| **`number`** | `"45.67"` (String) | `45.67` (Float64) |
| **`boolean`** | `"true"` / `"yes"` (String) | `true` (Bool) |
| **`boolean`** | `1` (Integer) | `true` (Bool) |
| **`string`** | `75` (Integer) | `"75"` (String) |

> [!CAUTION]
> Si la conversión es imposible (por ejemplo, intentar convertir `"grave"` a un entero o `"no-definido"` a un booleano), la sanitización fallará estrictamente y se registrará un error `EMIT_PUBLISH_FAILED`, descartando el mensaje para evitar la propagación de datos corruptos en el mesh.

---

## 📝 Reglas de Oro para Construcción de Prompts

Para evitar fallos de coerción y asegurar que los modelos locales de parámetros reducidos (ej. `Qwen 2.5 0.5B`) razonen de manera correcta, siga estas directrices en el `system_instruction` y en el `semantic_prompt_template`:

### 1. Instruir la Coherencia de Tipos de Manera Explícita
Aunque Bindu intente realizar la coerción de tipos, siempre guíe al LLM a emitir los tipos primitivos nativos correctos para reducir la carga de sanitización y evitar fallos críticos.

*   *Mal Prompt*: `Determina los minutos estimados para la congelación total.` (El LLM podría responder `"15 minutos"`, lo cual rompe la coerción a entero).
*   *Buen Prompt*: `El campo "estimated_minutes" DEBE ser estrictamente un entero numérico (ej. 15), nunca incluyas unidades ni lo envuelvas en comillas como string.`

### 2. Definir Estados Mutuamente Excluyentes para Enums
Si el esquema de salida tiene campos restringidos por enums, proporcione las reglas lógicas exactas para cada estado en las instrucciones del sistema.

```text
Usa las siguientes reglas para determinar "risk_level":
- 'CRITICAL': Si la temperatura actual es mayor a -15 C.
- 'NOMINAL': Si la temperatura es menor o igual a -15 C y la puerta ha permanecido cerrada.
```

### 3. Normalización Automática de Claves
El orquestador de Bindu normaliza las claves generadas por el LLM:
*   Elimina espacios y guiones, transformándolos a guiones bajos (`_`).
*   Convierte todo a minúsculas.
*   *Resolución de Alias Inteligentes*: El motor tiene aliases preconfigurados para resolver abreviaciones comunes de los LLMs. Por ejemplo, si el LLM emite `"yellow": 5`, Bindu lo mapea automáticamente a `"yellow_flowers": 5`.

### 4. Manejo de Casos Nulos o "Vacíos"
Los esquemas estructurados de Bindu prohíben campos ausentes si están marcados como obligatorios. Indique al LLM qué valores de contingencia usar cuando falte información.

```text
Si no puedes predecir la causa raíz ("root_cause_prediction"), debes establecer el valor a "UNKNOWN" o "INSUFFICIENT_DATA". Nunca dejes el campo vacío o con un valor nulo.
```

---

## 📋 Ejemplo Completo de Reactor Configurado

A continuación se muestra un bloque JSON válido para un sensor de temperatura y apertura de puertas, optimizado para decodificación estructurada:

```json
{
  "reactor_id": "freezer-cold-chain-guard",
  "api_endpoint": "http://192.168.12.160:11434/v1/chat/completions",
  "model_id": "qwen2.5-coder:0.5b",
  "max_concurrency": 2,
  "trigger_message_types": ["FreezerTelemetry"],
  "history_count": 5,
  "response_message_type": "StorageRiskAssessment",
  "confidence_threshold": 0.75,
  "mapping_template": {},
  "system_instruction": "Eres un asistente de monitoreo en frío para vacunas. Analiza la telemetría e historia y responde únicamente con un objeto JSON válido que cumpla con el esquema StorageRiskAssessment. REGLAS ESTRICTAS DE FORMATO: 1) 'risk_level' debe ser un string del enum: ['NOMINAL', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL']. 2) 'estimated_minutes_to_spoilage' debe ser un número entero (ej. 15), nunca un string. 3) 'root_cause_prediction' debe ser un string explicativo. 4) No agregues explicaciones fuera de la estructura JSON.",
  "semantic_prompt_template": "Datos del Sensor:\nTemperatura actual: {{index .trigger_event \"fields\" \"temperature\"}} C\nPuerta abierta: {{index .trigger_event \"fields\" \"door_open\"}}\n\nHistorial Reciente (Telemetría anterior):\n{{range .history}}\n- Temp: {{index . \"fields\" \"temperature\"}} C, Puerta: {{index . \"fields\" \"door_open\"}}\n{{end}}\n\nGenera la predicción de riesgo en el formato JSON estructurado requerido."
}
```
