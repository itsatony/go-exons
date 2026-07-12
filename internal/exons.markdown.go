package internal

import "strings"

// InertRegionKind classifies markdown regions the lexer treats as inert.
type InertRegionKind int

const (
	// InertRegionFence is a fenced code block (``` or ~~~).
	InertRegionFence InertRegionKind = iota
)

// InertRegion is a markdown region in which exons constructs are not
// recognized (MarkdownFences mode). Regions are non-overlapping and sorted by
// Start. Live regions (info string "exons") are recorded so the scanner does
// not misread fence-like lines inside them, but the lexer processes their
// contents normally.
type InertRegion struct {
	Kind       InertRegionKind
	Start      int    // byte offset of the opening fence line start
	BodyStart  int    // byte offset just after the opening fence line
	End        int    // byte offset after the closing fence line, or len(source)
	InfoString string // trimmed info string of the opening fence
	FenceChar  byte   // '`' or '~'
	FenceLen   int    // opening fence run length
	Live       bool   // first info-string word == MarkdownFenceInfoLive
	Unclosed   bool   // fence ran to end of input without a closer
	OpenPos    Position
}

// ScanMarkdownFences performs a single line-oriented pass over source and
// returns all fenced code blocks (live and inert) per a CommonMark subset:
// opener = up to MarkdownFenceMaxIndent spaces + a run of at least
// MarkdownFenceMinLen backticks or tildes + optional info string (backtick
// fences reject info strings containing a backtick); closer = same character,
// run length >= the opener's, nothing but whitespace after. An unclosed fence
// extends to the end of input and is flagged Unclosed. Indented code blocks
// and inline code spans are outside the subset.
func ScanMarkdownFences(source string) []InertRegion {
	var regions []InertRegion
	offset := 0
	lineNum := 1

	for offset < len(source) {
		line, next := readLine(source, offset)

		fenceChar, fenceLen, info, isOpen := parseFenceOpen(line)
		if !isOpen {
			offset = next
			lineNum++
			continue
		}

		region := InertRegion{
			Kind:       InertRegionFence,
			Start:      offset,
			BodyStart:  next,
			InfoString: info,
			FenceChar:  fenceChar,
			FenceLen:   fenceLen,
			Live:       isLiveFenceInfo(info),
			OpenPos:    Position{Offset: offset, Line: lineNum, Column: 1},
		}

		scanOffset := next
		nextLineNum := lineNum + 1
		closed := false
		for scanOffset < len(source) {
			bodyLine, bodyNext := readLine(source, scanOffset)
			nextLineNum++
			scanOffset = bodyNext
			if isFenceClose(bodyLine, fenceChar, fenceLen) {
				region.End = bodyNext
				closed = true
				break
			}
		}
		if !closed {
			region.End = len(source)
			region.Unclosed = true
		}

		regions = append(regions, region)
		offset = region.End
		lineNum = nextLineNum
	}

	return regions
}

// readLine returns the line starting at offset (without its newline) and the
// offset of the next line start (or len(source)).
func readLine(source string, offset int) (line string, next int) {
	idx := strings.IndexByte(source[offset:], CharNewline)
	if idx < 0 {
		return source[offset:], len(source)
	}
	return source[offset : offset+idx], offset + idx + 1
}

// trimLineEnd strips a trailing carriage return (CRLF sources).
func trimLineEnd(line string) string {
	return strings.TrimSuffix(line, string(CharCarriageRet))
}

// parseFenceOpen reports whether line opens a fenced code block and returns
// the fence character, run length, and trimmed info string.
func parseFenceOpen(line string) (fenceChar byte, fenceLen int, info string, ok bool) {
	trimmed := trimLineEnd(line)

	indent := 0
	for indent < len(trimmed) && trimmed[indent] == CharSpace {
		indent++
	}
	if indent > MarkdownFenceMaxIndent || indent == len(trimmed) {
		return 0, 0, "", false
	}

	ch := trimmed[indent]
	if ch != CharBacktick && ch != CharTilde {
		return 0, 0, "", false
	}

	run := 0
	for indent+run < len(trimmed) && trimmed[indent+run] == ch {
		run++
	}
	if run < MarkdownFenceMinLen {
		return 0, 0, "", false
	}

	infoStr := strings.TrimSpace(trimmed[indent+run:])
	// CommonMark: backtick-fence info strings may not contain a backtick
	// (prevents ambiguity with inline code spans).
	if ch == CharBacktick && strings.IndexByte(infoStr, CharBacktick) >= 0 {
		return 0, 0, "", false
	}

	return ch, run, infoStr, true
}

// isFenceClose reports whether line closes a fence of the given character and
// minimum run length: same character, run >= minLen, only whitespace after.
func isFenceClose(line string, fenceChar byte, minLen int) bool {
	trimmed := trimLineEnd(line)

	indent := 0
	for indent < len(trimmed) && trimmed[indent] == CharSpace {
		indent++
	}
	if indent > MarkdownFenceMaxIndent {
		return false
	}

	run := 0
	for indent+run < len(trimmed) && trimmed[indent+run] == fenceChar {
		run++
	}
	if run < minLen {
		return false
	}

	rest := trimmed[indent+run:]
	return strings.TrimSpace(rest) == StringValueEmpty
}

// isLiveFenceInfo reports whether the info string marks a live fence whose
// contents are rendered (first whitespace-separated word == "exons").
func isLiveFenceInfo(info string) bool {
	fields := strings.Fields(info)
	return len(fields) > 0 && fields[0] == MarkdownFenceInfoLive
}
