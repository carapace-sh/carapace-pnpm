package pnpm

// Span is a byte-offset range within the input string.
type Span struct {
	Start int
	End   int
}

// Pos is a source position used for error reporting.
type Pos struct {
	Offset int
	Line   int
	Column int
}
