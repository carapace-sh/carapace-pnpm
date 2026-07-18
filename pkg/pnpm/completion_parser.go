package pnpm

import (
	"slices"
	"unicode/utf8"
)

// ParseForCompletion parses a partial pnpm filter selector string and returns
// a CompletionContext describing what is expected at the end of the input.
// Partial selectors are allowed — the parser recovers from incomplete input to
// report what tokens would be valid at the cursor position.
func ParseForCompletion(input string) *CompletionContext {
	cursor := len(input)
	p := &compParser{
		input:  input,
		pos:    0,
		cursor: cursor,
		ctx:    &CompletionContext{},
	}
	p.skipWS()
	p.parseFilter()
	if len(p.ctx.ExpectedTokens) == 0 {
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
	}
	p.ctx.ExpectedTokens = dedupTokens(p.ctx.ExpectedTokens)
	p.ctx.ValidRelations = dedupRelations(p.ctx.ValidRelations)
	return p.ctx
}

type compParser struct {
	input  string
	pos    int
	cursor int
	ctx    *CompletionContext

	// consumed is true when at least one selector was fully or partially parsed.
	consumed bool

	// inSelector tracks whether the cursor is inside a selector (vs. between them).
	inSelector bool
}

func (p *compParser) atCursorOrEnd() bool {
	return p.pos >= len(p.input) || p.pos >= p.cursor
}

func (p *compParser) peek() rune {
	if p.pos >= len(p.input) || p.pos >= p.cursor {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *compParser) advance() rune {
	if p.pos >= len(p.input) || p.pos >= p.cursor {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += w
	p.consumed = true
	return r
}

func (p *compParser) skipWS() {
	for p.pos < len(p.input) && p.pos < p.cursor {
		r, w := utf8.DecodeRuneInString(p.input[p.pos:])
		if !isWhitespace(r) {
			break
		}
		p.pos += w
	}
}

func (p *compParser) matchString(s string) bool {
	if p.pos+len(s) > len(p.input) || p.pos+len(s) > p.cursor {
		return false
	}
	return p.input[p.pos:p.pos+len(s)] == s
}

func (p *compParser) parseFilter() {
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			// At the cursor: decide what's expected.
			p.recordExpectation()
			return
		}
		p.parseSelector()
		p.skipWS()
		if p.atCursorOrEnd() {
			// Just finished a selector and landed on the cursor (after whitespace).
			p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedComma, ExpectedSelector)
			p.ctx.AtNewSelector = true
			return
		}
		if p.peek() == ',' {
			p.advance() // consume comma
			continue
		}
		// Unexpected token; record and stop.
		p.recordExpectation()
		return
	}
}

func (p *compParser) parseSelector() {
	selCtx := &SelectorContext{}
	p.inSelector = true
	defer func() {
		p.ctx.Selector = selCtx
	}()

	p.skipWS()
	if p.atCursorOrEnd() {
		// Cursor right at the start of a selector.
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
		p.ctx.AtNewSelector = true
		return
	}

	// Negation.
	if p.peek() == '!' {
		selCtx.Negated = true
		p.advance()
		p.skipWS()
		if p.atCursorOrEnd() {
			// Cursor right after "!" — expect a selector base.
			p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
			p.ctx.PartialNegation = true
			return
		}
	}

	// Prefix relation: "...pkg".
	if ok, _ := p.scanRelationPrefixCursor(); ok {
		selCtx.HasRelation = true
		selCtx.Relation = RelDependencies
	}

	// Base.
	base, ok := p.scanSelectorBaseCursor()
	if !ok {
		if p.atCursorOrEnd() {
			p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
			return
		}
		// Some other character; record selector expectation and stop.
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
		return
	}
	selCtx.PartialBase = base
	selCtx.Kind = classifyBase(base)
	p.consumed = true

	if p.atCursorOrEnd() {
		// Cursor right after the base — a relational suffix or end is valid.
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedRelationOrEnd)
		p.ctx.ValidRelations = append(p.ctx.ValidRelations,
			ValidRelation{Op: "...", Description: "the package and its dependents"},
			ValidRelation{Op: "^...", Description: "the package and its direct dependencies"},
		)
		return
	}

	// Relational suffix.
	rel, _ := p.scanRelationalSuffixCursor()
	if rel != RelNone {
		selCtx.HasRelation = true
		selCtx.Relation = rel
		p.consumed = true
	}
	if p.atCursorOrEnd() {
		// Cursor right after a (possibly partial) suffix — expect comma or end.
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedComma, ExpectedSelector)
		p.ctx.AtNewSelector = true
		return
	}
}

// recordExpectation records the appropriate expected token at the cursor.
func (p *compParser) recordExpectation() {
	if !p.consumed {
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedSelector)
		p.ctx.AtNewSelector = true
		return
	}
	// After at least one selector, the cursor can take a comma + new selector.
	p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedComma, ExpectedSelector)
	p.ctx.AtNewSelector = true
}

// --- dedup helpers ---

func dedupTokens(toks []ExpectedToken) []ExpectedToken {
	slices.Sort(toks)
	return slices.Compact(toks)
}

func dedupRelations(rels []ValidRelation) []ValidRelation {
	if len(rels) <= 1 {
		return rels
	}
	seen := map[string]bool{}
	out := rels[:0]
	for _, r := range rels {
		if seen[r.Op] {
			continue
		}
		seen[r.Op] = true
		out = append(out, r)
	}
	return out
}
