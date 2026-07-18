package pnpm

import "encoding/json"

// SelectorKind identifies the kind of base selector.
type SelectorKind int

const (
	// KindPackageName is a selector by package name, e.g. "foo" or "@scope/bar".
	KindPackageName SelectorKind = iota
	// KindPathGlob is a filesystem-path selector, e.g. "./packages/*" or "{./apps/**}".
	KindPathGlob
	// KindSelf is the "." selector referring to the package in the current directory.
	KindSelf
	// KindAll is the "." selector with a dependents/dependencies suffix that
	// effectively selects from the whole workspace graph.
	KindAll
)

func (k SelectorKind) String() string {
	switch k {
	case KindPackageName:
		return "PackageName"
	case KindPathGlob:
		return "PathGlob"
	case KindSelf:
		return "Self"
	case KindAll:
		return "All"
	}
	return "Unknown"
}

// MarshalJSON renders SelectorKind as its name.
func (k SelectorKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// RelKind identifies the relational modifier of a selector.
type RelKind int

const (
	// RelNone means no relational modifier — the selector matches just the base.
	RelNone RelKind = iota
	// RelDependents is the "pkg..." suffix: the base package and all packages
	// that depend on it (directly or transitively).
	RelDependents
	// RelDependencies is the "...pkg" prefix: the base package and all of its
	// dependencies (directly or transitively).
	RelDependencies
	// RelDirectDependencies is the "pkg^..." suffix: the base package and its
	// direct dependencies only.
	RelDirectDependencies
)

func (k RelKind) String() string {
	switch k {
	case RelNone:
		return "None"
	case RelDependents:
		return "Dependents"
	case RelDependencies:
		return "Dependencies"
	case RelDirectDependencies:
		return "DirectDependencies"
	}
	return "Unknown"
}

// MarshalJSON renders RelKind as its name.
func (k RelKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Selector is a single filter selector: an optional negation, an optional
// prefix relation ("..."), a base, and an optional suffix relation ("..." or
// "^...").
type Selector struct {
	Span    Span         `json:"span"`
	Negated bool         `json:"negated"`
	Kind    SelectorKind `json:"kind"`
	Base    string       `json:"base"`
	// Relation is the relational modifier: a prefix ("...pkg" => RelDependencies)
	// or a suffix ("pkg..." => RelDependents, "pkg^..." => RelDirectDependencies).
	Relation   RelKind `json:"relation"`
	RelSpan    *Span   `json:"relSpan,omitempty"`
	NegateSpan *Span   `json:"negateSpan,omitempty"`
}

// Filter is the top-level AST node: a comma-separated list of selectors.
type Filter struct {
	Span      Span       `json:"span"`
	Selectors []Selector `json:"selectors"`
}
