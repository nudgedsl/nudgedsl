# nudgeDSL Specification
**Version:** 0.1.0  
**Status:** Draft — Pre-implementation  
**Purpose:** Agent context injection, grammar source of truth, contributor reference

---

## 1. What nudgeDSL Is

nudgeDSL is a token-dense, human-readable Domain Specific Language for encoding executable intent from LLMs to multi-language backends (Go, Python, Dart).

An agent outputs a nudgeDSL string. A universal parser converts it to a JSON AST. Language-native executors read the AST and run the corresponding functions.

nudgeDSL does **not**:
- Execute code directly
- Make network calls
- Carry state between turns

---

## 2. Core Concepts

### Atom
An **Atom** is a short-code (1–3 characters) registered by the developer and mapped to a backend function.

```
M       → MoveUnit
A       → SelectAgent
S       → SetOverwatch
HEAL    → HealUnit
```

Atoms are **always uppercase**. The atom registry is injected into the agent context at runtime (see Section 7).

### REGISTRY — the special declaration atom

`REGISTRY` is a built-in atom that declares which domain registry a nudgeDSL document expects. It must appear as the first expression if present.

```
REGISTRY("nudge-framework", version="0.1")
>> SHARD("A1") >> ACCEPT("zero stubs")
```

If the loaded registry does not match the declared domain and version, the parser returns a `RegistryError` before evaluating any other atom. This enforces the contract between a document and the tool that executes it.

### Operator
An **Operator** encodes the relationship between atoms. Operators are fixed — they are part of the nudgeDSL grammar and cannot be redefined by developers.

### Argument
Arguments are passed to atoms inside parentheses. Supported types:

| Type | Syntax | Example |
|------|--------|---------|
| String | `"value"` | `A("sniper")` |

> **v0.1 constraint:** Strings cannot contain escaped double quotes. `MSG("He said \"hello\"")` is invalid. Design atom arguments to avoid this need — use single-word identifiers or enum values where possible.
| Integer | bare number | `M(3, 7)` |
| Float | decimal number | `S(0.75)` |
| Boolean | `true` / `false` | `LOCK(true)` |
| Null | `null` | `RESET(null)` |

Multiple arguments are comma-separated: `M(3, 7)`.  
No arguments: `RESET()`.

---

## 3. Operator Table

| Operator | Name | Semantics | Failure Behavior |
|----------|------|-----------|-----------------|
| `>>` | Chain | Execute left, then right sequentially. Right does not run if left fails. | Fail-fast (default) |
| `\|` | Fallback | Try left. If left fails, try right. | Best-effort per branch |
| `//` | Parallel | Execute left and right concurrently. | Configurable (see Section 5) |
| `**N` | Amplify | Repeat the preceding atom N times. | Fail-fast |

### Precedence (high to low)
1. `**` (Amplify) — binds tightest, applies to the immediately preceding atom
2. `//` (Parallel)
3. `>>` (Chain)
4. `|` (Fallback) — binds loosest

Use parentheses to override precedence.

### Examples

```
A("sniper") >> M(3, 7)
```
Select sniper, then move to hex (3,7).

```
HEAL("self") | RUN("south")
```
Try to heal. If healing fails, run south.

```
ALERT("cops") // LOCK("doors")
```
Alert cops while simultaneously locking doors.

```
SHOOT() ** 3
```
Shoot three times sequentially.

```
A("tank") >> (M(2,2) // SHIELD())
```
Select tank, then simultaneously move and activate shield.

---

## 4. EBNF Grammar

```ebnf
program     ::= expression EOF
expression  ::= fallback
fallback    ::= parallel ( "|" parallel )*
parallel    ::= chain ( "//" chain )*
chain       ::= amplify ( ">>" amplify )*
amplify     ::= primary ( "**" INTEGER )?
primary     ::= atom_call | "(" expression ")"
atom_call   ::= ATOM "(" arg_list? ")"
arg_list    ::= arg ( "," arg )*
arg         ::= STRING | INTEGER | FLOAT | BOOLEAN | NULL
ATOM        ::= [A-Z][A-Z0-9]{0,2}
STRING      ::= '"' [^"]* '"'
INTEGER     ::= "-"? [0-9]+
FLOAT       ::= "-"? [0-9]+ "." [0-9]+
BOOLEAN     ::= "true" | "false"
NULL        ::= "null"
EOF         ::= end of input
```

---

## 5. Parallel Failure Modes

The `//` operator supports three configurable failure modes. The developer sets this at executor registration. Default is `fail-fast`.

### fail-fast
If any branch fails, all remaining branches are aborted immediately. No partial state is applied.

```
ALERT("cops") // LOCK("doors") // SEAL("exits")
```
If `LOCK` fails, `SEAL` is aborted. `ALERT` result depends on execution order.

### best-effort
All branches execute regardless of individual failures. Errors are collected and returned as an array after all branches complete. Partial state is applied.

### compensating
Each atom may declare a rollback atom in the registry. If a branch fails, the executor calls the rollback atom of each already-completed branch. Requires rollback declaration at atom registration (see Section 7).

The executor automatically passes the **exact same arguments** from the forward atom to its rollback atom. No additional argument mapping is required or supported in v0.1.

```
HEAL("ally1", 0.5) // LOCK("doors")
```
If `LOCK` fails, the executor calls `UNHEAL("ally1", 0.5)` — same args, no remapping.

Implication: rollback atoms must accept the same argument signature as their forward atom. Enforce this at registry validation time, not at execution time.

---

## 6. JSON AST Format

The parser always outputs this structure. Executors consume this — never the raw string.

```json
{
  "version": "0.1.0",
  "root": {
    "type": "chain",
    "nodes": [
      {
        "type": "call",
        "atom": "A",
        "fn": "SelectAgent",
        "args": ["sniper"]
      },
      {
        "type": "call",
        "atom": "M",
        "fn": "MoveUnit",
        "args": [3, 7]
      }
    ]
  }
}
```

### Node types

| `type` | Fields |
|--------|--------|
| `call` | `atom`, `fn`, `args[]` |
| `chain` | `nodes[]` |
| `parallel` | `nodes[]`, `failure_mode` |
| `fallback` | `nodes[]` |
| `amplify` | `node`, `count` |

---

## 7. Atom Registry Format

Developers register atoms before injecting context into the agent. The registry is the single source of truth for prompt generation and GBNF grammar compilation.

### Registration schema (JSON)

```json
{
  "atoms": [
    {
      "atom": "M",
      "fn": "MoveUnit",
      "description": "Move the selected unit to hex coordinates (x, y).",
      "args": [
        { "name": "x", "type": "integer", "min": 0, "max": 63 },
        { "name": "y", "type": "integer", "min": 0, "max": 63 }
      ],
      "rollback": null
    },
    {
      "atom": "HEAL",
      "fn": "HealUnit",
      "description": "Heal the target unit by a percentage of max HP.",
      "args": [
        { "name": "target", "type": "string", "enum": ["self", "ally1", "ally2"] },
        { "name": "amount", "type": "float", "min": 0.0, "max": 1.0 }
      ],
      "rollback": "UNHEAL"
    },
    {
      "atom": "MARK",
      "fn": "UpdateStatus",
      "description": "Transition an item to a new status.",
      "args": [
        { "name": "id", "type": "string" },
        { "name": "status", "type": "string", "enum": ["pending", "done", "skipped", "error"] }
      ],
      "rollback": null
    }
  ]
}
```

### Arg constraint fields

| Field | Applies to | Meaning |
|-------|-----------|---------|
| `min` / `max` | integer, float | Inclusive range |
| `enum` | string | Allowed values |
| `required` | all | Default true |

Constraints are enforced by the **semantic validator** (Section 8), not the parser.

### Rollback signature validation (registry load time)

If an atom declares a `rollback`, the following checks run when the registry is loaded — not at execution time:

1. The rollback atom must exist in the same registry.
2. The rollback atom must declare the same number of arguments as the forward atom.
3. Each argument must have the same `type` at the same position.

If any check fails, registry load is aborted with a structured `RegistryError`:

```json
{
  "type": "RegistryError",
  "code": "ROLLBACK_SIGNATURE_MISMATCH",
  "atom": "HEAL",
  "rollback": "UNHEAL",
  "detail": "UNHEAL declares 1 arg(s), expected 2 to match HEAL"
}
```

| Code | Meaning |
|------|---------|
| `ROLLBACK_NOT_FOUND` | Rollback atom declared but not present in registry |
| `ROLLBACK_SIGNATURE_MISMATCH` | Arg count or type mismatch between forward and rollback atom |

Fail at load time, not at execution time. A misconfigured rollback that surfaces during a compensating transaction is worse than a startup error.

---

## 8. Validation Pipeline

Parsing and validation are two separate steps. A string can be syntactically valid and semantically invalid.

```
raw string
    │
    ▼
[ Parser ]  ──── syntax error ──▶ ParseError
    │
    ▼
  JSON AST
    │
    ▼
[ Semantic Validator ]  ──── constraint violation ──▶ ValidationError
    │
    ▼
[ Executor ]
```

### Validator interface (Go)

```go
type Validator interface {
    Validate(ast *AST, registry *AtomRegistry) []ValidationError
}
```

nudgeDSL ships a `DefaultValidator` that enforces all registry-declared constraints. Developers may replace or extend it.

---

## 9. Error Taxonomy

All errors are structured. The parser and validator never panic.

### ParseError

```json
{
  "type": "ParseError",
  "code": "UNEXPECTED_TOKEN",
  "position": 14,
  "expected": ")",
  "got": ">>",
  "input": "M(3, 7 >> S()"
}
```

| Code | Meaning |
|------|---------|
| `UNEXPECTED_TOKEN` | Got a token that doesn't fit the grammar at this position |
| `UNTERMINATED_STRING` | String opened with `"` but never closed |
| `MISSING_CLOSE_PAREN` | Atom call opened but `)` never found |
| `TRAILING_OPERATOR` | Expression ends with an operator (`>>`, `//`, `\|`) |
| `UNKNOWN_ATOM` | Atom not found in registry |
| `EMPTY_INPUT` | Input string is empty or whitespace only |
| `TRUNCATED_INPUT` | Input ends mid-token (likely stream truncation) |

### ValidationError

```json
{
  "type": "ValidationError",
  "code": "ARG_OUT_OF_RANGE",
  "atom": "M",
  "arg": "x",
  "value": 999,
  "constraint": "max: 63"
}
```

| Code | Meaning |
|------|---------|
| `ARG_OUT_OF_RANGE` | Integer or float outside declared min/max |
| `ARG_NOT_IN_ENUM` | String not in declared enum values |
| `ARG_TYPE_MISMATCH` | Argument type doesn't match declaration |
| `ARG_COUNT_MISMATCH` | Wrong number of arguments for this atom |
| `UNKNOWN_ATOM` | Atom present in AST but absent from registry at validation time |

---

### RegistryError

Produced at registry load time, before any parsing or execution occurs.

```json
{
  "type": "RegistryError",
  "code": "ROLLBACK_NOT_FOUND",
  "atom": "HEAL",
  "rollback": "UNHEAL"
}
```

| Code | Meaning |
|------|---------|
| `ROLLBACK_NOT_FOUND` | Rollback atom declared but not present in registry |
| `ROLLBACK_SIGNATURE_MISMATCH` | Arg count or type mismatch between forward and rollback atom |
| `REGISTRY_MISMATCH` | Document `REGISTRY()` declaration does not match loaded registry domain or version |

nudgeDSL generates the system prompt injection from the atom registry. Agents should not be given this spec — they receive the generated prompt only.

### Generated prompt format

```
You are operating with nudgeDSL v0.1.0.
Output ONLY valid nudgeDSL strings. No explanation, no preamble, no markdown.

## Registered Atoms
MARK(id: string, status: string)  — Transition an item to a new status.
NOTIFY(channel: string)           — Broadcast a completion event to a channel.
FETCH(source: string)             — Retrieve data from the named source.

## Operators
>>   chain (sequential)
|    fallback (try left, then right)
//   parallel (concurrent)
**N  amplify (repeat N times)

## Examples
MARK("task-1", "done")
FETCH("primary") | FETCH("replica")
MARK("job-7", "running") >> FETCH("data") >> MARK("job-7", "done")

## Constraints
MARK.status: one of [pending, done, skipped, error]

Output nudgeDSL only.
```

---

## 11. Worked Examples (Increasing Complexity)

### 1. Single call
```
MARK("task-42", "done")
```
Transition item `task-42` to done status.

### 2. Chained sequence
```
CREATE("output/report.pdf") >> VERIFY("output/report.pdf") >> NOTIFY("complete")
```
Create a file, verify it exists, then broadcast completion. Each step waits for the previous.

### 3. Fallback
```
FETCH("primary-db") | FETCH("replica-db")
```
Try primary source. If it fails, fall back to replica.

### 4. Parallel with failure mode in context
```
ALERT("ops-team") // LOCK("write-access")
```
Simultaneously alert the ops team and lock write access. Failure mode is set by executor config.

### 5. Amplify
```
PING("health-check") ** 3
```
Run the health check three times sequentially.

### 6. Compound
```
MARK("job-7", "running") >> (WRITE("db") // CACHE("redis")) >> MARK("job-7", "done")
```
Mark job as running. Simultaneously write to DB and cache. Then mark done.

### 7. Fallback with chain
```
(ACQUIRE("gpu-0") >> RUN("inference")) | (ACQUIRE("gpu-1") >> RUN("inference"))
```
Try to acquire gpu-0 and run. If that sequence fails entirely, fall back to gpu-1.

### 8. Amplify binds before Parallel (precedence clarification)
```
A() // B() ** 3
```
Because `**` binds tighter than `//`, this is parsed as:

```
A() // (B() ** 3)
```

AST: run `A()` in parallel with (run `B()` sequentially 3 times). `**` never applies across both parallel branches. Use explicit parentheses if the intent is different.

## 12. Simulator / Translator

The nudgeDSL simulator takes existing agent output (JSON, Python function calls) and produces:

1. The equivalent nudgeDSL string
2. Token count comparison (input format vs nudgeDSL output)
3. Net savings after system prompt amortization
4. The atom registry it inferred
5. The GBNF grammar it would generate

### CLI usage
```bash
nudge analyze ./agent_output.json --format json
nudge analyze ./agent_output.py --format python
nudge analyze --stdin

nudge analyze ./output.json --registry ./atoms.json
nudge analyze ./output.json --registry core.json --registry nudge.json --registry project/atoms.json
```

Multiple `--registry` flags are additive. Last-defined atom wins on name conflicts. The translator validates the combined set for signature conflicts at load time and errors before parsing any input.

### Output format
```json
{
  "version": "0.1.0",
  "input_tokens": 847,
  "nudgeDSL_output_tokens": 12,
  "system_prompt_tokens": 143,
  "net_tokens": 155,
  "savings_percent": 81.7,
  "nudgeDSL": "MARK(\"task-42\", \"done\") >> NOTIFY(\"ops\")",
  "inferred_registry": { "atoms": [ ... ] },
  "warnings": [
    "Atom PING has no declared constraints — semantic validation will be skipped."
  ]
}
```

**Note:** `savings_percent` is always net (output + amortized prompt), never gross. Gross savings are not reported.

---

## 13. Versioning

This document is versioned. Breaking changes increment the minor version. The `version` field in every AST output references this spec version.

Agents and executors must agree on spec version. A mismatch is a runtime error, not a silent failure.

---

## 15. Registry System

nudgeDSL separates the grammar (this spec) from the vocabulary (atom registries). The grammar never changes within a major version. The vocabulary is fully customizable per project.

### Three registry layers

```
core.json          — generic atoms usable in any domain
                     (MARK, CREATE, NOTIFY, FETCH, PING...)

domain.json        — atoms for a specific methodology or framework
                     (SHARD, PHASE, ACCEPT, GATE, PROFILE...)

project/atoms.json — atoms for a specific project
                     (AGENT, CRISIS, RENDER, QUEUE...)
```

Each layer is a flat JSON file following the atom registration schema from Section 7. There is no inheritance between layers. If you want core atoms plus your own, you include both files explicitly.

### Registry file format

```json
{
  "domain": "nudge-framework",
  "version": "0.1",
  "extends": [],
  "atoms": [
    {
      "atom": "SHARD",
      "fn": "OpenShard",
      "description": "Open a new shard context for a named task.",
      "args": [
        { "name": "id", "type": "string" },
        { "name": "phase", "type": "string" }
      ],
      "rollback": null
    }
  ]
}
```

| Field | Required | Meaning |
|-------|----------|---------|
| `domain` | yes | Unique identifier for this registry |
| `version` | yes | Semver string — must match `REGISTRY()` declaration in documents |
| `extends` | no | Reserved for v0.2. Must be empty array in v0.1. |
| `atoms` | yes | Array of `AtomDef` objects (see Section 7) |

### Loading multiple registries

Registries are loaded in order. Last-defined atom wins on name conflicts.

```bash
nudge analyze ./output.json \
  --registry core.json \
  --registry nudge.json \
  --registry project/atoms.json
```

At load time, the combined registry is validated for:
- Rollback signature integrity (Section 7)
- `REGISTRY()` declaration match if present in the document
- No two atoms with the same code and conflicting argument signatures

A conflict at load time is a `RegistryError` — the tool exits before parsing any input.

### Composition rules

- **No inheritance.** A project registry does not extend or modify a parent registry. Include both files explicitly.
- **No override keys.** There is no `"override": true` flag. Last-defined wins, silently.
- **No merge logic in v0.1.** If two registries define `MARK` with different arg counts, the second definition replaces the first. The load-time conflict check catches signature mismatches and errors loudly.
- **`REGISTRY()` is optional.** If omitted, no domain/version check is performed. Recommended for all shared or published documents.

### Recommended file naming

| File | Purpose |
|------|---------|
| `nudgedsl-core.json` | Generic atoms shipped with the library |
| `nudgedsl-{domain}.json` | Framework or methodology atoms |
| `atoms.json` | Project-local atoms (lives next to shards) |

### The `REGISTRY` atom in documents

A nudgeDSL document can declare its expected registry at the top:

```
REGISTRY("nudge-framework", version="0.1")
>> SHARD("A1", phase="A")
>> ACCEPT("zero NotImplementedError")
```

If the loaded registry does not match `domain` and `version`, the parser returns:

```json
{
  "type": "RegistryError",
  "code": "REGISTRY_MISMATCH",
  "expected": { "domain": "nudge-framework", "version": "0.1" },
  "got": { "domain": "my-project", "version": "0.2" }
}
```

This makes documents self-describing and prevents silent mismatches when sharing nudgeDSL artifacts across teams.

---



The following are explicitly out of scope for this version:

- Variables or state references between atoms (`$result >> M($x, $y)`)
- Conditional branching (`IF(cond) >> A() | B()`)
- Loops with dynamic counts (`SHOOT() ** $n`)
- Typed return values from atoms
- Multi-agent addressing (`@agent1 >> M(3,7)`)
- Streaming partial AST execution
- Registry inheritance (`"extends": ["core.json"]`) — the `extends` field is reserved, must be empty array in v0.1
- Runtime registry merging or override semantics beyond last-defined-wins

These are tracked for v0.2+. Do not implement them in v0.1 parsers.

---

*nudgeDSL-spec-v0.1.md — generated as pre-build artifact. Implementation does not yet exist.*
