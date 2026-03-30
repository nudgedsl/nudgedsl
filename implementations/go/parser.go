package nudgedsl

import (
	"fmt"
	"strconv"
)

const specVersion = "0.1.0"

// Parser converts a token stream into an AST.
// Operator precedence (low to high): fallback < parallel < chain < amplify
type Parser struct {
	tokens   []Token
	pos      int
	input    string
	registry *Registry // optional — if nil, unknown atoms are not validated at parse time
}

// Parse is the main entry point. Returns an AST or a ParseError.
// Registry is optional here; pass it to also catch UNKNOWN_ATOM at parse time.
// Full semantic validation happens separately via Validate().
func Parse(input string, registry *Registry) (*AST, *ParseError) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	p := &Parser{tokens: tokens, input: input, registry: registry}
	root, perr := p.parseExpression()
	if perr != nil {
		return nil, perr
	}

	// Must have consumed all tokens
	if p.current().Kind != TEOF {
		tok := p.current()
		return nil, p.makeError(ErrUnexpectedToken, tok.Pos,
			fmt.Sprintf("unexpected token after expression: %q", tok.Value),
			tok.Value, "EOF")
	}

	return &AST{Version: specVersion, Root: root}, nil
}

// ── Precedence levels ──────────────────────────────────────────────────────

// parseExpression is the entry point for the grammar.
// expression ::= fallback
func (p *Parser) parseExpression() (*Node, *ParseError) {
	return p.parseFallback()
}

// fallback ::= parallel ( "|" parallel )*
func (p *Parser) parseFallback() (*Node, *ParseError) {
	left, err := p.parseParallel()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TFallback {
		return left, nil
	}

	nodes := []*Node{left}
	for p.current().Kind == TFallback {
		p.advance() // consume |
		if p.current().Kind == TEOF || p.isOperator(p.current().Kind) {
			return nil, p.makeError(ErrTrailingOperator, p.current().Pos,
				"fallback operator | has no right-hand side",
				p.current().Value, "atom or expression")
		}
		right, err := p.parseParallel()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, right)
	}

	return &Node{Type: NodeFallback, Nodes: nodes}, nil
}

// parallel ::= chain ( "//" chain )*
func (p *Parser) parseParallel() (*Node, *ParseError) {
	left, err := p.parseChain()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TParallel {
		return left, nil
	}

	nodes := []*Node{left}
	for p.current().Kind == TParallel {
		p.advance() // consume //
		if p.current().Kind == TEOF || p.isOperator(p.current().Kind) {
			return nil, p.makeError(ErrTrailingOperator, p.current().Pos,
				"parallel operator // has no right-hand side",
				p.current().Value, "atom or expression")
		}
		right, err := p.parseChain()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, right)
	}

	return &Node{Type: NodeParallel, Nodes: nodes, FailureMode: FailFast}, nil
}

// chain ::= amplify ( ">>" amplify )*
func (p *Parser) parseChain() (*Node, *ParseError) {
	left, err := p.parseAmplify()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TChain {
		return left, nil
	}

	nodes := []*Node{left}
	for p.current().Kind == TChain {
		p.advance() // consume >>
		if p.current().Kind == TEOF || p.isOperator(p.current().Kind) {
			return nil, p.makeError(ErrTrailingOperator, p.current().Pos,
				"chain operator >> has no right-hand side",
				p.current().Value, "atom or expression")
		}
		right, err := p.parseAmplify()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, right)
	}

	return &Node{Type: NodeChain, Nodes: nodes}, nil
}

// amplify ::= primary ( "**" INTEGER )?
func (p *Parser) parseAmplify() (*Node, *ParseError) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TAmplify {
		return node, nil
	}

	ampPos := p.current().Pos
	p.advance() // consume **

	// Must be followed by a positive integer
	if p.current().Kind != TInteger {
		return nil, p.makeError(ErrUnexpectedToken, p.current().Pos,
			fmt.Sprintf("** must be followed by a positive integer, got %q", p.current().Value),
			p.current().Value, "positive integer")
	}

	n, convErr := strconv.Atoi(p.current().Value)
	if convErr != nil || n < 1 {
		return nil, p.makeError(ErrUnexpectedToken, ampPos,
			fmt.Sprintf("amplify count must be a positive integer >= 1, got %q", p.current().Value),
			p.current().Value, "integer >= 1")
	}
	p.advance() // consume count

	return &Node{Type: NodeAmplify, Node: node, Count: n}, nil
}

// primary ::= atom_call | "(" expression ")"
func (p *Parser) parsePrimary() (*Node, *ParseError) {
	tok := p.current()

	switch tok.Kind {
	case TAtom:
		return p.parseAtomCall()

	case TLParen:
		p.advance() // consume (
		if p.current().Kind == TRParen {
			return nil, p.makeError(ErrUnexpectedToken, p.current().Pos,
				"empty grouping () is not valid", "()", "expression")
		}
		inner, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().Kind != TRParen {
			return nil, p.makeError(ErrMissingCloseParen, p.current().Pos,
				"expected ) to close grouped expression",
				p.current().Value, ")")
		}
		p.advance() // consume )
		return inner, nil

	case TEOF:
		return nil, p.makeError(ErrTruncatedInput, tok.Pos,
			"expression ended unexpectedly", "EOF", "atom or (")

	default:
		if p.isOperator(tok.Kind) {
			return nil, p.makeError(ErrUnexpectedToken, tok.Pos,
				fmt.Sprintf("operator %q at start of expression or after another operator", tok.Value),
				tok.Value, "atom or (")
		}
		return nil, p.makeError(ErrUnexpectedToken, tok.Pos,
			fmt.Sprintf("unexpected token %q", tok.Value),
			tok.Value, "atom or (")
	}
}

// atom_call ::= ATOM "(" arg_list? ")"
func (p *Parser) parseAtomCall() (*Node, *ParseError) {
	atomTok := p.current()
	p.advance() // consume atom name

	// Check registry if present
	var fn string
	if p.registry != nil {
		def := p.registry.Lookup(atomTok.Value)
		if def == nil {
			return nil, p.makeError(ErrUnknownAtom, atomTok.Pos,
				fmt.Sprintf("atom %q is not registered", atomTok.Value),
				atomTok.Value, "registered atom name")
		}
		fn = def.Fn
	}

	// Expect (
	if p.current().Kind != TLParen {
		// Whitespace between atom and paren is consumed by lexer already — 
		// but if we land here with no paren, it's a syntax error
		return nil, p.makeError(ErrUnexpectedToken, p.current().Pos,
			fmt.Sprintf("expected ( after atom %q, got %q", atomTok.Value, p.current().Value),
			p.current().Value, "(")
	}
	p.advance() // consume (

	// Parse args
	args, err := p.parseArgList()
	if err != nil {
		return nil, err
	}

	// Expect )
	if p.current().Kind == TEOF {
		return nil, p.makeError(ErrTruncatedInput, p.current().Pos,
			fmt.Sprintf("atom %q call truncated — missing )", atomTok.Value),
			"EOF", ")")
	}
	if p.current().Kind != TRParen {
		return nil, p.makeError(ErrMissingCloseParen, p.current().Pos,
			fmt.Sprintf("expected ) to close atom %q call, got %q", atomTok.Value, p.current().Value),
			p.current().Value, ")")
	}
	p.advance() // consume )

	return &Node{
		Type: NodeCall,
		Atom: atomTok.Value,
		Fn:   fn,
		Args: args,
	}, nil
}

// parseArgList parses zero or more comma-separated arguments.
func (p *Parser) parseArgList() ([]interface{}, *ParseError) {
	var args []interface{}

	if p.current().Kind == TRParen {
		return args, nil // empty arg list
	}

	for {
		arg, err := p.parseArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if p.current().Kind != TComma {
			break
		}
		p.advance() // consume ,
	}

	return args, nil
}

// parseArg parses a single argument value.
func (p *Parser) parseArg() (interface{}, *ParseError) {
	tok := p.current()

	switch tok.Kind {
	case TString:
		p.advance()
		return tok.Value, nil

	case TInteger:
		p.advance()
		n, _ := strconv.Atoi(tok.Value)
		return n, nil

	case TFloat:
		p.advance()
		f, _ := strconv.ParseFloat(tok.Value, 64)
		return f, nil

	case TBool:
		p.advance()
		return tok.Value == "true", nil

	case TNull:
		p.advance()
		return nil, nil

	case TEOF:
		return nil, p.makeError(ErrTruncatedInput, tok.Pos,
			"argument list truncated at EOF", "EOF", "argument value or )")

	default:
		return nil, p.makeError(ErrUnexpectedToken, tok.Pos,
			fmt.Sprintf("expected argument value, got %q", tok.Value),
			tok.Value, "string, integer, float, boolean, or null")
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (p *Parser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Kind: TEOF, Pos: len(p.input)}
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) isOperator(k TokenKind) bool {
	return k == TChain || k == TFallback || k == TParallel || k == TAmplify
}

func (p *Parser) makeError(code ParseErrorCode, pos int, msg, got, expected string) *ParseError {
	return &ParseError{
		Code:     code,
		Position: pos,
		Message:  msg,
		Got:      got,
		Expected: expected,
		Input:    p.input,
	}
}
