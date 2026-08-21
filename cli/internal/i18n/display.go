package i18n

func isWide(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0x3000 && r <= 0x303F:
		return true
	case r >= 0x3040 && r <= 0x309F:
		return true
	case r >= 0x30A0 && r <= 0x30FF:
		return true
	case r >= 0xAC00 && r <= 0xD7AF:
		return true
	case r >= 0xFF01 && r <= 0xFF60:
		return true
	case r >= 0xFE30 && r <= 0xFE4F:
		return true
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
