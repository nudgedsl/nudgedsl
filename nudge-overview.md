# Nudge — What We're Building and Why It Matters

> *Internal document. Technical audience. No softening.*

---

## The Short Version

We're building a **protocol** that makes AI agents cheaper, faster, and more reliable — by replacing verbose prose and JSON with a dense, human-readable instruction format called nudgeDSL.

It ships as:
- A published open specification
- A static web tool anyone can use in a browser
- A Go reference implementation
- A methodology upgrade for the Nudge Framework

---

## The Problem We're Solving

When you build software with AI agents, you face three compounding costs:

**Token burn.** Every word the AI outputs costs money and latency. A typical agent action expressed in JSON burns 60–100 tokens encoding brackets, quotes, key names, and indentation. The actual intent — "move this unit to hex 3,7" — is 6 tokens. The rest is structural noise you're paying Anthropic for.

**Context degradation.** LLMs reason better with tight, relevant context. Every unnecessary token in a session window is a token that displaced something useful. Verbose orchestration artifacts (handover notes, shard documents, status updates) accumulate across a session and erode the quality of every subsequent completion. This is not a theory — it's measurable and reproducible.

**Human audit cost.** When an AI agent produces a decision log, a handover, or a status update, a human has to read and parse it. Minified JSON is unreadable. Free prose is ambiguous. There's no current format that is simultaneously dense for machines and clear for humans.

nudgeDSL solves all three at once.

---

## Lexical Foundation

Before anything else, here are the terms we use precisely.

**DSL — Domain Specific Language**
A programming language designed for a specific problem domain rather than general-purpose use. SQL is a DSL for querying databases. CSS is a DSL for styling documents. nudgeDSL is a DSL for encoding executable intent from LLMs to any backend. It is not Turing-complete. It is not a scripting language. It is a structured instruction format with a grammar strict enough to be parsed deterministically.

**Token**
The atomic unit of LLM input and output. Roughly 3–4 characters of English text per token. Every API call charges per token consumed and produced. Token count determines cost, latency, and — critically — reasoning quality. A model given 1000 tokens of relevant context reasons better than the same model given 4000 tokens of mixed signal.

**AST — Abstract Syntax Tree**
The structured representation of a parsed expression. When the nudgeDSL parser reads `MARK("task-1", "done") >> NOTIFY("ops")`, it produces a tree structure with nodes for each atom, operator, and argument. Executors walk this tree — they never touch the raw string. This is the universal intermediate format that makes one DSL string executable by Go, Python, and Dart backends without writing three parsers.

**Atom**
A short-code (1–3 uppercase characters) registered by a developer and mapped to a real backend function. `MARK` maps to `UpdateStatus`. `MOV` maps to `MoveUnit`. `SHARD` maps to `OpenShard`. Atoms are the vocabulary. The grammar is fixed. The combination is where expressiveness comes from.

**Registry**
A JSON file that defines the atoms for a specific domain. `core.json` ships with the library — generic atoms for any project. `nudge-framework.json` defines the Nudge pipeline atoms. `game-ai.json` defines tactical game atoms. You bring your own registry for your project. The parser validates against it. The prompt generator reads from it. The GBNF compiler uses it. One file, three consumers.

**GBNF — Grammar Backus-Naur Form**
The grammar format used by `llama.cpp` for constrained decoding. When you run a local LLM and want it to output only valid nudgeDSL, you compile your registry into a GBNF file and pass it to the inference engine. The model is then mathematically constrained to only produce tokens that match the grammar. Not "encouraged" — constrained. Zero invalid output. This is the edge tier story for Jetson and Apple Silicon.

**Constrained Decoding**
The technique of mathematically restricting LLM token sampling to a defined grammar at inference time. The model's probability distribution is masked so invalid tokens have zero probability. Result: 100% syntactically valid output from any model, regardless of size. A 3B parameter model with constrained decoding produces better-structured output than a 70B model generating freely. This is the core technical argument for nudgeDSL on resource-constrained hardware.

**Executor**
The component that walks a nudgeDSL AST and calls the real backend functions. The parser produces structure. The validator checks constraints. The executor runs the code. These three are deliberately separated — you can validate without executing, execute without the original string, and swap any executor without changing the DSL or the parser.

**Semantic Validation**
The check that happens after parsing. The parser guarantees syntactic correctness — valid grammar. The validator guarantees semantic correctness — right number of args, right types, values within declared ranges, enum values within allowed set. `MOV(999, 999)` on a 10×10 grid parses successfully but fails semantic validation. This separation is non-negotiable: it's the difference between "the grammar is correct" and "this will actually work."

---

## The Architecture in One Picture

```
Human intent
     │
     ▼
Nudge Framework pipeline
(Blueprint → Shard → Execute → Review)
     │
     ▼
nudgeDSL string
MARK("slice-7","done") >> MOD("handler.go") // TEST("TestHandler")
     │
     ▼
nudgeDSL parser  ──── ParseError on syntax failure
     │
     ▼
JSON AST
     │
     ▼
Semantic validator  ──── ValidationError on constraint failure
     │
     ▼
Executor
(Go / Python / Dart — reads same AST)
     │
     ▼
Real backend functions run
```

Every layer has structured errors. Nothing panics. Nothing fails silently. The human can read any nudgeDSL string without tooling and understand what happened.

---

## What the Numbers Actually Say

These are from real production code (Mandat game, director agent pipeline), not invented benchmarks.

| Scenario | JSON tokens | nudgeDSL tokens | Gross saving |
|----------|-------------|-----------------|--------------|
| Simple agent response | 54 | 21 | 61% |
| Response with modifiers + flag | 78 | 47 | 40% |
| Full narrative with 3 modifiers | 110 | 85 | 23% |

The system prompt that teaches an agent nudgeDSL costs ~105 tokens once per session. Break-even is at **N=5 calls**. A single Mandat game turn involves 7 agent calls. You're past break-even on turn one.

The bigger number — the one that's harder to measure but more important — is the context quality improvement across a long session. Replacing 400-token prose handovers with 40-token nudgeDSL handovers means 360 more tokens available for the actual task on every subsequent turn.

---

## The Nudge Framework Connection

The Nudge Framework is a human-in-the-loop AI development methodology. Five stages, five AI role profiles, decreasing autonomy as you get closer to production code. It currently produces all its artifacts as prose documents — blueprints, task lists, shards, handovers, validation reports.

nudgeDSL is the v2 format for those documents.

The handover format in the playbook already approximates DSL — it's telegraphic YAML with locked decisions and file lists. The gap is that it's still prose: a human has to read it and re-summarize it for the next session. In nudgeDSL, the handover is directly executable:

```
MARK("crisis-engine-v1", "done")
  >> MOD("crisis_engine.go") // MOD("crisis_test.go")
  >> CREATE("crisis_types.go")
  >> NOTE("mutex not channel")
  >> NOTE("crisis IDs are UUIDs")
  >> NEXT("api-layer-v1", needs="contracts.yaml")
```

This is not a replacement for the framework. It is the framework with a grammar.

---

## Three User Levels

The project is deliberately tiered. Different people enter at different levels of formalization.

**Level 1 — framework.html**
Copy prompts, paste into any AI, follow the rules. No tooling. No DSL. Just the methodology. This is how most people start. The HTML tool we already built is the entry point.

**Level 2 — nudge.dev playground**
Import your registry, validate DSL, generate agent prompts, translate existing output. Uses your own Anthropic API key, runs entirely in the browser. No backend. No account. This is where the DSL becomes real for someone.

**Level 3 — CLI + local integration**
`nudge analyze`, batch conversion, Pi/MCP integration, constrained decoding on Jetson. The full edge stack. This is the power user layer — the one where the compounding benefits of token savings across thousands of calls become material.

Nobody is locked out of the core value. The protocol is free. The grammar is open. The tooling is tiered.

---

## What Makes This Different

Every other approach to this problem optimizes one axis:

- **Minified JSON** — machine efficient, human-unreadable, brittle
- **Structured outputs / function calling** — works for cloud models, no edge story, model-specific, no portability
- **Natural language instructions** — flexible, wildly token-inefficient, unreliable on small models
- **Fine-tuning** — powerful but requires data, compute, and re-runs every time the task changes

nudgeDSL optimizes three axes simultaneously: token density, human readability, and cross-platform portability. The grammar is the same whether you're running Claude 3.5 Sonnet on a cloud API or llama3.2:3b on a Jetson AGX Orin with constrained decoding. The executor doesn't care which model produced the string — it just walks the AST.

The portability argument is the one that compounds over time. Every atom you define works everywhere. Every registry you publish works for any model, any backend, any tier.

---

## Current State

| Component | Status |
|-----------|--------|
| Specification v0.1.0 | ✅ Complete — `nudgeDSL-spec-v0.1.md` |
| Fuzzing corpus (55 cases) | ✅ Complete — `nudgeDSL-fuzz-corpus-v0.1.json` |
| Go parser + validator | ✅ Complete — `nudgedsl/` package |
| Go test suite | ✅ Complete — passes all 55 corpus cases |
| Core registry | ✅ Complete — 14 atoms |
| Nudge Framework registry | ✅ Complete — 27 atoms |
| Game AI registry | ✅ Complete — 14 atoms |
| Framework alignment doc | ✅ Complete |
| Static web site | ✅ Complete — 4 pages, ready for GitHub Pages |
| JS parser (browser) | ✅ Complete — matches Go implementation |
| Prose → DSL translator | ✅ Complete — Anthropic API, BYOK |
| Registry browser + validator | ✅ Complete |
| Spec rendered as HTML | ✅ Complete |
| GBNF generator | 🔲 Phase 3 |
| Auto-prompt generator (Go) | 🔲 Phase 3 |
| CLI (`nudge analyze`) | 🔲 Phase 4 |
| Jetson constrained decoding | 🔲 Phase 4 |

---

## The Honest Risk

The technical risk is low. The parser works. The spec is solid. The Go implementation passes the full fuzzing corpus.

The real risk is discovery. A developer who would benefit from nudgeDSL will never search for "nudgeDSL" — they'll search for "reduce LLM token usage" or "LLM output format" or "AI agent output structure." The protocol has to be findable before it can be used.

That's a content and community problem, not a technical one. The answer is: ship the site, write the blog posts, post the real benchmark numbers on Hacker News with the actual before/after code. Let the numbers do the talking. They're good enough.

---

*nudge-overview.md — internal. Last updated with spec v0.1.0.*
