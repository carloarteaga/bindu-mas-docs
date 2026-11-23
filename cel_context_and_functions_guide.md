# Google CEL Context & Function Extensions Reference Guide 🧠⚡

Bindu uses **Google Common Expression Language (CEL)** as its unified, sub-microsecond declarative execution engine across all reactors:
* **CEL Declarative Reactor** (`systemplugins/cel_reactor.go`)
* **REST Call Reactor** (`bprplugins/rest_reactor_worker.go`)
* **Semantic AI Reactor Guard Engine** (`bprplugins/semantic_orchestrator.go`)
* **Dry-Run Simulators & API Validation** (`bindurestapi/simulate_rest.go`, `bindurestapi/validate.go`)

Every CEL expression evaluates directly against live, thread-safe, in-memory representations of the active node's runtime configuration, topology, schema registry, and ingress event envelopes with **zero disk I/O**.

---

## 1. 🌐 Context Variables Reference

The following root variables are automatically available in the evaluation environment:

### `head` — Message Header Envelope
Contains the envelope metadata of the triggering message:
* `head["$producer"]` or `head["producer"]`: Origin node or client ID.
* `head["$user"]` or `head["user"]`: Security context user or role.
* `head["$group"]` or `head["group"]`: Target group (e.g., `'mesh'`).
* `head["$when"]` or `head["when"]`: ISO 8601 creation timestamp.
* `head["$type"]` or `head["type"]`: Input message type name.
* `head["$where"]["locagram"]`: Geographical/logical hierarchical locagram of origin.

### `payload` — Message Payload
Contains the application/sensor fields defined in the input message type schema:
```javascript
payload.temperature > 25.0 && payload.status == "CRITICAL"
```
*(For proactive cron schedules, `payload` is an empty map).*

### `db` — Zero-RAG Database Context (SQLite)
Populated when a `context_query` is defined on the reactor:
* `db.rows`: Array of row maps. Access fields via `db.rows[0].fields.column_name`.
* `db.scalar`: The first column value of the first row (e.g. from `SELECT COUNT(...)`).

---

## 2. 🗺️ `mesh.*` — Live Mesh Topology Context

Exposes the active topology (`runtime/config/mesh.json`) loaded in memory:

| Variable | Type | Example Value | Description |
| :--- | :--- | :--- | :--- |
| `mesh.local_id` | `string` | `"node-dell-2"` | Current local node ID. |
| `mesh.root_id` | `string` | `"root"` | Root gateway node ID. |
| `mesh.self.id` | `string` | `"node-dell-2"` | Current node logical ID. |
| `mesh.self.locagram` | `string` | `"/galicia-dell-2"` | Current node locagram path. |
| `mesh.self.is_root` | `bool` | `false` | `true` if current node is root gateway. |
| `mesh.self.connectivity_kind` | `string` | `"gateway"` | Node kind (`"gateway"`, `"reference"`, `"ble_endpoint"`). |
| `mesh.self.platform.os` | `string` | `"linux"` | Operating system of local node. |
| `mesh.self.platform.arch` | `string` | `"amd64"` | Architecture of local node. |
| `mesh.self.plugins` | `list(string)` | `["hello-world-plugin"]` | Active plugins assigned to local node. |
| `mesh.nodes` | `map[string]object` | `mesh.nodes['root']` | Map of all mesh nodes indexed by ID. |

### 🛠️ Custom Mesh Helper Functions

* **`mesh.locagram(node_id)`** $\rightarrow$ `string`:
  Resolves the hierarchical locagram path for any node ID.
  ```javascript
  mesh.locagram('node-dell-2') == '/galicia-dell-2'
  ```
* **`mesh.is_parent(node_id)`** $\rightarrow$ `bool`:
  Returns `true` if the specified `node_id` is the operational parent of the local node in the mesh tree.
  ```javascript
  mesh.is_parent('root')
  ```
* **`mesh.is_root(node_id)`** $\rightarrow$ `bool`:
  Returns `true` if the specified `node_id` is the mesh root gateway.
  ```javascript
  mesh.is_root(head['$producer'])
  ```
* **`mesh.has_plugin(node_id, plugin_name)`** $\rightarrow$ `bool`:
  Returns `true` if the node has the specified plugin assigned in its mesh declaration.
  ```javascript
  mesh.has_plugin('node-dell-2', 'hello-world-plugin')
  ```

---

## 3. ⚙️ `config.*` — Live System Runtime Configuration

Exposes safe, non-sensitive runtime parameters loaded from `binduConfig.json`:

| Variable | Type | Example | Description |
| :--- | :--- | :--- | :--- |
| `config.server.producer_id` | `string` | `"node-dell-2"` | Configured producer identifier. |
| `config.server.description` | `string` | `"Galicia Edge 2"` | Host/node descriptive text. |
| `config.logging.level` | `string` | `"INFO"` | Active system logging level (`"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"`). |
| `config.logging.format` | `string` | `"json"` | Log format (`"json"` or `"text"`). |
| `config.time.ntp_server` | `string` | `"pool.ntp.org"` | Active NTP time synchronization host. |
| `config.maintenance_cron` | `string` | `"0 0 4 * * *"` | Automated maintenance cron schedule. |
| `config.maintenance_reset` | `bool` | `true` | Automated post-maintenance service reset toggle (frees CGO/glibc memory). |

> [!NOTE]
> Sensitive configuration items such as private TLS keys, tokens, and installation IDs are strictly omitted from `config.*` for security and memory safety. To safely inject integration API keys and tokens in CEL, use `secrets.*` or the `secret()` function.

---

## 4. 🔐 `secrets.*` & `secret()` — Encrypted Integration Credentials

Enables declarative REST reactors, headers, and pipelines to dynamically resolve third-party tokens and API keys stored in the encrypted `accessControlConfig.brs` (or environment variables) without hardcoding secrets in message definitions or repository commits.

### Helper Functions

| Function Signature | Description | Return Type | Example |
| :--- | :--- | :---: | :--- |
| `secret(key)` | Resolves secret from encrypted access control or OS environment. | `string` | `secret('TYPESAFE_API_KEY')` |
| `secret(key, fallback)` | Resolves secret with a default fallback value if not configured. | `string` | `secret('ERP_TOKEN', 'mock-token')` |

### Direct Map Access

All configured secrets are also accessible via the `secrets` map:
```javascript
// Access via map syntax or property lookup
'Bearer ' + secrets.TYPESAFE_API_KEY
'Basic ' + secrets['ERP_BASIC_AUTH']
```

---

## 5. 📐 `schema.*` — Live Message Type Registry & Functions

Exposes registered message contracts loaded from `messageTypesConfiguration.json`:

* **`schema.types`** $\rightarrow$ `map[string]object`:
  Lookup metadata for any registered message type:
  ```javascript
  schema.types['RawMarineData'].emoji       // "🌊"
  schema.types['RawMarineData'].version     // "1.0.0"
  schema.types['RawMarineData'].category    // "telemetry"
  schema.types['RawMarineData'].persist     // true
  schema.types['RawMarineData'].description // "Oceanographic data"
  ```

### 🛠️ Custom Schema Helper Functions

* **`schema.is_registered(type_name)`** $\rightarrow$ `bool`:
  Returns `true` if the message type is known and active in the registry.
  ```javascript
  schema.is_registered('RawMarineData')
  ```
* **`schema.has_property(type_name, prop_name)`** $\rightarrow$ `bool`:
  Returns `true` if the registered message type includes the given field property.
  ```javascript
  schema.has_property('RawMarineData', 'chlorophyll_mg_m3')
  ```
* **`schema.property_names(type_name)`** $\rightarrow$ `list(string)`:
  Returns the complete list of property names declared for the specified message type.
  ```javascript
  schema.property_names('RawMarineData').contains('sea_surface_temp_c')
  ```
* **`schema.as_typescript(type_name)`** $\rightarrow$ `string`:
  Generates a compact TypeScript interface declaration matching the active message type schema. Ideal for prompt template dynamic injection (`{{.response_schema}}`).
  ```javascript
  schema.as_typescript('PredictedFishingZone')
  ```
* **`schema.as_json_sample(type_name)`** $\rightarrow$ `string`:
  Generates a formatted JSON sample object with typed default values matching the message type specification.
  ```javascript
  schema.as_json_sample('RawMarineData')
  ```

---

## 5. 🧠 `ai.*` & Timestamp Utility Functions

Equips reactors and pipelines with numeric bounding, text sanitization, dynamic ISO timestamps, and fallback helpers:

| Function Signature | Description | Return Type | Example |
| :--- | :--- | :---: | :--- |
| `now()` | Returns current UTC timestamp in ISO 8601 / RFC3339 format. | `string` | `now()` $\to$ `'2026-09-29T17:47:34Z'` |
| `time.now()` | Alias for `now()` returning current UTC timestamp in RFC3339. | `string` | `time.now()` |
| `ai.clean_json(text)` | Strips markdown code fences (` ```json `), outer text, and extracts valid JSON objects/arrays. | `string` | `ai.clean_json(resp.raw_content)` |
| `ai.clamp(val, min, max)` | Clamps a numeric float or int value between lower and upper bounds. | `double` / `int` | `ai.clamp(resp.score, 0.0, 1.0)` |
| `ai.round(val, decimals)` | Rounds a float value to the specified number of decimal digits. | `double` | `ai.round(resp.lat, 4)` |
| `ai.default(val, fallback)` | Returns `val` if non-empty / non-zero; otherwise returns `fallback`. | `string`/`double`/`int` | `ai.default(resp.id, payload.vessel_id)` |

---

## 6. 🧰 Google CEL Standard Extension Libraries

Bindu activates the complete suite of **Google CEL Standard Extensions** (`github.com/google/cel-go/ext`) across all CEL evaluation environments. These provide rich string transformations, base64 encoding/decoding, advanced math, list/set manipulation, regular expressions, and local variable scoping (`cel.bind`):

### 🔡 6.1 `ext.Strings()` — String Manipulation & Parsing
Provides standard string inspection and transformation methods. In CEL, string functions are called as **methods on the string instance** (e.g., `str.contains(...)`, `str.split(...)`):

| Method / Function | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| **`.contains(substr)`** | `string.contains(string) -> bool` | Checks if substring exists | `'minio_local://raw/file.jpg'.contains('://')` |
| **`.startsWith(prefix)`** | `string.startsWith(string) -> bool` | Checks string prefix | `payload.url.startsWith('https://')` |
| **`.endsWith(suffix)`** | `string.endsWith(string) -> bool` | Checks string suffix | `payload.filename.endsWith('.jpeg')` |
| **`.split(separator)`** | `string.split(string) -> list<string>` | Splits string by delimiter | `'a://b/c'.split('://')[0]` $\to$ `'a'` |
| **`.split(sep, limit)`** | `string.split(string, int) -> list<string>` | Splits with element limit | `'a/b/c/d'.split('/', 2)` $\to$ `['a', 'b/c/d']` |
| **`.substring(start)`** | `string.substring(int) -> string` | Slices from index to end | `'raw/image.png'.substring(4)` $\to$ `'image.png'` |
| **`.substring(start, end)`** | `string.substring(int, int) -> string` | Slices range `[start, end)` | `'PatioMonitor'.substring(0, 5)` $\to$ `'Patio'` |
| **`.indexOf(substr)`** | `string.indexOf(string) -> int` | First index of substring (-1 if not found) | `'raw/img.png'.indexOf('/')` $\to$ `3` |
| **`.lastIndexOf(substr)`** | `string.lastIndexOf(string) -> int` | Last index of substring | `'a/b/c'.lastIndexOf('/')` $\to$ `3` |
| **`.replace(old, new)`** | `string.replace(string, string) -> string` | Replaces occurrences | `'http://'.replace('http', 'https')` |
| **`.trim()`** | `string.trim() -> string` | Strips leading/trailing whitespace | `'  alert  '.trim()` $\to$ `'alert'` |
| **`.lowerAscii()`** | `string.lowerAscii() -> string` | Lowercases ASCII characters | `'MinIO'.lowerAscii()` $\to$ `'minio'` |
| **`.upperAscii()`** | `string.upperAscii() -> string` | Uppercases ASCII characters | `'galicia'.upperAscii()` $\to$ `'GALICIA'` |
| **`.reverse()`** | `string.reverse() -> string` | Reverses string characters | `'abc'.reverse()` $\to$ `'cba'` |
| **`.join(sep)`** | `list<string>.join(string) -> string` | Joins list of strings with separator | `['a', 'b', 'c'].join('/')` $\to$ `'a/b/c'` |
| **`.format(args)`** | `string.format(list) -> string` | Formats template string | `'%s://%s/%s'.format(['minio', 'raw', 'img.png'])` |
| **`strings.quote(str)`** | `strings.quote(string) -> string` | Escapes string safely | `strings.quote('a "b" c')` |

> [!IMPORTANT]
> **Syntax Rule:** In Google CEL, string methods are called as `str.contains('abc')`, **never** `strings.contains(str, 'abc')`.

---

### 🔐 6.2 `ext.Encoders()` — Base64 Encoding & Decoding
Converts raw bytes to Base64/Base64URL and vice versa:

| Function | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| **`base64.encode(b)`** | `base64.encode(bytes) -> string` | Encodes bytes to standard Base64 string | `base64.encode(bytes('hello'))` $\to$ `'aGVsbG8='` |
| **`base64.decode(s)`** | `base64.decode(string) -> bytes` | Decodes Base64 string to bytes | `string(base64.decode('aGVsbG8='))` $\to$ `'hello'` |

---

### ➗ 6.3 `ext.Math()` — Advanced Math Functions
Extends standard arithmetic with boundary, rounding, and bitwise operations:

| Function | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| **`math.greatest(...)`** | `math.greatest(T, ...) -> T` | Returns maximum value among arguments | `math.greatest(10, 42, 5)` $\to$ `42` |
| **`math.least(...)`** | `math.least(T, ...) -> T` | Returns minimum value among arguments | `math.least(10, 42, 5)` $\to$ `5` |
| **`math.ceil(x)`** | `math.ceil(double) -> double` | Smallest integer $\ge x$ | `math.ceil(4.2)` $\to$ `5.0` |
| **`math.floor(x)`** | `math.floor(double) -> double` | Largest integer $\le x$ | `math.floor(4.8)` $\to$ `4.0` |
| **`math.round(x)`** | `math.round(double) -> double` | Rounds to nearest integer | `math.round(4.5)` $\to$ `5.0` |
| **`math.abs(x)`** | `math.abs(double/int) -> double/int` | Absolute value | `math.abs(-25.4)` $\to$ `25.4` |
| **`math.sign(x)`** | `math.sign(double/int) -> int` | Returns `-1`, `0`, or `1` | `math.sign(-50)` $\to$ `-1` |
| **`math.trunc(x)`** | `math.trunc(double) -> double` | Truncates decimal portion | `math.trunc(4.9)` $\to$ `4.0` |

---

### 📋 6.4 `ext.Lists()` & `ext.Sets()` — Collection Operations
Provides advanced collection transformations and set relations:

| Function / Method | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| **`list.distinct()`** | `list<T>.distinct() -> list<T>` | Removes duplicates from list | `[1, 2, 2, 3].distinct()` $\to$ `[1, 2, 3]` |
| **`list.sort()`** | `list<T>.sort() -> list<T>` | Sorts elements in ascending order | `[3, 1, 2].sort()` $\to$ `[1, 2, 3]` |
| **`list.reverse()`** | `list<T>.reverse() -> list<T>` | Reverses element order | `[1, 2, 3].reverse()` $\to$ `[3, 2, 1]` |
| **`list.flatten()`** | `list<list<T>>.flatten() -> list<T>` | Flattens nested list | `[[1, 2], [3]].flatten()` $\to$ `[1, 2, 3]` |
| **`list.slice(s, e)`** | `list<T>.slice(int, int) -> list<T>` | Returns sublist from `s` to `e` | `[10, 20, 30].slice(1, 3)` $\to$ `[20, 30]` |
| **`lists.range(stop)`** | `lists.range(int) -> list<int>` | Generates range `[0, stop)` | `lists.range(4)` $\to$ `[0, 1, 2, 3]` |
| **`sets.contains(a, b)`** | `sets.contains(list, list) -> bool` | Checks if set $A \supseteq B$ | `sets.contains([1, 2, 3], [1, 2])` $\to$ `true` |
| **`sets.equivalent(a, b)`** | `sets.equivalent(list, list) -> bool` | Checks if set $A \equiv B$ (order/dups ignored) | `sets.equivalent([1, 2, 2], [2, 1])` $\to$ `true` |
| **`sets.intersects(a, b)`** | `sets.intersects(list, list) -> bool` | Checks if $A \cap B \neq \emptyset$ | `sets.intersects([1, 2], [2, 3])` $\to$ `true` |

---

### 📦 6.5 `ext.Bindings()` (`cel.bind`) — Scoped Local Variables
The `cel.bind(var, expr, body)` macro creates an in-expression variable binding to eliminate redundant sub-expressions and accelerate execution:

```javascript
// Example: Extract and compute multiple fields from a semantic URI without re-splitting:
cel.bind(uri, string(db.rows[0].fields.media_uri),
  uri.contains('://') ? uri.split('://')[1].split('/')[0] : 'raw'
)
```

---

### 🔁 6.6 `ext.TwoVarComprehensions()` — Two-Variable Comprehensions
Enables 2-variable comprehension macros over lists (index, value) and maps (key, value):

* **`list.all(i, v, predicate)`**: `[10, 20, 30].all(i, v, v > i)` $\to$ `true`
* **`list.exists(i, v, predicate)`**: `['a', 'b'].exists(i, v, i == 1 && v == 'b')` $\to$ `true`
* **`map.all(k, v, predicate)`**: `{'a': 1, 'b': 2}.all(k, v, v > 0)` $\to$ `true`
* **`map.exists(k, v, predicate)`**: `{'admin': true}.exists(k, v, k == 'admin' && v == true)` $\to$ `true`

---

### 🔍 6.7 `ext.Regex()` — Regular Expressions
Provides regex extraction and replacement functions:

| Function | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| **`regex.replace(tgt, pat, rep)`** | `regex.replace(string, string, string) -> string` | Replaces regex matches | `regex.replace('id:123', 'id:(\\d+)', 'val:$1')` |
| **`regex.extractAll(tgt, pat)`** | `regex.extractAll(string, string) -> list<string>` | Returns all matching strings | `regex.extractAll('a1 b2 c3', '[a-z]\\d')` $\to$ `['a1', 'b2', 'c3']` |

---

## 7. 📡 `resp.*` — Response Context (REST & Semantic AI Reactors)

Available in the response mapping stage of REST Call and Semantic AI Reactors:

* **`resp`** $\rightarrow$ `map[string]any`: Decoded JSON response payload from the LLM or HTTP server.
* In REST Reactor: `resp.status_code`, `resp.headers`, `resp.body`.
* In Semantic AI Reactor: `resp.vessel_id`, `resp.optimal_lat`, `resp.biomass_probability_score`, etc.

---

## 8. 💡 Real-World CEL Expression Recipes

### Semantic MinIO URI Deconstruction (Media Storage Pruning):
```javascript
// Extract Bucket name from semantic URI:
string(db.rows[0].fields.media_uri).contains('://') ? 
  string(db.rows[0].fields.media_uri).split('://')[1].split('/')[0] : 'raw'

// Extract Object Key from semantic URI:
cel.bind(path, string(db.rows[0].fields.media_uri).split('://')[1],
  path.substring(path.indexOf('/') + 1)
)
```

### Semantic AI Reactor Output Mapping with Clamping & Fallbacks:
```javascript
// Mapping template expression in Semantic AI Reactor:
ai.clamp(ai.default(double(resp.biomass_probability_score), 0.0), 0.0, 1.0)
```

### Dynamic Multi-Tenant Request Routing:
```javascript
// URL expression in REST Reactor using mesh locagram:
'https://marine-api.ocean.org/v1/sensors' + mesh.self.locagram + '/weather'
```

### Producer & Hierarchy Guard:
```javascript
// Trigger condition ensuring event comes from direct child:
mesh.is_parent('root') && config.server.producer_id == head['$producer']
```

### Dynamic Auth Header using Node Configuration:
```javascript
// Header expression in REST Reactor:
'Bearer ' + config.server.producer_id + '-' + mesh.self.id
```

### Schema-Safe Conditional Mapping:
```javascript
// Response mapping fallback using schema check:
schema.is_registered('RawMarineData') ? double(resp.body.chlorophyll) : 0.0
```

---

## 9. 📚 Official Google CEL Specifications & References

* 📖 **[Google CEL Language Definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md)**: Core language syntax, type system, operators, and macros.
* 📦 **[Google CEL Standard Extensions Specification](https://github.com/google/cel-spec/tree/master/doc/extensions)**: Canonical specification of all extension libraries.
  * 📝 [Strings Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/strings.md)
  * 🔐 [Encoders Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/encoders.md)
  * ➗ [Math Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/math.md)
  * 📋 [Lists Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/lists.md)
  * 🗂️ [Sets Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/sets.md)
  * 🏷️ [Bindings (`cel.bind`) Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/bindings.md)
  * 🔁 [Two-Variable Comprehensions Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/comprehensions-v2.md)
  * 🔍 [Regex Extension Spec](https://github.com/google/cel-spec/blob/master/doc/extensions/regex.md)
* ⚙️ **[CEL Go Engine Repository (`github.com/google/cel-go`)](https://github.com/google/cel-go)**: Official Go implementation powering Bindu.

