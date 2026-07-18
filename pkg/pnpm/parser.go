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
// Grammar:
//
//	filter    = selector ( "," selector )*
//	selector  = [ "!" ] base [ relation ]
//	base      = packageName | pathGlob | "." | ".."
//	relation  = "..." | "^..."
//
// Whitespace around commas and between tokens is skipped. A selector base may
// not contain whitespace or commas. Relational suffixes attach directly to the
// base with no intervening whitespace.
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
			// A trailing comma with nothing after it is not a valid selector.
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
		p.skipWhitespace()
		if p.atEnd() {
			return sel, p.syntaxError("'!' must be followed by a selector")
		}
	}

	// Prefix relation: "...pkg" selects the package and its dependencies.
	if ok, relSpan := p.scanRelationPrefix(); ok {
		sel.Relation = RelDependencies
		sel.RelSpan = &relSpan
	}

	// Base.
	base, ok := p.scanSelectorBase()
	if !ok {
		if sel.Negated {
			return sel, p.syntaxError("expected a selector after '!'")
		}
		return sel, p.syntaxErrorf("expected a selector, got %q", p.remaining())
	}
	sel.Base = base
	sel.Kind = classifyBase(base)
	baseEnd := p.pos

	// Relational suffix (no whitespace between base and suffix).
	rel, relSpan := p.scanRelationalSuffix()
	if rel != RelNone {
		sel.Relation = rel
		sel.RelSpan = &relSpan
	}

	sel.Span = Span{Start: selStart, End: p.pos}
	if sel.Span.End < baseEnd {
		sel.Span.End = baseEnd
	}
	return sel, nil
}

// classifyBase determines the SelectorKind from the base text.
func classifyBase(base string) SelectorKind {
	switch base {
	case ".":
		return KindSelf
	case "..":
		// ".." alone is a path-like base; treat as self for completion purposes
		// since pnpm interprets it relative to the current dir.
		return KindSelf
	}
	if strings.HasPrefix(base, "./") || strings.HasPrefix(base, "../") || strings.HasPrefix(base, "/") ||
		strings.HasPrefix(base, "{") || strings.ContainsAny(base, "*?[") {
		return KindPathGlob
	}
	// A lone "..." or "^..." (rare, but pnpm treats "." + "...") selects the
	// whole graph; classify as All.
	if base == "" && (strings.HasPrefix(base, "...") || strings.HasPrefix(base, "^...")) {
		return KindAll
	}
	return KindPackageName
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

func (p *parser) matchString(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}
	return p.input[p.pos:p.pos+len(s)] == s
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
