package nudgedsl

import "encoding/json"

// NodeType identifies the kind of AST node.
type NodeType string

const (
	NodeCall     NodeType = "call"
	NodeChain    NodeType = "chain"
	NodeParallel NodeType = "parallel"
	NodeFallback NodeType = "fallback"
	NodeAmplify  NodeType = "amplify"
)

// FailureMode controls parallel branch error handling.
type FailureMode string

const (
	FailFast     FailureMode = "fail-fast"
	BestEffort   FailureMode = "best-effort"
	Compensating FailureMode = "compensating"
)

// Node is a single AST node. Fields are populated based on NodeType.
type Node struct {
	Type NodeType `json:"type"`

	// NodeCall fields
	Atom string        `json:"atom,omitempty"`
	Fn   string        `json:"fn,omitempty"`
	Args []interface{} `json:"args,omitempty"`

	// NodeChain / NodeFallback fields
	Nodes []*Node `json:"nodes,omitempty"`

	// NodeParallel fields
	FailureMode FailureMode `json:"failure_mode,omitempty"`

	// NodeAmplify fields
	Node  *Node `json:"node,omitempty"`
	Count int   `json:"count,omitempty"`
}

// AST is the top-level output of a successful parse.
type AST struct {
	Version string `json:"version"`
	Root    *Node  `json:"root"`
}

// JSON returns the AST serialized to indented JSON.
func (a *AST) JSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}
