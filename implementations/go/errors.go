package nudgedsl

import "fmt"

// ParseErrorCode identifies the category of a parse failure.
type ParseErrorCode string

const (
	ErrEmptyInput        ParseErrorCode = "EMPTY_INPUT"
	ErrTruncatedInput    ParseErrorCode = "TRUNCATED_INPUT"
	ErrUnexpectedToken   ParseErrorCode = "UNEXPECTED_TOKEN"
	ErrUnterminatedStr   ParseErrorCode = "UNTERMINATED_STRING"
	ErrMissingCloseParen ParseErrorCode = "MISSING_CLOSE_PAREN"
	ErrTrailingOperator  ParseErrorCode = "TRAILING_OPERATOR"
	ErrUnknownAtom       ParseErrorCode = "UNKNOWN_ATOM"
)

// ParseError is returned by the lexer or parser. Never panics.
type ParseError struct {
	Code     ParseErrorCode `json:"code"`
	Position int            `json:"position"`
	Expected string         `json:"expected,omitempty"`
	Got      string         `json:"got,omitempty"`
	Input    string         `json:"input"`
	Message  string         `json:"message"`
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError[%s] at %d: %s (got %q, expected %q)",
		e.Code, e.Position, e.Message, e.Got, e.Expected)
}

// ValidationErrorCode identifies the category of a semantic failure.
type ValidationErrorCode string

const (
	ErrArgOutOfRange    ValidationErrorCode = "ARG_OUT_OF_RANGE"
	ErrArgNotInEnum     ValidationErrorCode = "ARG_NOT_IN_ENUM"
	ErrArgTypeMismatch  ValidationErrorCode = "ARG_TYPE_MISMATCH"
	ErrArgCountMismatch ValidationErrorCode = "ARG_COUNT_MISMATCH"
	ErrUnknownAtomV     ValidationErrorCode = "UNKNOWN_ATOM"
)

// ValidationError is returned by the semantic validator after a successful parse.
type ValidationError struct {
	Code       ValidationErrorCode `json:"code"`
	Atom       string              `json:"atom"`
	Arg        string              `json:"arg,omitempty"`
	Value      interface{}         `json:"value,omitempty"`
	Constraint string              `json:"constraint,omitempty"`
	Message    string              `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("ValidationError[%s] atom=%s arg=%s: %s",
		e.Code, e.Atom, e.Arg, e.Message)
}

// RegistryErrorCode identifies registry load failures.
type RegistryErrorCode string

const (
	ErrRollbackNotFound          RegistryErrorCode = "ROLLBACK_NOT_FOUND"
	ErrRollbackSignatureMismatch RegistryErrorCode = "ROLLBACK_SIGNATURE_MISMATCH"
)

// RegistryError is returned when loading an atom registry fails validation.
type RegistryError struct {
	Code     RegistryErrorCode `json:"code"`
	Atom     string            `json:"atom"`
	Rollback string            `json:"rollback"`
	Detail   string            `json:"detail"`
}

func (e *RegistryError) Error() string {
	return fmt.Sprintf("RegistryError[%s] atom=%s rollback=%s: %s",
		e.Code, e.Atom, e.Rollback, e.Detail)
}
