package nudgedsl

import "fmt"

// ArgType is the declared type of an atom argument.
type ArgType string

const (
	ArgString  ArgType = "string"
	ArgInteger ArgType = "integer"
	ArgFloat   ArgType = "float"
	ArgBoolean ArgType = "boolean"
)

// ArgDef declares a single argument for an atom.
type ArgDef struct {
	Name     string      `json:"name"`
	Type     ArgType     `json:"type"`
	Min      *float64    `json:"min,omitempty"`
	Max      *float64    `json:"max,omitempty"`
	Enum     []string    `json:"enum,omitempty"`
	Required *bool       `json:"required,omitempty"` // defaults to true
}

// IsRequired returns true unless explicitly set to false.
func (a *ArgDef) IsRequired() bool {
	return a.Required == nil || *a.Required
}

// AtomDef is a single registered atom.
type AtomDef struct {
	Atom     string   `json:"atom"`
	Fn       string   `json:"fn"`
	Desc     string   `json:"description,omitempty"`
	Args     []ArgDef `json:"args"`
	Rollback string   `json:"rollback,omitempty"` // atom code of the rollback, or ""
}

// Registry holds all registered atoms and validates them at load time.
type Registry struct {
	atoms map[string]*AtomDef
}

// NewRegistry builds and validates a registry from a slice of AtomDefs.
// Returns a RegistryError if any rollback declaration is invalid.
func NewRegistry(defs []AtomDef) (*Registry, error) {
	r := &Registry{atoms: make(map[string]*AtomDef, len(defs))}
	for i := range defs {
		r.atoms[defs[i].Atom] = &defs[i]
	}
	// Validate rollback signatures at load time (spec §7)
	for _, def := range r.atoms {
		if def.Rollback == "" {
			continue
		}
		rb, ok := r.atoms[def.Rollback]
		if !ok {
			return nil, &RegistryError{
				Code:     ErrRollbackNotFound,
				Atom:     def.Atom,
				Rollback: def.Rollback,
				Detail:   fmt.Sprintf("rollback atom %q not found in registry", def.Rollback),
			}
		}
		if len(rb.Args) != len(def.Args) {
			return nil, &RegistryError{
				Code:     ErrRollbackSignatureMismatch,
				Atom:     def.Atom,
				Rollback: def.Rollback,
				Detail: fmt.Sprintf("%s declares %d arg(s), expected %d to match %s",
					def.Rollback, len(rb.Args), len(def.Args), def.Atom),
			}
		}
		for i, arg := range def.Args {
			if rb.Args[i].Type != arg.Type {
				return nil, &RegistryError{
					Code:     ErrRollbackSignatureMismatch,
					Atom:     def.Atom,
					Rollback: def.Rollback,
					Detail: fmt.Sprintf("arg[%d] type mismatch: %s has %q, %s has %q",
						i, def.Rollback, rb.Args[i].Type, def.Atom, arg.Type),
				}
			}
		}
	}
	return r, nil
}

// Lookup returns the AtomDef for the given atom code, or nil.
func (r *Registry) Lookup(atom string) *AtomDef {
	if r == nil {
		return nil
	}
	return r.atoms[atom]
}

// Has returns true if the atom is registered.
func (r *Registry) Has(atom string) bool {
	return r.Lookup(atom) != nil
}
