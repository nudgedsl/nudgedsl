package nudgedsl

import (
	"fmt"
	"math"
)

// Validator checks a parsed AST against the atom registry.
// Returns a slice of ValidationErrors — never panics.
type Validator interface {
	Validate(ast *AST, registry *Registry) []ValidationError
}

// DefaultValidator enforces all registry-declared constraints.
type DefaultValidator struct{}

// Validate walks the AST and checks every call node against the registry.
func (v *DefaultValidator) Validate(ast *AST, registry *Registry) []ValidationError {
	if ast == nil || registry == nil {
		return nil
	}
	var errs []ValidationError
	v.walkNode(ast.Root, registry, &errs)
	return errs
}

func (v *DefaultValidator) walkNode(node *Node, reg *Registry, errs *[]ValidationError) {
	if node == nil {
		return
	}
	switch node.Type {
	case NodeCall:
		v.validateCall(node, reg, errs)
	case NodeChain, NodeFallback, NodeParallel:
		for _, child := range node.Nodes {
			v.walkNode(child, reg, errs)
		}
	case NodeAmplify:
		v.walkNode(node.Node, reg, errs)
	}
}

func (v *DefaultValidator) validateCall(node *Node, reg *Registry, errs *[]ValidationError) {
	def := reg.Lookup(node.Atom)
	if def == nil {
		*errs = append(*errs, ValidationError{
			Code:    ErrUnknownAtomV,
			Atom:    node.Atom,
			Message: fmt.Sprintf("atom %q is not registered", node.Atom),
		})
		return
	}

	// Arg count
	if len(node.Args) != len(def.Args) {
		*errs = append(*errs, ValidationError{
			Code:    ErrArgCountMismatch,
			Atom:    node.Atom,
			Message: fmt.Sprintf("atom %q expects %d arg(s), got %d", node.Atom, len(def.Args), len(node.Args)),
		})
		return // no point checking types if count is wrong
	}

	// Arg types and constraints
	for i, argDef := range def.Args {
		val := node.Args[i]
		v.validateArg(node.Atom, argDef, val, i, errs)
	}
}

func (v *DefaultValidator) validateArg(atom string, def ArgDef, val interface{}, idx int, errs *[]ValidationError) {
	switch def.Type {
	case ArgString:
		s, ok := val.(string)
		if !ok {
			*errs = append(*errs, ValidationError{
				Code:       ErrArgTypeMismatch,
				Atom:       atom,
				Arg:        def.Name,
				Value:      val,
				Constraint: "type: string",
				Message:    fmt.Sprintf("arg %q (pos %d): expected string, got %T", def.Name, idx, val),
			})
			return
		}
		if len(def.Enum) > 0 && !containsStr(def.Enum, s) {
			*errs = append(*errs, ValidationError{
				Code:       ErrArgNotInEnum,
				Atom:       atom,
				Arg:        def.Name,
				Value:      s,
				Constraint: fmt.Sprintf("one of %v", def.Enum),
				Message:    fmt.Sprintf("arg %q: %q is not a valid value", def.Name, s),
			})
		}

	case ArgInteger:
		n, ok := toInt(val)
		if !ok {
			*errs = append(*errs, ValidationError{
				Code:       ErrArgTypeMismatch,
				Atom:       atom,
				Arg:        def.Name,
				Value:      val,
				Constraint: "type: integer",
				Message:    fmt.Sprintf("arg %q (pos %d): expected integer, got %T", def.Name, idx, val),
			})
			return
		}
		v.checkNumericRange(atom, def, float64(n), errs)

	case ArgFloat:
		f, ok := toFloat(val)
		if !ok {
			*errs = append(*errs, ValidationError{
				Code:       ErrArgTypeMismatch,
				Atom:       atom,
				Arg:        def.Name,
				Value:      val,
				Constraint: "type: float",
				Message:    fmt.Sprintf("arg %q (pos %d): expected float, got %T", def.Name, idx, val),
			})
			return
		}
		v.checkNumericRange(atom, def, f, errs)

	case ArgBoolean:
		if _, ok := val.(bool); !ok {
			*errs = append(*errs, ValidationError{
				Code:       ErrArgTypeMismatch,
				Atom:       atom,
				Arg:        def.Name,
				Value:      val,
				Constraint: "type: boolean",
				Message:    fmt.Sprintf("arg %q (pos %d): expected boolean, got %T", def.Name, idx, val),
			})
		}
	}
}

func (v *DefaultValidator) checkNumericRange(atom string, def ArgDef, val float64, errs *[]ValidationError) {
	if def.Min != nil && val < *def.Min {
		*errs = append(*errs, ValidationError{
			Code:       ErrArgOutOfRange,
			Atom:       atom,
			Arg:        def.Name,
			Value:      val,
			Constraint: fmt.Sprintf("min: %v", *def.Min),
			Message:    fmt.Sprintf("arg %q: %v is below minimum %v", def.Name, val, *def.Min),
		})
	}
	if def.Max != nil && val > *def.Max {
		*errs = append(*errs, ValidationError{
			Code:       ErrArgOutOfRange,
			Atom:       atom,
			Arg:        def.Name,
			Value:      val,
			Constraint: fmt.Sprintf("max: %v", *def.Max),
			Message:    fmt.Sprintf("arg %q: %v exceeds maximum %v", def.Name, val, *def.Max),
		})
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == math.Trunc(n) {
			return int(n), true
		}
		return 0, false
	}
	return 0, false
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
