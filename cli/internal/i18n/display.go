package i18n

var wideRanges = []struct{ lo, hi rune }{
	{0x4E00, 0x9FFF},
	{0x3400, 0x4DBF},
	{0xF900, 0xFAFF},
	{0x3000, 0x303F},
	{0x3040, 0x309F},
	{0x30A0, 0x30FF},
	{0xAC00, 0xD7AF},
	{0xFF01, 0xFF60},
	{0xFE30, 0xFE4F},
}

func isWide(r rune) bool {
	for _, rng := range wideRanges {
		if r >= rng.lo && r <= rng.hi {
			return true
		}
	}
	return false
}

func DisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if isWide(r) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

func PadRight(s string, displayWidth int) string {
	current := DisplayWidth(s)
	if current >= displayWidth {
		return s
	}
	pad := displayWidth - current
	var buf []byte
	buf = append(buf, s...)
	for i := 0; i < pad; i++ {
		buf = append(buf, ' ')
	}
	return string(buf)
}
