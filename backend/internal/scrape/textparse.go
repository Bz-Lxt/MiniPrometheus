package scrape

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/alkaid/miniprometheus/internal/model"
)

type Parsed struct {
	Labels    model.Labels
	Value     float64
	Timestamp int64
}

func ParseText(r io.Reader) ([]Parsed, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []Parsed
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, ok := parseLine(line)
		if ok {
			out = append(out, p)
		}
	}
	return out, sc.Err()
}

func parseLine(line string) (Parsed, bool) {
	nameEnd := 0
	for nameEnd < len(line) && (isName(rune(line[nameEnd])) || line[nameEnd] == ':') {
		nameEnd++
	}
	if nameEnd == 0 {
		return Parsed{}, false
	}
	name := line[:nameEnd]
	rest := strings.TrimSpace(line[nameEnd:])
	labels := model.FromMap(name, nil)
	if strings.HasPrefix(rest, "{") {
		end := strings.LastIndex(rest, "}")
		if end < 0 {
			return Parsed{}, false
		}
		inside := rest[1:end]
		rest = strings.TrimSpace(rest[end+1:])
		lm, ok := parseLabels(inside)
		if !ok {
			return Parsed{}, false
		}
		labels = model.FromMap(name, lm)
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return Parsed{}, false
	}
	v, err := model.ParseFloat(fields[0])
	if err != nil {
		return Parsed{}, false
	}
	var ts int64
	if len(fields) > 1 {
		if t, e := strconv.ParseInt(fields[1], 10, 64); e == nil {
			if t < 1e11 {
				t *= 1000
			}
			ts = t
		}
	}
	return Parsed{Labels: labels, Value: v, Timestamp: ts}, true
}

func parseLabels(s string) (map[string]string, bool) {
	m := map[string]string{}
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ',' || unicode.IsSpace(rune(s[i]))) {
			i++
		}
		if i >= len(s) {
			break
		}
		start := i
		for i < len(s) && s[i] != '=' {
			i++
		}
		if i >= len(s) {
			return nil, false
		}
		k := strings.TrimSpace(s[start:i])
		i++
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
			return nil, false
		}
		q := s[i]
		i++
		var b strings.Builder
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
			if s[i] == q {
				i++
				break
			}
			b.WriteByte(s[i])
			i++
		}
		m[k] = b.String()
	}
	return m, true
}

func isName(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func ParseFloatFast(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
