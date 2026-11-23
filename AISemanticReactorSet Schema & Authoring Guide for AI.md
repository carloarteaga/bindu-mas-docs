# AISemanticReactorSet — Message Schema & Authoring Guide for AI Agents 🧠🤖

This document provides a precise schema definition and authoring contract for AI agents, developers, and tools to construct and publish `AISemanticReactorSet` messages to deploy or update AI reactors dynamically across the Bindu mesh.

---

## 📋 Message Identity & Envelope

In Bindu, a reactor configuration is deployed by publishing an envelope message with `$type: "AISemanticReactorSet"` to the system integration MQTT channel (`bindu/in`) or mesh route.

### Top-Level Envelope Properties

| JSON Path | Data Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `$type` | String | **Yes** | Must be strictly `"AISemanticReactorSet"`. |
| `$group` | String | **Yes** | Typically `"mesh"`. |
| `$producer` | String | **Yes** | The node ID generating/delivering the configuration request (e.g. `"admin"`, `"node-1"`). |
| `$user` | String | **Yes** | Authorization identity (e.g. `"admin"`). |
| `$when` | String | **Yes** | ISO-8601/RFC3339 timestamp of configuration dispatch. |
| `fields` | Object | **Yes** | Container for the configuration payload. |
| `fields.gateway_node_id` | String | **Yes** | The target node ID where these reactors will be spawned and executed. |
| `fields.reactors` | Array of Objects | **Yes** | List of active reactor definitions running in parallel on the target node. |

---

## ⚙️ Reactor Object Schema Details

Each object inside the `fields.reactors` array defines a single independent AI reactor instance:

| Field Name | Data Type | Required | Description / Constraints |
| :--- | :--- | :--- | :--- |
| `reactor_id` | String | **Yes** | Unique alphanumeric identifier for the instance (e.g. `"thermal-guard"`). No spaces or special characters allowed. |
| `api_endpoint` | URL | **Yes** | OpenAI-compatible endpoint (`"http://localhost:11434/v1/chat/completions"`) or TypeSafe System One endpoint (`"https://api.typesafe.ai/v1/systemone"`). |
| `api_key_secret` | String | No | Bearer token for remote API gateways. For secure production use, reference encrypted secrets via `"secret:KEY_NAME"` (e.g. `"secret:TYPESAFE_API_KEY"`). Use placeholders (e.g. `"ollama-local"`) for local unauthenticated LLMs. |
| `model_id` | String | **Yes** | Exact catalog model name on the host (e.g. `"qwen2.5-coder:0.5b"`, `"llama3:8b"`, or `"jev-latest"` / `"jev-1.14.0"` for TypeSafe System One). |
| `provider` | String | No | Optional protocol selector: `"openai"` (System-2 Chat/Completions) or `"typesafe"` (System-1 `/v1/systemone` calibrated decisions). Auto-detected from `api_endpoint` or `model_id` if omitted. |
| `questions` | Object | No | Optional map of TypeSafe System One question primitives (`noul`, `choice`, `score`) to override or extend the questions auto-compiled from `response_message_type`. |
| `max_concurrency` | Integer | **Yes** | Max concurrent asynchronous LLM tasks allowed (usually `1` or `2` for edge hardware). |
| `trigger_message_types` | Array of Strings | **Yes** | List of message types that activate the reactor (e.g. `["FreezerTelemetry"]`). |
| `trigger_condition` | String | No | Safe Google CEL pre-filter evaluated on CPU before executing the reactor (e.g. `"payload.temperature > -15.0"`). |
| `inference_timeout` | Integer | No | Optional custom execution timeout in seconds for LLM inference (default: `90`). Increase to prevent timeouts when loading heavy models on slow CPU/Edge hardware. |
| `history_count` | Integer | **Yes** | Number of previous messages of the trigger type to pull from SQLite database memory. |
| `context_query` | SQL String | No | Parameterized SELECT query to extract advanced metrics from `v_messages`. |
| `response_message_type` | String | **Yes** | The message type schema the LLM result is packed and published as (e.g. `"StorageRiskAssessment"`). |
| `confidence_threshold` | Float | **Yes** | Floating threshold (`0.0` to `1.0`). Inferences below this confidence score are skipped. |
| `mapping_template` | Object | **Yes** | Map translating LLM response fields to the output message fields. Can be `{}` for direct pass-through mode or populated for template transformation. |
| `system_instruction` | String | **Yes** | System persona, logical guards, and output formatting rules for the LLM. |
| `semantic_prompt_template` | String | **Yes** | Prompt template arrangement containing handlebars references to trigger details, history, and queries. |

> [!CAUTION]
> **🚫 Anti-Recursion Invariant for AI Agents (`E_REACTOR_RECURSIVE_LOOP`):**
> When generating or editing an `AISemanticReactorSet`, AI Agents **must ensure that `response_message_type` is NEVER identical to any type in `trigger_message_types`**.
> * Emitting the trigger message type creates an infinite recursive loop ($T \to \text{LLM} \to T \to \text{LLM} \dots$) wasting compute and token costs.
> * Always emit a distinct semantic output type (e.g. `FreezerTelemetry` ➔ `StorageRiskAssessment`).
> * Configurations violating this rule are rejected with error code `E_REACTOR_RECURSIVE_LOOP`.

> **Note on `mapping_template` Modes, Expressions, Type Coercion & Dynamic Aliasing:**
> - **Direct Pass-Through Mode (`{}`):** Use when the LLM prompt instructs the AI model to output keys matching the response message schema. Bindu emits the LLM JSON directly after key normalization, dynamic aliasing, and type coercion.
> - **Template Transformation Mode (Populated):** Use when LLM output keys differ from response schema keys or when evaluating math/fallback expressions (e.g. `"total_flowers": "{{or .fields.total_flowers (add .fields.yellow_flowers .fields.orange_light_flowers .fields.orange_dark_flowers)}}"`).
> - **Dynamic Smart Alias Resolution:** Bindu automatically resolves LLM abbreviations and variations into exact schema property keys using fuzzy/substring matching (e.g., mapping `"yellow"` or `"yellows"` ➔ `"yellow_flowers"`) at runtime based on the target response message type's schema properties. Unmapped temporary keys are cleaned up to prevent validation failures.
> - **Universal Schema-Driven Coercion & Sanitization:** Bindu automatically coerces primitive types bidirectionally based on the JSON Schema of the output message (`integer`, `number`, `boolean`, `string`). It clamps integer/number values to specified `minimum`/`maximum` boundaries, normalizes dates to RFC3339 format, executes smart substring/prefix matching for `enum` lists, and recursively coerces nested `object` and `array` JSON strings. Unparseable inputs fallback to safe schema defaults, ensuring 100% obedience rate at runtime.
> - **Grammar-Guided Decoding (System-2 LLMs):** Bindu automatically constructs a dynamic `json_schema` payload in `response_format` with `strict: true` and `additionalProperties: false`, constraining OpenAI, Ollama, and vLLM backends at token generation time.
- **System One Hybrid Decision Compiler (TypeSafe AI / `Jev`):**
  When `provider: "typesafe"` is set (or `api_endpoint` targets `/v1/systemone` or `model_id` starts with `jev-`), Bindu acts as a **Hybrid Semantic Compiler**:
  1. **Zero-Config Auto-Compilation:** Bindu inspects the target `response_message_type` schema properties and automatically compiles parallel System One questions:
     - `boolean` property $\rightarrow$ `noul` question (returns calibrated probability `[0.0, 1.0]` in `resp.<field>_noul` and boolean decision in `resp.<field>`).
     - `string` with `enum` property $\rightarrow$ `choice` question (returns winning option in `resp.<field>`, per-option distribution in `resp.<field>_probabilities`, and calibrated certainty in `resp.<field>_confidence`).
     - `integer` / `number` property $\rightarrow$ `score` question (evaluates a 5-level ordered rubric and scales the continuous score to `[minimum, maximum]`).
  2. **Explicit Override (`questions`):** Authors may optionally pass a `questions` map in the reactor definition to customize `instructions` and `criteria` rubrics for `noul`, `choice`, or `score` primitives.
  3. **Calibrated Confidence Gate & CEL Synthesis:** Because System One models do not generate unconstrained prose (`string` without `enum`), any free-text explanation fields in `response_message_type` should be synthesized deterministically in `mapping_template` using Google CEL expressions over `resp.*` and `payload.*`.
> - **Designing for Polymorphic Triggers (Multiple Message Types):**
>   When subscribing to multiple message types (e.g., `["CountFlowersByColor", "ManualGardenSurvey"]`) with different schemas:
>   - **Conditional Prompting & Mapping Templates:** Use Go `text/template` condition blocks in the `semantic_prompt_template` and `mapping_template` to extract values conditionally (e.g., `{{if .fields.yellow_flowers}}{{.fields.yellow_flowers}}{{else if .fields.yellow_count}}{{.fields.yellow_count}}{{else}}0{{end}}`).
>   - **Safe CEL Expressions (`trigger_condition`):** Use type checks (e.g., `head["$type"] == "CountFlowersByColor"`) and the macro `has(payload.field_name)` to prevent execution errors when evaluating fields that are not present in all message schemas. For example: `(head["$type"] == "CountFlowersByColor" && has(payload.yellow_flowers) && payload.yellow_flowers > 5) || (head["$type"] == "ManualGardenSurvey" && has(payload.yellow_count) && payload.yellow_count > 5)`.

---

## 🛠️ Prompt Templating & Context Placeholders

The `semantic_prompt_template` is compiled using Go/Handlebars templating. AI engines can reference three context variables during compilation:

1.  `{{.trigger_event}}`: The raw JSON object of the event that activated the reactor.
    *   *Usage:* `{{index .trigger_event "fields" "temperature"}}`
2.  `{{.history}}`: A JSON array of historical messages of the trigger type (newest first).
    *   *Usage:* `{{range .history}} Temp: {{index . "fields" "temperature"}} {{end}}`
3.  `{{.context_query_results}}`: The output array of the `context_query` if defined.

---

## 📋 Complete Production JSON Example

AI Agents can use the following complete, validated message block as a baseline template:

> [!WARNING]
> **Avoid Backslash Copy-Paste Errors:**
> When copying the `context_query` SQL string from the JSON block below to paste into the **Context Query** field of the Console UI text box, make sure to remove any backslashes (`\`). Write normal double quotes: `"$type"` and `"$producer"`. The backslashes (`\"`) are only used for escaping quotes inside raw JSON files.

```json
{
  "$type": "AISemanticReactorSet",
  "$group": "mesh",
  "$producer": "node-1",
  "$user": "admin",
  "$when": "2026-06-29T14:46:00.000Z",
  "fields": {
    "gateway_node_id": "node-1",
    "reactors": [
      {
        "reactor_id": "freezer-cold-chain-guard",
        "api_endpoint": "http://192.168.12.160:11434/v1/chat/completions",
        "api_key_secret": "ollama-local",
        "model_id": "qwen2.5-coder:0.5b",
        "max_concurrency": 2,
        "inference_timeout": 90,
        "trigger_message_types": ["FreezerTelemetry"],
        "history_count": 5,
        "context_query": "SELECT raw_json FROM v_messages WHERE \"$type\" = 'FreezerTelemetry' AND \"$producer\" = ? ORDER BY \"$when\" DESC LIMIT 5",
        "response_message_type": "StorageRiskAssessment",
        "confidence_threshold": 0.75,
        "mapping_template": {
          "risk_level": "{{.fields.risk_level}}",
          "estimated_minutes_to_spoilage": "{{.fields.estimated_minutes_to_spoilage}}",
          "root_cause_prediction": "{{.fields.root_cause_prediction}}",
          "recommended_action_code": "{{.fields.recommended_action_code}}",
          "target_producer": "{{index .trigger_event \"$producer\"}}"
        },
        "system_instruction": "You are a cold chain monitoring AI agent protecting vaccine storage. You analyze temperature trends and output a single valid JSON object following the StorageRiskAssessment schema. Do not output any markdown formatting, preambles, or explanations. Only raw JSON. IMPORTANT RULES: 1. If the latest temperature is above -15 C or the door is open, risk_level MUST be 'CRITICAL'. Otherwise, it is 'NOMINAL'. 2. 'estimated_minutes_to_spoilage' MUST be a valid integer (e.g. 15). Never use a string. 3. 'root_cause_prediction' MUST be a non-empty string explaining the cause. Never use an integer. 4. 'recommended_action_code' MUST be a non-empty string. Never use an integer.",
        "semantic_prompt_template": "Analyze the risk for Freezer '{{index .trigger_event \"$producer\"}}' at locagram '{{index .trigger_event \"$where\"}}'.\n\nTrigger Event:\nTemperature: {{index .trigger_event \"fields\" \"temperature\"}} C\nDoor Open: {{index .trigger_event \"fields\" \"door_open\"}}\n\nHistorical trend of FreezerTelemetry (newest first):\n{{range .history}}\n- At {{index . \"$when\"}}: Temp: {{index . \"fields\" \"temperature\"}} C\n{{end}}\n\nOutput a JSON object:\n{\n  \"risk_level\": \"NOMINAL\" | \"LOW\" | \"MEDIUM\" | \"HIGH\" | \"CRITICAL\",\n  \"estimated_minutes_to_spoilage\": <int>,\n  \"root_cause_prediction\": \"<string>\",\n  \"recommended_action_code\": \"<string>\"\n}"
      }
    ]
  }
}
```

---

## 🎛️ Command Line Management via Bindu Studio

AI models and system operators can also import and export active reactor configurations directly using `bindu-studio`.

### 1. Export Configurations
Retrieves the most recent `AISemanticReactorSet` configuration message for each active node in the database and saves them to a JSON file.

```bash
# Interactive selection of configurations to export
bindu-studio semantic-ai-reactor export --auth console:SuperSecret1

# Non-interactive export of all active configurations
bindu-studio semantic-ai-reactor export --auth console:SuperSecret1 --all --out-path ./export-reactors.json
```

### 2. Import & Deploy Configurations (Full Overwrite)
Parses an exported JSON file containing one or more configurations, and replays them back into the mesh via MQTT to reconfigure the gateways.

```bash
# Interactive selection of configurations to import
bindu-studio semantic-ai-reactor import --in-path ./export-reactors.json --auth console:SuperSecret1

# Non-interactive import of all configurations
bindu-studio semantic-ai-reactor import --in-path ./export-reactors.json --auth console:SuperSecret1 --all
```

### 3. Merge & Incrementally Deploy Configurations (Safe Merge)
Merges new or updated reactor definitions with the active set running on the target gateway node, preserving any other existing reactors.

```bash
# Merge configurations into target nodes
bindu-studio semantic-ai-reactor merge --in-path ./new-reactors.json --auth console:SuperSecret1 --all
```

### 4. Inspect Deployment Status
Displays the live operational state of AI reactors running across all mesh nodes.

```bash
bindu-studio semantic-ai-reactor status --auth console:SuperSecret1
```

