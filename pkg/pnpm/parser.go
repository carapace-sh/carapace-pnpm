package pnpm

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseError is a syntax error with a span indicating the problematic region.
type ParseError struct {
	Message string
	Span    Span
}

func (e *ParseError) Error() string {
	return e.Message
}

// parser is the internal recursive-descent parser state.
type parser struct {
	input string
	pos   int
}

// Parse parses a pnpm filter selector string into a Filter AST.
//
// Grammar (mirrors pnpm's parse_project_selector):
//
//	filter    = selector ( "," selector )*
//	selector  = "!"? prefix? base suffix?
//	prefix    = "..." "^"?            (dependents; optional excludeSelf)
//	suffix    = "^"? "..."            (dependencies; optional excludeSelf)
//	base      = name? brace? diff?
//	          | location
//	name      = [^.] [^{}[\]]*        (package-name glob; may start with '@')
//	brace     = "{" [^}]+ "}"         (directory selector; inner resolved vs prefix)
//	diff      = "[" [^\]]+ "]"        (changed-packages selector; git ref)
//	location  = "." | ".." | "./"… | "../"…
//
// The relational modifiers are orthogonal: a selector may have a prefix, a
// suffix, both, or neither, and "^" may appear adjacent to any "..." to
// exclude the matched package itself. Whitespace is allowed around commas but
// not within a selector.
func Parse(input string) (*Filter, error) {
	p := &parser{input: input}
	p.skipWhitespace()
	start := p.pos
	f, err := p.parseFilter()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, p.syntaxErrorf("unexpected token %q", p.remaining())
	}
	f.Span = Span{Start: start, End: p.pos}
	return f, nil
}

func (p *parser) parseFilter() (*Filter, error) {
	f := &Filter{}
	for {
		p.skipWhitespace()
		if p.atEnd() {
			if len(f.Selectors) == 0 {
				return nil, p.syntaxError("expected a selector")
			}
			break
		}
		sel, err := p.parseSelector()
		if err != nil {
			return nil, err
		}
		f.Selectors = append(f.Selectors, sel)
		p.skipWhitespace()
		if p.atEnd() {
			break
		}
		if p.peek() == ',' {
			p.advance()
			p.skipWhitespace()
			if p.atEnd() {
				return nil, p.syntaxError("expected a selector after ','")
			}
			continue
		}
		return nil, p.syntaxErrorf("expected ',' or end of input, got %q", p.remaining())
	}
	return f, nil
}

func (p *parser) parseSelector() (Selector, error) {
	selStart := p.pos
	sel := Selector{}
	p.skipWhitespace()

	// Negation.
	if p.peek() == '!' {
		negStart := p.pos
		p.advance()
		sel.Negated = true
		sel.NegateSpan = &Span{Start: negStart, End: p.pos}
		if p.atEnd() {
			return sel, p.syntaxError("'!' must be followed by a selector")
		}
	}

	// We need to look at the whole remaining selector text to strip the
	// prefix/suffix relational modifiers. Find the end of this selector
	// (the next top-level comma or end of input, respecting brace depth).
	rest, end := p.selectorRest()
	if rest == "" {
		if sel.Negated {
			return sel, p.syntaxError("expected a selector after '!'")
		}
		return sel, p.syntaxError("expected a selector")
	}

	parsed, err := parseSelectorBody(rest)
	if err != nil {
		return sel, err
	}
	// Merge: carry over negation, offset spans by p.pos.
	sel.Kind = parsed.kind
	sel.IncludeDependents = parsed.includeDependents
	sel.IncludeDependencies = parsed.includeDependencies
	sel.ExcludeSelf = parsed.excludeSelf
	offset := p.pos
	if parsed.name != "" {
		sel.Name = parsed.name
		sel.NameSpan = offsetSpan(parsed.nameSpan, offset)
	}
	if parsed.braceInner != "" {
		sel.BraceInner = parsed.braceInner
		sel.BraceSpan = offsetSpan(parsed.braceSpan, offset)
	}
	if parsed.diff != "" {
		sel.Diff = parsed.diff
		sel.DiffSpan = offsetSpan(parsed.diffSpan, offset)
	}
	if parsed.path != "" {
		sel.Path = parsed.path
		sel.PathSpan = offsetSpan(parsed.pathSpan, offset)
	}
	if parsed.prefixSpan != nil {
		sel.PrefixSpan = offsetSpan(parsed.prefixSpan, offset)
	}
	if parsed.suffixSpan != nil {
		sel.SuffixSpan = offsetSpan(parsed.suffixSpan, offset)
	}

	p.pos = end
	sel.Span = Span{Start: selStart, End: p.pos}
	return sel, nil
}

// selectorRest returns the text of the current selector (up to the next
// top-level comma or end of input) and the absolute position just past it.
// Brace depth is tracked so commas inside {...} do not terminate the selector.
func (p *parser) selectorRest() (string, int) {
	start := p.pos
	depth := 0
	for p.pos < len(p.input) {
		r, w := utf8.DecodeRuneInString(p.input[p.pos:])
		if r == ',' && depth == 0 {
			break
		}
		if isWhitespace(r) && depth == 0 {
			break
		}
		if r == '{' {
			depth++
		} else if r == '}' && depth > 0 {
			depth--
		} else if r == '[' && depth == 0 {
			// brackets are not nested; track so a comma inside [...] (rare)
			// does not terminate. We only see [diff] at the end though.
			depth++
		} else if r == ']' && depth > 0 {
			depth--
		}
		p.pos += w
	}
	return p.input[start:p.pos], p.pos
}

// --- selector body parser (operates on a single selector's text) ---

type selectorBody struct {
	kind SelectorKind

	name       string
	nameSpan   *Span
	braceInner string
	braceSpan  *Span
	diff       string
	diffSpan   *Span
	path       string
	pathSpan   *Span

	includeDependents   bool
	includeDependencies bool
	excludeSelf         bool
	prefixSpan          *Span
	suffixSpan          *Span
}

func parseSelectorBody(raw string) (selectorBody, error) {
	var b selectorBody
	s := raw

	// Strip negation is handled by the caller; here s has no leading "!".

	// Suffix: trailing "..." with an optional preceding "^".
	if strings.HasSuffix(s, "...") {
		b.includeDependencies = true
		s = s[:len(s)-3]
		start := len(raw) - 3
		b.suffixSpan = &Span{Start: start, End: start + 3}
		if strings.HasSuffix(s, "^") {
			b.excludeSelf = true
			s = s[:len(s)-1]
		}
	}

	// Prefix: leading "..." with an optional following "^".
	if strings.HasPrefix(s, "...") {
		b.includeDependents = true
		s = s[3:]
		b.prefixSpan = &Span{Start: 0, End: 3}
		if strings.HasPrefix(s, "^") {
			b.excludeSelf = true
			s = s[1:]
		}
	}

	// Now s is the base: name? brace? diff?  OR a location.
	if err := parseBase(s, &b, len(raw)-len(s)); err != nil {
		return b, err
	}
	return b, nil
}

// parseBase parses the base portion of a selector (after relations stripped).
// offset is the byte offset of s within the original selector text, used to
// compute absolute spans.
func parseBase(s string, b *selectorBody, offset int) error {
	if s == "" {
		return &ParseError{Message: "expected a selector base", Span: Span{Start: offset, End: offset + 1}}
	}

	// Location forms: ".", "..", "./...", "../...", or a bare path.
	if isLocation(s) {
		b.kind = kindForLocation(s)
		b.path = s
		b.pathSpan = &Span{Start: offset, End: offset + len(s)}
		return nil
	}

	// name? brace? diff?
	// The name is a greedy leading run of characters that are not '{' or '['.
	// A name may not start with '.' (that would be a location). Package-name
	// characters are letters, digits, '_', '-', '@', '/', '.', '*', '?', and
	// glob chars — but NOT '!', whitespace, or commas.
	rest := s
	nameEnd := 0
	if len(s) > 0 && s[0] != '.' && s[0] != '{' && s[0] != '[' {
		i := strings.IndexAny(s, "{[")
		if i < 0 {
			nameEnd = len(s)
		} else {
			nameEnd = i
		}
	}
	if nameEnd > 0 {
		name := s[:nameEnd]
		if !isValidName(name) {
			return &ParseError{Message: fmt.Sprintf("invalid package name %q", name), Span: Span{Start: offset, End: offset + nameEnd}}
		}
		b.name = name
		b.nameSpan = &Span{Start: offset, End: offset + nameEnd}
		rest = s[nameEnd:]
		b.kind = KindPackageName
	}

	// Brace: {inner}
	if strings.HasPrefix(rest, "{") {
		closeIdx := strings.IndexByte(rest, '}')
		if closeIdx < 1 {
			return &ParseError{Message: "unterminated '{' in selector", Span: Span{Start: offset + nameEnd, End: offset + nameEnd + 1}}
		}
		b.braceInner = rest[1:closeIdx]
		b.braceSpan = &Span{Start: offset + nameEnd, End: offset + nameEnd + closeIdx + 1}
		rest = rest[closeIdx+1:]
		if b.kind == KindUnset {
			b.kind = KindBrace
		}
	}

	// Diff: [ref]
	if strings.HasPrefix(rest, "[") {
		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx < 1 {
			braceLen := lenBrace(b)
			return &ParseError{Message: "unterminated '[' in selector", Span: Span{Start: offset + nameEnd + braceLen, End: offset + nameEnd + braceLen + 1}}
		}
		b.diff = rest[1:closeIdx]
		diffStart := offset + nameEnd + lenBrace(b)
		b.diffSpan = &Span{Start: diffStart, End: diffStart + closeIdx + 1}
		rest = rest[closeIdx+1:]
		if b.kind == KindUnset {
			b.kind = KindDiff
		}
	}

	if rest != "" {
		return &ParseError{Message: fmt.Sprintf("unexpected trailing %q in selector", rest), Span: Span{Start: offset + len(s) - len(rest), End: offset + len(s)}}
	}

	return nil
}

// isValidName reports whether s is a valid package-name/glob token: no '!',
// no whitespace, no commas, no unbalanced braces/brackets. The characters
// allowed are those that can appear in an npm package name or a glob pattern.
func isValidName(s string) bool {
	if s == "" {
		return false
	}
	depth := 0
	for _, r := range s {
		switch r {
		case '!':
			// '!' is only allowed inside {...} for glob exclusion.
			if depth == 0 {
				return false
			}
		case ',', ' ', '\t', '\r', '\n':
			if depth == 0 {
				return false
			}
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return true
}

func lenBrace(b *selectorBody) int {
	if b.braceSpan != nil {
		return b.braceSpan.End - b.braceSpan.Start
	}
	return 0
}

// isLocation reports whether s is a relative-path location selector.
func isLocation(s string) bool {
	if s == "." || s == ".." {
		return true
	}
	return strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, ".\\") || strings.HasPrefix(s, "..\\")
}

func kindForLocation(s string) SelectorKind {
	switch s {
	case ".":
		return KindSelf
	case "..":
		return KindParent
	}
	return KindPath
}

func offsetSpan(s *Span, offset int) *Span {
	if s == nil {
		return nil
	}
	return &Span{Start: s.Start + offset, End: s.End + offset}
}

// --- parser helpers ---

func (p *parser) peek() rune {
	if p.pos >= len(p.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *parser) advance() rune {
	if p.pos >= len(p.input) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += w
	return r
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.input)
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) {
		r, w := utf8.DecodeRuneInString(p.input[p.pos:])
		if !isWhitespace(r) {
			break
		}
		p.pos += w
	}
}

func (p *parser) remaining() string {
	if p.pos >= len(p.input) {
		return ""
	}
	r := []rune(p.input[p.pos:])
	if len(r) > 12 {
		r = append(r[:12], []rune("...")...)
	}
	return string(r)
}

func (p *parser) syntaxError(msg string) *ParseError {
	return &ParseError{
		Message: msg,
		Span:    Span{Start: p.pos, End: min(p.pos+1, len(p.input))},
	}
}

func (p *parser) syntaxErrorf(format string, args ...any) *ParseError {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Span:    Span{Start: p.pos, End: min(p.pos+1, len(p.input))},
	}
}
