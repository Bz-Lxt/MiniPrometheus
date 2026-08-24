package index

import (
	"strings"
	"unicode"
)

func LiteralPrefix(re string) string {
	var b strings.Builder
	esc := false
	for _, r := range re {
		if esc {
			b.WriteRune(r)
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		if strings.ContainsRune(".|*+?()[]{}^$", r) {
			break
		}
		if unicode.IsControl(r) {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func ValuesWithPrefix(all []string, prefix string) []string {
	if prefix == "" {
		return all
	}
	out := make([]string, 0, len(all))
	for _, v := range all {
		if strings.HasPrefix(v, prefix) {
			out = append(out, v)
		}
	}
	return out
}
