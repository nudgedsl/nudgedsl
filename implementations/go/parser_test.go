package nudgedsl

import (
	"testing"
)

// ── Registry fixture ───────────────────────────────────────────────────────

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	minOne := 1.0
	maxHundred := 100.0

	reg, err := NewRegistry([]AtomDef{
		{
			Atom: "MARK",
			Fn:   "UpdateStatus",
			Args: []ArgDef{
				{Name: "id", Type: ArgString},
				{Name: "status", Type: ArgString, Enum: []string{"pending", "done", "skipped", "error"}},
			},
		},
		{
			Atom: "NOTIFY",
			Fn:   "BroadcastEvent",
			Args: []ArgDef{
				{Name: "channel", Type: ArgString},
			},
		},
		{
			Atom: "FETCH",
			Fn:   "RetrieveData",
			Args: []ArgDef{
				{Name: "source", Type: ArgString},
			},
		},
		{
			Atom: "PING",
			Fn:   "HealthCheck",
			Args: []ArgDef{
				{Name: "count", Type: ArgInteger, Min: &minOne, Max: &maxHundred},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build test registry: %v", err)
	}
	return reg
}

// ── Helpers ────────────────────────────────────────────────────────────────

func assertParseError(t *testing.T, id, input string, expectedCode ParseErrorCode) {
	t.Helper()
	_, parseErr := Parse(input, nil)
	if parseErr == nil {
		t.Errorf("[%s] expected ParseError %q, got nil (input: %q)", id, expectedCode, input)
		return
	}
	if parseErr.Code != expectedCode {
		t.Errorf("[%s] expected ParseError code %q, got %q (msg: %s)", id, expectedCode, parseErr.Code, parseErr.Message)
	}
}

func assertValidationError(t *testing.T, id, input string, expectedCode ValidationErrorCode, reg *Registry) {
	t.Helper()
	ast, parseErr, valErrs := ParseAndValidate(input, reg)
	if parseErr != nil {
		t.Errorf("[%s] expected ValidationError %q but got ParseError %q instead", id, expectedCode, parseErr.Code)
		return
	}
	if ast == nil {
		t.Errorf("[%s] ast is nil without parse error", id)
		return
	}
	for _, ve := range valErrs {
		if ve.Code == expectedCode {
			return
		}
	}
	t.Errorf("[%s] expected ValidationError code %q, got errors: %v", id, expectedCode, valErrs)
}

func assertValid(t *testing.T, id, input string, reg *Registry) *AST {
	t.Helper()
	ast, parseErr, valErrs := ParseAndValidate(input, reg)
	if parseErr != nil {
		t.Errorf("[%s] expected valid parse, got ParseError %q: %s", id, parseErr.Code, parseErr.Message)
		return nil
	}
	if len(valErrs) > 0 {
		t.Errorf("[%s] expected valid, got %d validation error(s): %v", id, len(valErrs), valErrs)
		return nil
	}
	if ast == nil {
		t.Errorf("[%s] ast is nil with no errors", id)
	}
	return ast
}

// ── Empty input (F001-F003) ────────────────────────────────────────────────

func TestF001_EmptyString(t *testing.T) {
	assertParseError(t, "F001", "", ErrEmptyInput)
}

func TestF002_WhitespaceOnly(t *testing.T) {
	assertParseError(t, "F002", "   ", ErrEmptyInput)
}

func TestF003_NewlinesOnly(t *testing.T) {
	assertParseError(t, "F003", "\n\n\t", ErrEmptyInput)
}

// ── Truncated input (F004-F010) ────────────────────────────────────────────

func TestF004_TruncatedOpenParen(t *testing.T) {
	assertParseError(t, "F004", "MARK(", ErrTruncatedInput)
}

func TestF005_TruncatedMissingCloseParen(t *testing.T) {
	assertParseError(t, "F005", `MARK("task-1"`, ErrTruncatedInput)
}

func TestF006_TrailingChainOperator(t *testing.T) {
	assertParseError(t, "F006", `MARK("task-1", "done") >>`, ErrTrailingOperator)
}

func TestF007_TrailingFallbackOperator(t *testing.T) {
	assertParseError(t, "F007", `FETCH("db") |`, ErrTrailingOperator)
}

func TestF008_TrailingParallelOperator(t *testing.T) {
	assertParseError(t, "F008", `FETCH("db") //`, ErrTrailingOperator)
}

func TestF009_TrailingAmplifyOperator(t *testing.T) {
	assertParseError(t, "F009", "PING() **", ErrTrailingOperator)
}

func TestF010_UnterminatedStringInArg(t *testing.T) {
	assertParseError(t, "F010", `MARK("task-1", "do`, ErrUnterminatedStr)
}

// ── Missing parens (F011-F015) ─────────────────────────────────────────────

func TestF011_AtomNoParens(t *testing.T) {
	assertParseError(t, "F011", "MARK", ErrUnexpectedToken)
}

func TestF012_AtomStringNoParens(t *testing.T) {
	assertParseError(t, "F012", `MARK "task-1"`, ErrUnexpectedToken)
}

func TestF013_MissingCloseParenReplacedByOperator(t *testing.T) {
	assertParseError(t, "F013", `MARK("task-1" >> NOTIFY("ops")`, ErrMissingCloseParen)
}

func TestF014_OuterGroupNeverClosed(t *testing.T) {
	assertParseError(t, "F014", `(MARK("task-1", "done") >> NOTIFY("ops")`, ErrMissingCloseParen)
}

func TestF015_ExtraCloseParen(t *testing.T) {
	assertParseError(t, "F015", `MARK("task-1", "done"))`, ErrUnexpectedToken)
}

// ── Trailing/leading operators (F016-F019) ─────────────────────────────────

func TestF016_ChainAtStart(t *testing.T) {
	assertParseError(t, "F016", `>> MARK("task-1", "done")`, ErrUnexpectedToken)
}

func TestF017_FallbackAtStart(t *testing.T) {
	assertParseError(t, "F017", `| FETCH("db")`, ErrUnexpectedToken)
}

func TestF018_DoubleChainOperator(t *testing.T) {
	assertParseError(t, "F018", `MARK("a", "done") >> >> NOTIFY("ops")`, ErrUnexpectedToken)
}

func TestF019_MixedOperatorsNoOperand(t *testing.T) {
	assertParseError(t, "F019", `MARK("a", "done") >> | NOTIFY("ops")`, ErrUnexpectedToken)
}

// ── Unknown atoms (F020-F024) ──────────────────────────────────────────────

func TestF020_LowercaseAtom(t *testing.T) {
	assertParseError(t, "F020", `xyz("task-1")`, ErrUnknownAtom)
}

func TestF021_KnownNameLowercase(t *testing.T) {
	assertParseError(t, "F021", `mark("task-1", "done")`, ErrUnknownAtom)
}

func TestF022_AtomTooLong(t *testing.T) {
	// MARK123456 — longer than 3 chars — still emitted as TAtom but registry rejects it
	reg := testRegistry(t)
	assertParseError(t, "F022", `MARK123456("task-1")`, ErrUnknownAtom)
	_ = reg
}

func TestF023_AtomStartsWithDigit(t *testing.T) {
	assertParseError(t, "F023", `1MARK("task-1")`, ErrUnexpectedToken)
}

func TestF024_AtomStartsWithUnderscore(t *testing.T) {
	assertParseError(t, "F024", `_MARK("task-1")`, ErrUnexpectedToken)
}

// ── Bad arg types (F025-F029) — ValidationError ────────────────────────────

func TestF025_IntegerWhereStringExpected(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F025", `MARK(42, "done")`, ErrArgTypeMismatch, reg)
}

func TestF026_IntegerForEnumArg(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F026", `MARK("task-1", 99)`, ErrArgTypeMismatch, reg)
}

func TestF027_BoolWhereStringExpected(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F027", `MARK(true, "done")`, ErrArgTypeMismatch, reg)
}

func TestF028_NullWhereStringExpected(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F028", `MARK(null, "done")`, ErrArgTypeMismatch, reg)
}

func TestF029_StringWhereIntegerExpected(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F029", `PING("not-a-number")`, ErrArgTypeMismatch, reg)
}

// ── Bad arg counts (F030-F033) — ValidationError ───────────────────────────

func TestF030_ZeroArgsWhereTwoRequired(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F030", `MARK()`, ErrArgCountMismatch, reg)
}

func TestF031_OneArgWhereTwoRequired(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F031", `MARK("task-1")`, ErrArgCountMismatch, reg)
}

func TestF032_ThreeArgsWhereTwoExpected(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F032", `MARK("task-1", "done", "extra")`, ErrArgCountMismatch, reg)
}

func TestF033_ZeroArgsWhereOneRequired(t *testing.T) {
	reg := testRegistry(t)
	assertValidationError(t, "F033", `NOTIFY()`, ErrArgCountMismatch, reg)
}

// ── Unterminated strings (F034-F036) ──────────────────────────────────────

func TestF034_FirstStringNotClosed(t *testing.T) {
	assertParseError(t, "F034", `MARK("task-1, "done")`, ErrUnterminatedStr)
}

func TestF035_StringClosedByParen(t *testing.T) {
	assertParseError(t, "F035", `NOTIFY("ops)`, ErrUnterminatedStr)
}

func TestF036_SecondArgNotClosed(t *testing.T) {
	assertParseError(t, "F036", `MARK("task-1", "done) >> NOTIFY("ops")`, ErrUnterminatedStr)
}

// ── Nested paren errors (F037-F039) ───────────────────────────────────────

func TestF037_DoubleWrappedGrouping(t *testing.T) {
	assertParseError(t, "F037", `((MARK("task-1", "done")))`, ErrUnexpectedToken)
}

func TestF038_EmptyGrouping(t *testing.T) {
	assertParseError(t, "F038", "()", ErrUnexpectedToken)
}

func TestF039_OuterGroupNotClosed(t *testing.T) {
	assertParseError(t, "F039", `(MARK("a", "done") >> (NOTIFY("ops") >> FETCH("db"))`, ErrMissingCloseParen)
}

// ── Amplify operator misuse (F040-F043) ───────────────────────────────────

func TestF040_AmplifyCountZero(t *testing.T) {
	assertParseError(t, "F040", `MARK("a", "done") ** 0`, ErrUnexpectedToken)
}

func TestF041_AmplifyCountNegative(t *testing.T) {
	assertParseError(t, "F041", `MARK("a", "done") ** -1`, ErrUnexpectedToken)
}

func TestF042_AmplifyCountFloat(t *testing.T) {
	assertParseError(t, "F042", `MARK("a", "done") ** 2.5`, ErrUnexpectedToken)
}

func TestF043_AmplifyCountString(t *testing.T) {
	assertParseError(t, "F043", `MARK("a", "done") ** "three"`, ErrUnexpectedToken)
}

// ── Whitespace variants — must be VALID (F044-F047) ────────────────────────

func TestF044_WhitespaceBetweenAtomAndParen(t *testing.T) {
	reg := testRegistry(t)
	assertValid(t, "F044", `MARK  ("task-1", "done")`, reg)
}

func TestF045_NoSpaceAfterComma(t *testing.T) {
	reg := testRegistry(t)
	assertValid(t, "F045", `MARK("task-1","done")`, reg)
}

func TestF046_ExtraSpacesAroundArgs(t *testing.T) {
	reg := testRegistry(t)
	assertValid(t, "F046", `MARK( "task-1" , "done" )`, reg)
}

func TestF047_NewlineBeforeOperator(t *testing.T) {
	reg := testRegistry(t)
	assertValid(t, "F047", "MARK(\"task-1\", \"done\")\n>> NOTIFY(\"ops\")", reg)
}

// ── Natural language bleed (F048-F052) ────────────────────────────────────

func TestF048_NaturalLanguageOnly(t *testing.T) {
	assertParseError(t, "F048", "Please mark task-1 as done.", ErrUnknownAtom)
}

func TestF049_LLMPreamble(t *testing.T) {
	assertParseError(t, "F049", `Sure! Here is the nudgeDSL: MARK("task-1", "done")`, ErrUnknownAtom)
}

func TestF050_MarkdownCodeFence(t *testing.T) {
	assertParseError(t, "F050", "```nudgedsl\nMARK(\"task-1\", \"done\")\n```", ErrUnexpectedToken)
}

func TestF051_NaturalWordAfterParallel(t *testing.T) {
	assertParseError(t, "F051", `MARK("a", "done") // done`, ErrUnknownAtom)
}

func TestF052_InlineComment(t *testing.T) {
	assertParseError(t, "F052", `MARK("a", "done") # update the index`, ErrUnexpectedToken)
}

// ── Valid boundary cases (F053-F055) ──────────────────────────────────────

func TestF053_AmplifyCountOne(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "F053", `MARK("a", "done") ** 1`, reg)
	if ast != nil && ast.Root.Type != NodeAmplify {
		t.Errorf("F053: expected NodeAmplify root, got %s", ast.Root.Type)
	}
}

func TestF054_SameAtomBothSidesOfFallback(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "F054", `FETCH("x") | FETCH("x")`, reg)
	if ast != nil && ast.Root.Type != NodeFallback {
		t.Errorf("F054: expected NodeFallback root, got %s", ast.Root.Type)
	}
}

func TestF055_ValidEnumValue(t *testing.T) {
	reg := testRegistry(t)
	assertValid(t, "F055", `MARK("task-1", "skipped")`, reg)
}

// ── AST structure checks ───────────────────────────────────────────────────

func TestAST_ChainProducesCorrectNodes(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "AST-CHAIN", `MARK("task-1", "done") >> NOTIFY("ops")`, reg)
	if ast == nil {
		return
	}
	if ast.Root.Type != NodeChain {
		t.Fatalf("expected NodeChain, got %s", ast.Root.Type)
	}
	if len(ast.Root.Nodes) != 2 {
		t.Fatalf("expected 2 chain nodes, got %d", len(ast.Root.Nodes))
	}
	if ast.Root.Nodes[0].Atom != "MARK" {
		t.Errorf("expected first node atom MARK, got %s", ast.Root.Nodes[0].Atom)
	}
	if ast.Root.Nodes[1].Atom != "NOTIFY" {
		t.Errorf("expected second node atom NOTIFY, got %s", ast.Root.Nodes[1].Atom)
	}
}

func TestAST_FallbackProducesCorrectNodes(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "AST-FALLBACK", `FETCH("primary") | FETCH("replica")`, reg)
	if ast == nil {
		return
	}
	if ast.Root.Type != NodeFallback {
		t.Fatalf("expected NodeFallback, got %s", ast.Root.Type)
	}
}

func TestAST_AmplifyPrecedenceBeforeParallel(t *testing.T) {
	// A() // B() ** 3 should parse as A() // (B() ** 3)
	// Using NOTIFY and FETCH as stand-ins for A and B
	reg := testRegistry(t)
	ast := assertValid(t, "AST-PREC", `NOTIFY("a") // NOTIFY("b") ** 3`, reg)
	if ast == nil {
		return
	}
	if ast.Root.Type != NodeParallel {
		t.Fatalf("expected NodeParallel root, got %s", ast.Root.Type)
	}
	right := ast.Root.Nodes[1]
	if right.Type != NodeAmplify {
		t.Errorf("expected right branch to be NodeAmplify, got %s", right.Type)
	}
	if right.Count != 3 {
		t.Errorf("expected amplify count 3, got %d", right.Count)
	}
}

func TestAST_Version(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "AST-VER", `MARK("x", "done")`, reg)
	if ast == nil {
		return
	}
	if ast.Version != specVersion {
		t.Errorf("expected version %q, got %q", specVersion, ast.Version)
	}
}

func TestAST_CallFnPopulated(t *testing.T) {
	reg := testRegistry(t)
	ast := assertValid(t, "AST-FN", `MARK("x", "done")`, reg)
	if ast == nil {
		return
	}
	if ast.Root.Fn != "UpdateStatus" {
		t.Errorf("expected fn UpdateStatus, got %q", ast.Root.Fn)
	}
}

// ── Registry validation ────────────────────────────────────────────────────

func TestRegistry_RollbackNotFound(t *testing.T) {
	_, err := NewRegistry([]AtomDef{
		{
			Atom:     "HEAL",
			Fn:       "HealUnit",
			Args:     []ArgDef{{Name: "target", Type: ArgString}},
			Rollback: "UNHEAL", // UNHEAL not in registry
		},
	})
	if err == nil {
		t.Fatal("expected RegistryError, got nil")
	}
	re, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected *RegistryError, got %T", err)
	}
	if re.Code != ErrRollbackNotFound {
		t.Errorf("expected ROLLBACK_NOT_FOUND, got %s", re.Code)
	}
}

func TestRegistry_RollbackSignatureMismatch(t *testing.T) {
	_, err := NewRegistry([]AtomDef{
		{
			Atom:     "HEAL",
			Fn:       "HealUnit",
			Args:     []ArgDef{{Name: "target", Type: ArgString}, {Name: "amount", Type: ArgFloat}},
			Rollback: "UNHEAL",
		},
		{
			Atom: "UNHEAL",
			Fn:   "UnhealUnit",
			Args: []ArgDef{{Name: "target", Type: ArgString}}, // only 1 arg, HEAL has 2
		},
	})
	if err == nil {
		t.Fatal("expected RegistryError, got nil")
	}
	re, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected *RegistryError, got %T", err)
	}
	if re.Code != ErrRollbackSignatureMismatch {
		t.Errorf("expected ROLLBACK_SIGNATURE_MISMATCH, got %s", re.Code)
	}
}

func TestRegistry_ValidRollback(t *testing.T) {
	_, err := NewRegistry([]AtomDef{
		{
			Atom:     "HEAL",
			Fn:       "HealUnit",
			Args:     []ArgDef{{Name: "target", Type: ArgString}, {Name: "amount", Type: ArgFloat}},
			Rollback: "UNHEAL",
		},
		{
			Atom: "UNHEAL",
			Fn:   "UnhealUnit",
			Args: []ArgDef{{Name: "target", Type: ArgString}, {Name: "amount", Type: ArgFloat}},
		},
	})
	if err != nil {
		t.Errorf("expected valid registry, got error: %v", err)
	}
}
