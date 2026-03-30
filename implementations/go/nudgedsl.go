// Package nudgedsl implements the nudgeDSL v0.1.0 parser, validator, and registry.
// See: nudgeDSL-spec-v0.1.md
package nudgedsl

// ParseAndValidate is the main pipeline entry point.
// It parses the input string and, if a registry is provided, validates the AST.
// Returns the AST, parse errors, and validation errors separately.
func ParseAndValidate(input string, registry *Registry) (*AST, *ParseError, []ValidationError) {
	ast, parseErr := Parse(input, registry)
	if parseErr != nil {
		return nil, parseErr, nil
	}
	if registry == nil {
		return ast, nil, nil
	}
	v := &DefaultValidator{}
	valErrs := v.Validate(ast, registry)
	return ast, nil, valErrs
}
