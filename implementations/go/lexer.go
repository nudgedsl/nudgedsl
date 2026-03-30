package nudgedsl

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenKind classifies a lexed token.
type TokenKind int

const (
	TAtom        TokenKind = iota // [A-Z][A-Z0-9]{0,2}
	TString                       // "..."
	TInteger                      // bare integer
	TFloat                        // decimal number
	TBool                         // true | false
	TNull                         // null
	TLParen                       // (
	TRParen                       // )
	TComma                        // ,
	TChain                        // >>
	TFallback                     // |
	TParallel                     // //
	TAmplify                      // **
	TEOF
)

var tokenKindNames = map[TokenKind]string{
	TAtom:     "ATOM",
	TString:   "STRING",
	TInteger:  "INTEGER",
	TFloat:    "FLOAT",
	TBool:     "BOOL",
	TNull:     "NULL",
	TLParen:   "(",
	TRParen:   ")",
	TComma:    ",",
	TChain:    ">>",
	TFallback: "|",
	TParallel: "//",
	TAmplify:  "**",
	TEOF:      "EOF",
}

func (k TokenKind) String() string {
	if s, ok := tokenKindNames[k]; ok {
		return s
	}
	return fmt.Sprintf("Token(%d)", int(k))
}

// Token is a single lexed unit with its source position.
type Token struct {
	Kind    TokenKind
	Value   string // raw source text of the token
	Pos     int    // byte offset of first character in original input
}

// Lexer converts a raw input string into a token stream.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
	err    *ParseError
}

// Tokenize runs the lexer and returns all tokens or the first error.
func Tokenize(input string) ([]Token, *ParseError) {
	l := &Lexer{input: input}
	l.run()
	return l.tokens, l.err
}

func (l *Lexer) run() {
	// Empty / whitespace-only input
	if strings.TrimSpace(l.input) == "" {
		l.err = l.makeError(ErrEmptyInput, 0, "input is empty or whitespace only", "", "non-empty nudgeDSL expression")
		return
	}

	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		start := l.pos
		ch := l.input[l.pos]

		switch {
		case ch == '"':
			l.lexString(start)
		case ch == '(':
			l.emit(TLParen, start, 1)
		case ch == ')':
			l.emit(TRParen, start, 1)
		case ch == ',':
			l.emit(TComma, start, 1)
		case ch == '>' && l.peek(1) == '>':
			l.emit(TChain, start, 2)
		case ch == '|':
			l.emit(TFallback, start, 1)
		case ch == '/' && l.peek(1) == '/':
			l.emit(TParallel, start, 2)
		case ch == '*' && l.peek(1) == '*':
			l.emit(TAmplify, start, 2)
		case ch == '-' || (ch >= '0' && ch <= '9'):
			l.lexNumber(start)
		case ch >= 'A' && ch <= 'Z':
			l.lexAtomOrKeyword(start)
		case ch >= 'a' && ch <= 'z':
			l.lexLowerWord(start)
		default:
			l.err = l.makeError(ErrUnexpectedToken, start,
				fmt.Sprintf("unexpected character %q", string(ch)),
				string(ch), "atom, operator, or argument")
			return
		}

		if l.err != nil {
			return
		}
	}

	l.tokens = append(l.tokens, Token{Kind: TEOF, Pos: l.pos})
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t' ||
		l.input[l.pos] == '\n' || l.input[l.pos] == '\r') {
		l.pos++
	}
}

func (l *Lexer) peek(offset int) byte {
	p := l.pos + offset
	if p < len(l.input) {
		return l.input[p]
	}
	return 0
}

func (l *Lexer) emit(kind TokenKind, start, length int) {
	l.tokens = append(l.tokens, Token{
		Kind:  kind,
		Value: l.input[start : start+length],
		Pos:   start,
	})
	l.pos = start + length
}

func (l *Lexer) lexString(start int) {
	l.pos++ // consume opening quote
	for l.pos < len(l.input) {
		if l.input[l.pos] == '"' {
			// closing quote found
			l.tokens = append(l.tokens, Token{
				Kind:  TString,
				Value: l.input[start+1 : l.pos], // value without quotes
				Pos:   start,
			})
			l.pos++ // consume closing quote
			return
		}
		l.pos++
	}
	// Reached end of input without closing quote
	l.err = l.makeError(ErrUnterminatedStr, start,
		"string opened but never closed", l.input[start:], `"`)
}

func (l *Lexer) lexNumber(start int) {
	i := l.pos
	if l.input[i] == '-' {
		i++
	}
	if i >= len(l.input) || l.input[i] < '0' || l.input[i] > '9' {
		// lone minus
		l.err = l.makeError(ErrUnexpectedToken, start, "unexpected '-'", "-", "digit")
		return
	}
	isFloat := false
	for i < len(l.input) && l.input[i] >= '0' && l.input[i] <= '9' {
		i++
	}
	if i < len(l.input) && l.input[i] == '.' {
		isFloat = true
		i++
		for i < len(l.input) && l.input[i] >= '0' && l.input[i] <= '9' {
			i++
		}
	}
	kind := TInteger
	if isFloat {
		kind = TFloat
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: l.input[start:i], Pos: start})
	l.pos = i
}

func (l *Lexer) lexAtomOrKeyword(start int) {
	i := l.pos
	for i < len(l.input) && (l.input[i] >= 'A' && l.input[i] <= 'Z' || l.input[i] >= '0' && l.input[i] <= '9') {
		i++
	}
	word := l.input[start:i]
	// Atoms are 1–3 uppercase chars/digits, must start with letter
	// Longer words are also emitted as TAtom — registry lookup catches unknown atoms
	l.tokens = append(l.tokens, Token{Kind: TAtom, Value: word, Pos: start})
	l.pos = i
}

func (l *Lexer) lexLowerWord(start int) {
	i := l.pos
	for i < len(l.input) && (unicode.IsLetter(rune(l.input[i])) || l.input[i] == '_') {
		i++
	}
	word := l.input[start:i]
	switch word {
	case "true", "false":
		l.tokens = append(l.tokens, Token{Kind: TBool, Value: word, Pos: start})
	case "null":
		l.tokens = append(l.tokens, Token{Kind: TNull, Value: word, Pos: start})
	default:
		// Lowercase word that isn't a keyword — natural language bleed or unknown atom
		l.err = l.makeError(ErrUnknownAtom, start,
			fmt.Sprintf("%q is not a valid atom (atoms must be uppercase)", word),
			word, "uppercase atom")
	}
	l.pos = i
}

func (l *Lexer) makeError(code ParseErrorCode, pos int, msg, got, expected string) *ParseError {
	return &ParseError{
		Code:     code,
		Position: pos,
		Message:  msg,
		Got:      got,
		Expected: expected,
		Input:    l.input,
	}
}
