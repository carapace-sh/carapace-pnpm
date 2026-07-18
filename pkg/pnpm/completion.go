package pnpm

// ExpectedToken represents a type of token expected at a completion position.
type ExpectedToken int

const (
	// ExpectedSelector means a base selector (name, path, ".", "[ref]", "{...}") is expected.
	ExpectedSelector ExpectedToken = iota
	// ExpectedComma means a "," separating selectors is expected.
	ExpectedComma
	// ExpectedRelationOrEnd means either a relational modifier ("..." or "^...")
	// or the end of the selector (comma or end of input) is expected.
	ExpectedRelationOrEnd
)

func (t ExpectedToken) String() string {
	switch t {
	case ExpectedSelector:
		return "Selector"
	case ExpectedComma:
		return ","
	case ExpectedRelationOrEnd:
		return "RelationOrEnd"
	}
	return "Unknown"
}

// MarshalText renders ExpectedToken as its name for JSON output.
func (t ExpectedToken) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ValidRelation is a relational suffix/prefix that could be valid at a completion position.
type ValidRelation struct {
	Op          string `json:"op"`
	Description string `json:"description"`
}

// SelectorContext describes a selector being completed.
type SelectorContext struct {
	// Negated is true when the selector started with "!".
	Negated bool `json:"negated"`

	// PartialBase is the partial base text being typed (the portion of the
	// selector after any !/.../^  prefix and before any trailing .../^ suffix).
	PartialBase string `json:"partialBase,omitempty"`

	// Kind is the classified kind of the partial base, if determinable.
	Kind SelectorKind `json:"kind,omitempty"`

	// IncludeDependents is true when a leading "..." was consumed.
	IncludeDependents bool `json:"includeDependents"`
	// IncludeDependencies is true when a trailing "..." was consumed.
	IncludeDependencies bool `json:"includeDependencies"`
	// ExcludeSelf is true when a "^" adjacent to a "..." was consumed.
	ExcludeSelf bool `json:"excludeSelf"`

	// HasRelation is true when any relational modifier was consumed.
	HasRelation bool `json:"hasRelation"`
}

// CompletionContext describes what is expected at the completion position.
type CompletionContext struct {
	ExpectedTokens []ExpectedToken `json:"expectedTokens"`

	// ValidRelations lists the relational modifiers valid at this position.
	ValidRelations []ValidRelation `json:"validRelations,omitempty"`

	// Selector is non-nil when the cursor is inside a selector (base or suffix).
	Selector *SelectorContext `json:"selector,omitempty"`

	// AtNewSelector is true when the cursor is at the start of a new selector
	// (after a comma or at the very start).
	AtNewSelector bool `json:"atNewSelector"`

	// PartialNegation is true when the cursor is right after a "!" with no base yet.
	PartialNegation bool `json:"partialNegation"`
}
