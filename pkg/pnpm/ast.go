package pnpm

import "encoding/json"

// SelectorKind identifies the kind of base selector.
type SelectorKind int

const (
	// KindUnset is the zero value, indicating the kind has not been determined.
	// It is not a valid final kind.
	KindUnset SelectorKind = iota
	// KindPackageName is a selector by package name, e.g. "foo" or "@scope/bar".
	KindPackageName
	// KindPath is a filesystem-path selector, e.g. "./packages/*" or "../shared".
	KindPath
	// KindSelf is the "." selector referring to the package in the current directory.
	KindSelf
	// KindParent is the ".." selector referring to the parent directory.
	KindParent
	// KindBrace is a "{...}" directory selector whose inner text is resolved
	// against the workspace prefix, e.g. "{foo}" or "{./apps/*}".
	KindBrace
	// KindDiff is a "[ref]" changed-packages selector with no base.
	KindDiff
)

func (k SelectorKind) String() string {
	switch k {
	case KindUnset:
		return "Unset"
	case KindPackageName:
		return "PackageName"
	case KindPath:
		return "Path"
	case KindSelf:
		return "Self"
	case KindParent:
		return "Parent"
	case KindBrace:
		return "Brace"
	case KindDiff:
		return "Diff"
	}
	return "Unknown"
}

// MarshalJSON renders SelectorKind as its name.
func (k SelectorKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Selector is a single filter selector. It mirrors pnpm's
// parse_project_selector: an optional negation, optional prefix "..." and "^",
// a base (name? brace? diff?), and an optional suffix "^" and "...".
//
// The relational and self-exclusion modifiers are orthogonal booleans, not a
// single enum — pnpm allows any combination:
//
//   - "foo"            → none
//   - "foo..."         → IncludeDependencies
//   - "foo^..."        → IncludeDependencies + ExcludeSelf
//   - "...foo"         → IncludeDependents
//   - "...^foo"        → IncludeDependents + ExcludeSelf
//   - "...foo..."      → IncludeDependents + IncludeDependencies
//   - "...^foo^..."    → IncludeDependents + IncludeDependencies + ExcludeSelf
type Selector struct {
	Span Span `json:"span"`

	// Negated is set by a leading "!" — the matched projects are subtracted
	// from the selection rather than added.
	Negated    bool  `json:"negated"`
	NegateSpan *Span `json:"negateSpan,omitempty"`

	// Kind classifies the base. KindDiff means the selector is a bare "[ref]"
	// with no name/brace/path.
	Kind SelectorKind `json:"kind"`

	// Name is the package-name glob when Kind is KindPackageName, or the name
	// prefix of a "name{brace}" form. Empty for pure path/brace/diff selectors.
	Name     string `json:"name,omitempty"`
	NameSpan *Span  `json:"nameSpan,omitempty"`

	// BraceInner is the inner text of a "{...}" directory selector, resolved
	// against the workspace prefix. Set for KindBrace and for the "name{brace}"
	// combination. Empty when no braces are present.
	BraceInner string `json:"braceInner,omitempty"`
	BraceSpan  *Span  `json:"braceSpan,omitempty"`

	// Diff is the inner text of a "[ref]" changed-packages selector — a git
	// ref (branch, tag, commit) whose diff selects packages.
	Diff     string `json:"diff,omitempty"`
	DiffSpan *Span  `json:"diffSpan,omitempty"`

	// Path is the raw location text for a KindPath selector (e.g. "./packages/*",
	// "../shared"). Empty for name/brace/diff selectors.
	Path     string `json:"path,omitempty"`
	PathSpan *Span  `json:"pathSpan,omitempty"`

	// Relational modifiers (orthogonal):
	//   - IncludeDependents   — leading "..."  (select packages that depend on the match)
	//   - IncludeDependencies — trailing "..." (select packages the match depends on)
	//   - ExcludeSelf         — a "^" adjacent to a "..." (exclude the matched package itself)
	IncludeDependents   bool `json:"includeDependents"`
	IncludeDependencies bool `json:"includeDependencies"`
	ExcludeSelf         bool `json:"excludeSelf"`

	// Span of the prefix "..." (when IncludeDependents).
	PrefixSpan *Span `json:"prefixSpan,omitempty"`
	// Span of the suffix "..." (when IncludeDependencies).
	SuffixSpan *Span `json:"suffixSpan,omitempty"`
}

// Filter is the top-level AST node: a comma-separated list of selectors.
type Filter struct {
	Span      Span       `json:"span"`
	Selectors []Selector `json:"selectors"`
}
