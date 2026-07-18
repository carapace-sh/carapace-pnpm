package pnpm

// ExpectedToken represents a type of token expected at a completion position.
type ExpectedToken int

const (
	// ExpectedSelector means a base selector (package name, path, or ".") is expected.
	ExpectedSelector ExpectedToken = iota
	// ExpectedComma means a "," separating selectors is expected.
	ExpectedComma
	// ExpectedRelation means a relational suffix ("..." or "^...") is expected.
	ExpectedRelation
	// ExpectedRelationOrEnd means either a relational suffix or the end of the
	// selector (comma or end of input) is expected.
	ExpectedRelationOrEnd
)

func (t ExpectedToken) String() string {
	switch t {
	case ExpectedSelector:
		return "Selector"
	case ExpectedComma:
		return ","
	case ExpectedRelation:
		return "Relation"
	case ExpectedRelationOrEnd:
		return "RelationOrEnd"
	}
	return "Unknown"
}

// MarshalText renders ExpectedToken as its name for JSON output.
func (t ExpectedToken) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ValidRelation is a relational suffix that could be valid at a completion position.
type ValidRelation struct {
	Op          string `json:"op"`
	Description string `json:"description"`
}

// SelectorContext describes a selector being completed.
type SelectorContext struct {
	// Negated is true when the selector started with "!".
	Negated bool `json:"negated"`
	// PartialBase is the partial base text being typed (e.g. "rea" in "react").
	PartialBase string `json:"partialBase,omitempty"`
	// Kind is the classified kind of the partial base, if determinable.
	Kind SelectorKind `json:"kind,omitempty"`
	// HasRelation is true when a relational suffix was already consumed.
	HasRelation bool `json:"hasRelation"`
	// Relation is the already-consumed relation kind, if any.
	Relation RelKind `json:"relation,omitempty"`
}

// CompletionContext describes what is expected at the completion position.
type CompletionContext struct {
	ExpectedTokens []ExpectedToken `json:"expectedTokens"`

	// ValidRelations lists the relational suffixes valid at this position.
	ValidRelations []ValidRelation `json:"validRelations,omitempty"`

	// Selector is non-nil when the cursor is inside a selector (base or suffix).
	Selector *SelectorContext `json:"selector,omitempty"`

	// AtNewSelector is true when the cursor is at the start of a new selector
	// (after a comma or at the very start).
	AtNewSelector bool `json:"atNewSelector"`

	// PartialNegation is true when the cursor is right after a "!" with no base yet.
	PartialNegation bool `json:"partialNegation"`
}
