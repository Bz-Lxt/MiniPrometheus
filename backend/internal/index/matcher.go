package index

import (
	"fmt"
	"regexp"
)

type MatchType uint8

const (
	MatchEqual MatchType = iota
	MatchNotEqual
	MatchRegexp
	MatchNotRegexp
)

type Matcher struct {
	Name  string
	Value string
	Type  MatchType
	re    *regexp.Regexp
}

func (m *Matcher) Negative() bool {
	return m.Type == MatchNotEqual || m.Type == MatchNotRegexp
}

func (m *Matcher) Compile() error {
	if m.Type == MatchRegexp || m.Type == MatchNotRegexp {
		re, err := regexp.Compile("^(?:" + m.Value + ")$")
		if err != nil {
			return fmt.Errorf("matcher regex %q: %w", m.Value, err)
		}
		m.re = re
	}
	return nil
}

func (m *Matcher) Matches(v string) bool {
	switch m.Type {
	case MatchEqual:
		return v == m.Value
	case MatchNotEqual:
		return v != m.Value
	case MatchRegexp:
		return m.re != nil && m.re.MatchString(v)
	case MatchNotRegexp:
		return m.re == nil || !m.re.MatchString(v)
	default:
		return false
	}
}

func (m *Matcher) String() string {
	op := "="
	switch m.Type {
	case MatchNotEqual:
		op = "!="
	case MatchRegexp:
		op = "=~"
	case MatchNotRegexp:
		op = "!~"
	}
	return fmt.Sprintf("%s%s%q", m.Name, op, m.Value)
}
