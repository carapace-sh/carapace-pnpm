package pnpm

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\x0c'
}

// isPkgNameStart reports whether r can begin a bare package-name selector.
// pnpm package names are npm package names: letters, digits, '_', '-', '@',
// '.', '/', '{', '}', '*', '*', '?', and '[' for glob characters.
func isPkgNameStart(r rune) bool {
	if r == '@' || r == '.' || r == '/' || r == '{' || r == '*' || r == '?' || r == '[' {
		return true
	}
	return isIdentStart(r)
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-'
}

// isPkgNamePart reports whether r can continue a package-name/glob token.
// '!' is excluded: it is the negation operator and only valid at the start of
// a selector. npm package names never contain '!'.
func isPkgNamePart(r rune) bool {
	if r == '/' || r == '.' || r == '*' || r == '?' || r == '{' || r == '}' || r == '[' || r == ']' || r == '@' {
		return true
	}
	return isIdentPart(r)
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}

// scanSelectorBase scans a base selector token starting at the current position.
// The base is the longest run of selector-base characters, stopping at a
// top-level ',' (one not inside {}), relational suffixes ('...' or '^...'),
// whitespace, or end of input. Commas inside {...} are part of the glob and
// do not terminate the base.
// Returns the token text and true if a non-empty token was scanned.
func (p *parser) scanSelectorBase() (string, bool) {
	if p.atEnd() || !isPkgNameStart(p.peek()) {
		return "", false
	}
	start := p.pos
	braceDepth := 0
	for !p.atEnd() {
		r := p.peek()
		if r == ',' && braceDepth == 0 {
			break
		}
		if isWhitespace(r) {
			break
		}
		if r == '{' {
			braceDepth++
		} else if r == '}' && braceDepth > 0 {
			braceDepth--
		}
		// Stop at a relational suffix: '...' or '^...' attached to the base.
		// '...' as a suffix starts at a '.' that is followed by two more dots
		// and then a word boundary. We must not consume it as part of the base.
		if r == '.' && braceDepth == 0 && p.matchString("...") {
			// Only treat as suffix if it's at the end of the base token.
			// A leading './' or '../' is part of a path, not a suffix.
			// Distinguish: if the '...' is preceded by at least one base char
			// and followed by ','/whitespace/EOF, it's a suffix.
			if p.pos > start {
				end := p.pos + 3
				if end >= len(p.input) || p.input[end] == ',' || isWhitespace(rune(p.input[end])) {
					break
				}
			}
		}
		if r == '^' && braceDepth == 0 && p.matchString("^...") {
			break
		}
		if !isPkgNamePart(r) {
			// ',' is allowed inside {...} as a glob separator
			// (e.g. {./apps/*,./packages/*}).
			// '!' is allowed inside {...} for glob exclusion patterns
			// (e.g. {./apps/*,!./apps/legacy}).
			if !((r == ',' || r == '!') && braceDepth > 0) {
				break
			}
		}
		p.advance()
	}
	if p.pos == start {
		return "", false
	}
	return p.input[start:p.pos], true
}

// scanRelationPrefix scans a leading "..." that marks a dependencies-prefix
// selector (e.g. "...foo"). Returns true if such a prefix was consumed.
func (p *parser) scanRelationPrefix() (bool, Span) {
	if p.matchString("...") {
		// Only treat as prefix if followed by a base-start character (not a
		// comma, whitespace, or EOF). A bare "..." is not a valid selector.
		end := p.pos + 3
		if end < len(p.input) && isPkgNameStart(rune(p.input[end])) {
			start := p.pos
			p.pos += 3
			return true, Span{Start: start, End: p.pos}
		}
	}
	return false, Span{}
}

// scanRelationalSuffix scans a '...' or '^...' suffix at the current position.
// Returns the kind and the span; RelNone with a nil span if no suffix is present.
func (p *parser) scanRelationalSuffix() (RelKind, Span) {
	if p.matchString("^...") {
		start := p.pos
		p.pos += 4
		return RelDirectDependencies, Span{Start: start, End: p.pos}
	}
	if p.matchString("...") {
		start := p.pos
		p.pos += 3
		return RelDependents, Span{Start: start, End: p.pos}
	}
	return RelNone, Span{}
}

// --- cursor-bounded scanner (completion parser) ---

// scanSelectorBaseCursor is the cursor-bounded version of scanSelectorBase.
func (p *compParser) scanSelectorBaseCursor() (string, bool) {
	if p.atCursorOrEnd() || !isPkgNameStart(p.peek()) {
		return "", false
	}
	start := p.pos
	braceDepth := 0
	for !p.atCursorOrEnd() {
		r := p.peek()
		if r == ',' && braceDepth == 0 {
			break
		}
		if isWhitespace(r) {
			break
		}
		if r == '{' {
			braceDepth++
		} else if r == '}' && braceDepth > 0 {
			braceDepth--
		}
		if r == '.' && braceDepth == 0 && p.matchString("...") {
			if p.pos > start {
				end := p.pos + 3
				if end >= p.cursor || end >= len(p.input) || p.input[end] == ',' || isWhitespace(rune(p.input[end])) {
					break
				}
			}
		}
		if r == '^' && braceDepth == 0 && p.matchString("^...") {
			break
		}
		if !isPkgNamePart(r) {
			if !((r == ',' || r == '!') && braceDepth > 0) {
				break
			}
		}
		p.advance()
	}
	if p.pos == start {
		return "", false
	}
	return p.input[start:p.pos], true
}

// scanRelationPrefixCursor is the cursor-bounded version of scanRelationPrefix.
func (p *compParser) scanRelationPrefixCursor() (bool, Span) {
	if p.matchString("...") {
		end := p.pos + 3
		if end < p.cursor && end < len(p.input) && isPkgNameStart(rune(p.input[end])) {
			start := p.pos
			for i := 0; i < 3 && !p.atCursorOrEnd(); i++ {
				p.advance()
			}
			return true, Span{Start: start, End: p.pos}
		}
	}
	return false, Span{}
}

func (p *compParser) scanRelationalSuffixCursor() (RelKind, Span) {
	if p.matchString("^...") {
		start := p.pos
		// advance up to cursor
		for i := 0; i < 4 && !p.atCursorOrEnd(); i++ {
			p.advance()
		}
		return RelDirectDependencies, Span{Start: start, End: p.pos}
	}
	if p.matchString("...") {
		start := p.pos
		for i := 0; i < 3 && !p.atCursorOrEnd(); i++ {
			p.advance()
		}
		return RelDependents, Span{Start: start, End: p.pos}
	}
	return RelNone, Span{}
}
