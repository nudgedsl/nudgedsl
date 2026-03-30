# Nudge Framework × nudgeDSL — Alignment Document
**Version:** 0.1  
**Purpose:** Maps every Nudge Framework pipeline stage to its nudgeDSL encoding. Reference for contributors and for the v2 framework document rewrite.

---

## Core Principle

The Nudge Framework's own philosophy answers why nudgeDSL exists:

> *"Token Efficiency = Quality Efficiency. Format follows consumer: AI-consumed outputs are telegraphic. Human-consumed outputs are scannable."*

Today's framework documents are human-scannable but not AI-executable. nudgeDSL makes them both — dense enough for machines, readable enough for humans at 2am.

The code inside every deliverable stays native language (Go, Python, Dart). nudgeDSL encodes the **orchestration layer only** — what gets built, in what order, with what constraints, and what the outcome was.

---

## Pipeline Mapping

### Stage 1 — Blueprint (Architecture Critic)
**Role:** High autonomy. Adversarial. Challenges every assumption.  
**Current output:** `context.md`, `blueprint.md`, `tasklist.md`, `index.md` — all prose.

| Today (prose) | nudgeDSL equivalent |
|---|---|
| context.md — free text rules and stack | `BLUEPRINT("project-id", version="1.0")` |
| Stack listed as prose | `STACK("go", "flutter", "postgres")` |
| Risks as prose paragraphs | `RISK("no-auth-on-ws", severity=2)` |
| Constraints as "must" sentences | `CONSTRAINT("jetson-only-inference")` |

---

### Stage 2 — Task List (Dependency Analyst)
**Role:** Medium autonomy. Topological sort. Flags hidden coupling.  
**Current output:** Ordered markdown list with inline dependency annotations.

| Today (prose) | nudgeDSL equivalent |
|---|---|
| Ordered markdown list | `TASK("auth-01") >> TASK("api-01")` |
| Dependency annotations inline | `TASK("ui-01", depends="api-01")` |
| index.md updated manually | `MARK("index", "updated")` |

---

### Stage 3 — Shard / Spec (Specification Writer)
**Role:** Medium autonomy. No implementation code. Acceptance criteria.  
**Current output:** `shard-{n}.md` — deliverable, inputs/outputs, constraints, acceptance criteria.

| Today (prose) | nudgeDSL equivalent |
|---|---|
| Deliverable description | `SHARD("A1", phase="A")` |
| Files in scope as bullet list | `SCOPE("base_renderer.py")` |
| Acceptance criteria as checkboxes | `ACCEPT("zero NotImplementedError")` |
| Must-not-touch items | `EXCLUDE("gif_writer.py")` |
| Stop condition | `STOP("Phase A complete")` |

---

### Stage 4 — Development (Disciplined Implementer)
**Role:** Low autonomy. Executes against shard. Stops if ambiguous.  
**Current output:** Code deliverable + `handover.md` + `_INDEX.md` updated.

| Today (prose) | nudgeDSL equivalent |
|---|---|
| `handover.md` — telegraphic prose | `MARK("A1", "done")` |
| Files created — listed as bullets | `CREATE("migrations/v2.sql")` |
| Decisions locked — bullet points | `NOTE("used mutex not channel")` |
| Next slice needs — prose | `NEXT("A2", needs="contracts.yaml")` |
| Session continuation decision | `CONTINUE("A2")` or `RESTART("B1")` |

---

### Stage 5 — Verification (QA Auditor)
**Role:** Zero autonomy. Reviews git diff against shard. Flags out-of-scope changes.  
**Current output:** `validation.md` — READY / NEEDS REVISION / BLOCKED.

| Today (prose) | nudgeDSL equivalent |
|---|---|
| READY verdict | `VERIFY("A1") >> RESULT("ready")` |
| NEEDS REVISION verdict | `VERIFY("A1") >> RESULT("revision", reason="missing F034")` |
| BLOCKED verdict | `VERIFY("A1") >> RESULT("blocked", reason="dep-missing")` |
| Regression risk flagged | `FLAG("regression", scope="lexer")` |
| Out-of-scope changes flagged | `FLAG("out-of-scope", file="unrelated.go")` |

---

## The Handover Format

The current `.handover.md` format from the playbook is already almost nudgeDSL. This is the existing format:

```yaml
slice: crisis-engine-v1
status: complete
files_modified: [crisis_engine.go, crisis_test.go]
files_created: [crisis_types.go]
decisions_locked:
  - CrisisEngine uses mutex, not channel
  - Crisis IDs are UUIDs, not sequential
next_slice_needs: [contracts.yaml, crisis_types.go skeleton]
```

In nudgeDSL, the same information is:

```
MARK("crisis-engine-v1", "done")
>> MODIFY("crisis_engine.go") // MODIFY("crisis_test.go")
>> CREATE("crisis_types.go")
>> NOTE("mutex not channel")
>> NOTE("crisis-ids are UUIDs")
>> NEXT("next-slice", needs="contracts.yaml")
```

Both are human-readable. The nudgeDSL version is directly executable by the orchestrator — no parsing required.

---

## The Three Audiences

| Audience | What they read | How they use it |
|---|---|---|
| New to Nudge | HTML tool (nudge-framework.html) | Copy prompts, learn the methodology |
| Using Nudge regularly | nudgeDSL documents | Read and write structured pipeline artifacts |
| Building on Nudge | nudgeDSL + translator tool | Machine-execute the pipeline, automate handovers |

The HTML tool is the front door. nudgeDSL is what you graduate to. They are the same framework at different levels of formalization.

---

## What nudgeDSL Does NOT Replace

- The AI role profiles (Architecture Critic, Dependency Analyst, etc.) — these are behavioral configurations, not document formats
- The session continuation heuristic (60% file surface overlap rule) — this is human judgment
- The permanent tooling layer (get_skeleton, find_symbol, run_task) — these are execution tools
- The code inside deliverables — always native language

nudgeDSL replaces the **document format** of the orchestration artifacts. Everything else in the framework stays exactly as specified in the playbook.

---

## Atom Registry for Nudge Framework (Draft v0.1)

These are the atoms specific to the Nudge Framework domain. They use the standard nudgeDSL grammar from the spec.

```json
{
  "domain": "nudge-framework",
  "version": "0.1",
  "atoms": [
    { "atom": "BLUEPRINT", "fn": "InitBlueprint", "args": [{"name":"id","type":"string"},{"name":"version","type":"string"}] },
    { "atom": "STACK", "fn": "DeclareStack", "args": [{"name":"tech","type":"string"}] },
    { "atom": "RISK", "fn": "RegisterRisk", "args": [{"name":"id","type":"string"},{"name":"severity","type":"integer","min":1,"max":3}] },
    { "atom": "CONSTRAINT", "fn": "AddConstraint", "args": [{"name":"rule","type":"string"}] },
    { "atom": "TASK", "fn": "RegisterTask", "args": [{"name":"id","type":"string"}] },
    { "atom": "SHARD", "fn": "OpenShard", "args": [{"name":"id","type":"string"},{"name":"phase","type":"string"}] },
    { "atom": "SCOPE", "fn": "AddScopeFile", "args": [{"name":"file","type":"string"}] },
    { "atom": "ACCEPT", "fn": "AddAcceptanceCriteria", "args": [{"name":"criterion","type":"string"}] },
    { "atom": "EXCLUDE", "fn": "AddExclusion", "args": [{"name":"file","type":"string"}] },
    { "atom": "STOP", "fn": "SetStopCondition", "args": [{"name":"message","type":"string"}] },
    { "atom": "MARK", "fn": "UpdateStatus", "args": [{"name":"id","type":"string"},{"name":"status","type":"string","enum":["done","pending","skipped","blocked"]}] },
    { "atom": "CREATE", "fn": "RecordCreatedFile", "args": [{"name":"path","type":"string"}] },
    { "atom": "MODIFY", "fn": "RecordModifiedFile", "args": [{"name":"path","type":"string"}] },
    { "atom": "NOTE", "fn": "AddDecisionNote", "args": [{"name":"text","type":"string"}] },
    { "atom": "NEXT", "fn": "SetNextSlice", "args": [{"name":"id","type":"string"}] },
    { "atom": "FLAG", "fn": "RaiseFlag", "args": [{"name":"type","type":"string","enum":["regression","out-of-scope","blocker","risk"]},{"name":"scope","type":"string"}] },
    { "atom": "VERIFY", "fn": "OpenVerification", "args": [{"name":"shard_id","type":"string"}] },
    { "atom": "RESULT", "fn": "SetVerificationResult", "args": [{"name":"verdict","type":"string","enum":["ready","revision","blocked"]}] }
  ]
}
```

---

*nudge-framework-alignment-v0.1.md — pre-implementation. Atom registry is a draft and will evolve with the framework v2 rewrite.*
