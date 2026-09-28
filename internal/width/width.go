// Package width measures text as a terminal draws it: how many cells a
// string takes, with escape sequences skipped, wide characters (East
// Asian, emoji) counted as two, and combining marks as none. A small
// table of ranges rather than a Unicode library: the renderer and the
// picture code need a number that is right for prose, and a library
// that is right for everything costs half a megabyte in each tool that
// measures a line.
package width

import (
	"strings"
	"unicode"
)

// String is the width of s in cells.
func String(s string) int {
	w := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		r, n := decode(s[i:])
		w += Rune(r)
		i += n
	}
	return w
}

// Strip returns s with its escape sequences removed.
func Strip(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// escapeLen is the length of the escape sequence at the start of s (which
// begins with ESC): a CSI sequence to its final byte, an OSC string to
// its terminator, or two bytes for the rest.
func escapeLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	default:
		return 2
	}
}

func decode(s string) (rune, int) {
	for _, r := range s {
		n := len(string(r))
		if r == unicode.ReplacementChar && n == 3 && !strings.HasPrefix(s, "�") {
			return r, 1
		}
		return r, n
	}
	return 0, 1
}

// Rune is the width of one character: 0 for a control or combining mark,
// 2 for a wide one, 1 otherwise.
func Rune(r rune) int {
	switch {
	case r == 0 || r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case r == 0x200b || r == 0x200c || r == 0x200d || r == 0xfe0f || r == 0x2060:
		return 0 // zero-width space and joiners, the emoji presentation selector
	case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r):
		return 0
	case wide(r):
		return 2
	}
	return 1
}

// wide says whether r takes two cells: the East Asian wide and fullwidth
// ranges, and the emoji that terminals draw double width.
func wide(r rune) bool {
	for _, rg := range wideRanges {
		if r >= rg[0] && r <= rg[1] {
			return true
		}
	}
	return false
}

var wideRanges = [][2]rune{
	{0x1100, 0x115f},   // Hangul Jamo
	{0x2e80, 0x303e},   // CJK radicals, punctuation
	{0x3041, 0x33ff},   // Hiragana, Katakana, CJK compatibility
	{0x3400, 0x4dbf},   // CJK extension A
	{0x4e00, 0x9fff},   // CJK unified
	{0xa000, 0xa4cf},   // Yi
	{0xac00, 0xd7a3},   // Hangul syllables
	{0xf900, 0xfaff},   // CJK compatibility ideographs
	{0xfe30, 0xfe4f},   // CJK compatibility forms
	{0xff00, 0xff60},   // fullwidth forms
	{0xffe0, 0xffe6},   // fullwidth signs
	{0x1f300, 0x1f64f}, // emoji: symbols, pictographs, faces
	{0x1f680, 0x1f6ff}, // transport
	{0x1f900, 0x1f9ff}, // supplemental symbols
	{0x1fa70, 0x1faff}, // symbols extended
	{0x20000, 0x3fffd}, // CJK extensions B and on
}
