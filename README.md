# nudgeDSL

> Token-dense, human-readable DSL for encoding executable intent from LLMs to any backend.

```
MARK("slice-7", "done")
  >> MOD("handler.go") // TEST("TestHandler")
  >> NOTE("mutex not channel")
  >> NEXT("api-layer")
```

vs the JSON equivalent that costs 4× the tokens and requires parsing to read.

---

## Why this exists

When you build with AI agents, you pay three compounding costs:

**Token burn.** A typical agent action in JSON burns 60–100 tokens on brackets, quotes, and key names. The actual intent is 15 tokens. You're paying for noise.

**Context degradation.** Every unnecessary token in a session window displaces something useful. Verbose handover notes, shard documents, and status updates accumulate across a session and erode reasoning quality on every subsequent completion.

**Human audit cost.** Minified JSON is unreadable. Free prose is ambiguous. There is no current format that is simultaneously dense for machines and clear for humans at 2am.

nudgeDSL solves all three at once.

---

## Real numbers

From production code — a director agent pipeline, not a constructed benchmark:

| Scenario | JSON | nudgeDSL | Gross saving |
|----------|------|----------|--------------|
| Simple agent response | 54 tok | 21 tok | **61%** |
| Response with modifiers + flag | 78 tok | 47 tok | **40%** |
| Full response with 3 modifiers | 110 tok | 85 tok | **23%** |

System prompt cost to teach an agent nudgeDSL: ~105 tokens, once per session.
Break-even: **N = 5 calls**. A single game turn involves 7 agent calls.

`savings_percent` is always reported net — output tokens plus amortized system prompt. Gross savings are not reported. They are misleading.

---

## Try it in 30 seconds

No install. No account.

**[nudgedsl.dev](https://nudgedsl.dev)** — paste nudgeDSL, validate syntax, generate agent prompts, import custom registries. Bring your own Anthropic key to translate existing agent output.

---

## How it works

nudgeDSL decouples generation from execution.

```
LLM outputs a string
        │
        ▼
[ Parser ]  ──── ParseError on syntax failure
        │
        ▼
   JSON AST
        │
        ▼
[ Validator ]  ──── ValidationError on constraint failure
        │
        ▼
[ Executor ]  ──── Go / Python / Dart — reads same AST
```

### Four operators

| Operator | Name | Semantics |
|----------|------|-----------|
| `>>` | Chain | Sequential — right runs only if left succeeds |
| `\|` | Fallback | Try left, use right if left fails |
| `//` | Parallel | Concurrent execution |
| `**N` | Amplify | Repeat N times sequentially |

### Precedence (high → low)

`**` → `//` → `>>` → `|`

Use parentheses to override. `A() // B() ** 3` parses as `A() // (B() ** 3)`.

### Examples

```
# Single call
MARK("task-1", "done")

# Chain
CREATE("report.pdf") >> NOTIFY("ops")

# Fallback
FETCH("primary-db") | FETCH("replica-db")

# Parallel
WRITE("db") // CACHE("redis")

# Compound
MARK("job-7", "running")
  >> (WRITE("db") // CACHE("redis"))
  >> MARK("job-7", "done")

# Nudge Framework — shard handover
REGISTRY("nudge-framework", version="0.1.0")
  >> MARK("slice-7", "done")
  >> MOD("handler.go") // TEST("TestHandler")
  >> NOTE("mutex not channel")
  >> NEXT("api-layer", needs="contracts.yaml")
```

---

## Registry system

The grammar is fixed. The vocabulary is yours.

An **atom** is a 1–3 character uppercase code mapped to a backend function. You define them in a JSON registry file.

```json
{
  "domain": "my-project",
  "version": "0.1.0",
  "atoms": [
    {
      "atom": "MARK",
      "fn": "UpdateStatus",
      "description": "Transition an item to a new status.",
      "args": [
        { "name": "id", "type": "string" },
        { "name": "status", "type": "string",
          "enum": ["pending", "done", "skipped", "error"] }
      ],
      "rollback": null
    }
  ]
}
```

Load multiple registries — last-defined wins on conflicts:

```bash
nudge analyze ./output.json \
  --registry registries/core.json \
  --registry registries/nudge-framework.json \
  --registry ./my-project/atoms.json
```

### Published registries

| Registry | Domain | Atoms | Description |
|----------|--------|-------|-------------|
| [`core.json`](registries/core.json) | `nudgedsl-core` | 14 | Generic atoms for any domain |
| [`nudge-framework.json`](registries/nudge-framework.json) | `nudge-framework` | 27 | Nudge Framework 5-stage pipeline |
| [`game-ai.json`](registries/game-ai.json) | `game-ai` | 14 | Real-time game AI agents |

Browse and download at [nudgedsl.dev/registries](https://nudgedsl.dev/registries).

---

## Nudge Framework

nudgeDSL is the v2 document format for the [Nudge Framework](https://nudgedsl.dev/framework) — a human-in-the-loop methodology for building software with AI agents.

The framework produces five artifact types across its pipeline. All of them are currently prose. nudgeDSL makes them machine-executable without losing human readability.

| Stage | Role | Today | nudgeDSL |
|-------|------|-------|----------|
| Blueprint | Architecture Critic | `blueprint.md` prose | `BLUEPRINT >> STACK >> RISK` |
| Task list | Dependency Analyst | ordered markdown | `TASK("a") >> TASK("b", depends="a")` |
| Shard | Specification Writer | `shard-{n}.md` prose | `SHARD >> ACCEPT >> EXCLUDE` |
| Development | Disciplined Implementer | `handover.md` prose | `MARK >> MOD // TEST >> NOTE >> NEXT` |
| Verification | QA Auditor | `validation.md` prose | `VERIFY >> RESULT >> FLAG` |

The framework is an ideology. nudgeDSL is the grammar it was always moving toward.

---

## Edge tier — constrained decoding

On resource-constrained hardware (Jetson AGX Orin, Apple Silicon), nudgeDSL compiles to a GBNF grammar for `llama.cpp` constrained decoding.

The model is mathematically restricted to only emit valid nudgeDSL tokens. Not encouraged — constrained. A 3B parameter model with constrained decoding produces more reliable structured output than a 70B model generating freely.

```bash
# compile registry to GBNF
nudge gbnf --registry ./atoms.json --output nudgedsl.gbnf

# run with llama.cpp
./llama-cli -m model.gguf --grammar-file nudgedsl.gbnf -p "your prompt"
```

---

## Access tiers

The protocol is always free. The tooling is tiered.

| | Tier 1 — Free | Tier 2 — BYOK | Tier 3 — CLI |
|--|---------------|---------------|--------------|
| Validate DSL | ✓ | ✓ | ✓ |
| Generate agent prompt | ✓ | ✓ | ✓ |
| Import custom registry | ✓ | ✓ | ✓ |
| Export GBNF grammar | ✓ | ✓ | ✓ |
| Prose → DSL translation | — | ✓ your API key | ✓ |
| Token savings benchmark | — | ✓ | ✓ |
| Batch processing | — | — | ✓ |
| Local pipeline integration | — | — | ✓ |
| Pi / MCP integration | — | — | ✓ |

Tier 2 uses your Anthropic API key directly in the browser. Your key never touches our servers — there are none.

---

## Current status

| Component | Status |
|-----------|--------|
| Specification v0.1.0 | ✅ Complete |
| EBNF grammar | ✅ Complete |
| Fuzzing corpus (55 cases) | ✅ Complete |
| Go parser + validator | ✅ Complete |
| Go test suite | ✅ Passes all 55 corpus cases |
| Core registry (14 atoms) | ✅ Complete |
| Nudge Framework registry (27 atoms) | ✅ Complete |
| Game AI registry (14 atoms) | ✅ Complete |
| Browser JS parser | ✅ Complete |
| Web playground | ✅ Complete |
| Prose → DSL translator | ✅ Complete |
| Registry browser | ✅ Complete |
| Auto-prompt generator (Go) | 🔲 Phase 3 |
| GBNF compiler (Go) | 🔲 Phase 3 |
| CLI (`nudge analyze`) | 🔲 Phase 4 |
| Jetson constrained decoding | 🔲 Phase 4 |
| Dart executor | 🔲 Phase 4 |

This is a pre-release. The spec is locked. The parser works. The web tool is live. The CLI is coming.

---

## Spec

The full specification is a single markdown file:
[`spec/nudgeDSL-spec-v0.1.md`](spec/nudgeDSL-spec-v0.1.md)

It covers: grammar (EBNF), operator table, parallel failure modes, JSON AST format, atom registry schema, validation pipeline, error taxonomy, auto-generated prompt format, worked examples, simulator/CLI, versioning, registry system.

Rendered at [nudgedsl.dev/spec](https://nudgedsl.dev/spec).

---

## Contributing

**Add a registry.** The fastest way to contribute. Define atoms for your domain, open a PR adding your file to `registries/`. One file, instant value.

**Report parser discrepancies.** The JS browser parser and Go reference implementation must agree on every input. If you find a case where they differ, open an issue with the input string and both outputs.

**Add a reference implementation.** The spec is the contract. If you implement a parser in Python, Rust, TypeScript, or any other language that passes the fuzzing corpus, open a PR adding it to `implementations/`.

**Improve the fuzzing corpus.** [`spec/nudgeDSL-fuzz-corpus-v0.1.json`](spec/nudgeDSL-fuzz-corpus-v0.1.json) — 55 malformed input cases covering 13 categories. More cases, especially natural language bleed and truncation patterns from real LLM output, are always useful.

---

## Repository structure

```
nudgedsl/
├── README.md
├── spec/
│   ├── nudgeDSL-spec-v0.1.md       ← grammar source of truth
│   └── nudgeDSL-fuzz-corpus-v0.1.json
├── registries/
│   ├── core.json                    ← generic atoms
│   ├── nudge-framework.json         ← framework pipeline atoms
│   └── game-ai.json                 ← game AI atoms
├── implementations/
│   └── go/                          ← reference implementation
│       ├── nudgedsl.go
│       ├── lexer.go
│       ├── parser.go
│       ├── validator.go
│       ├── registry.go
│       ├── ast.go
│       ├── errors.go
│       └── parser_test.go
└── web/                             ← static site (GitHub Pages)
    ├── index.html
    ├── framework.html
    ├── spec.html
    ├── registries.html
    ├── shared.css
    └── shared.js
```

---

## License

MIT. Use it, fork it, build on it. If you publish a registry or implementation, a link back is appreciated but not required.

---

*nudgeDSL v0.1.0 — pre-release · [nudgedsl.dev](https://nudgedsl.dev) · Structure beats hope.*
