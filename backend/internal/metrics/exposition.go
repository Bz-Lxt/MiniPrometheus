package metrics

import (
	"fmt"
	"sort"
	"strings"
)

type Sample struct {
	Name   string
	Labels map[string]string
	Value  float64
	Help   string
	Type   string
}

func RenderSamples(ss []Sample) string {
	var b strings.Builder
	seen := map[string]struct{}{}
	for _, s := range ss {
		if _, ok := seen[s.Name]; !ok {
			if s.Help != "" {
				fmt.Fprintf(&b, "# HELP %s %s\n", s.Name, s.Help)
			}
			if s.Type != "" {
				fmt.Fprintf(&b, "# TYPE %s %s\n", s.Name, s.Type)
			}
			seen[s.Name] = struct{}{}
		}
		b.WriteString(s.Name)
		if len(s.Labels) > 0 {
			keys := make([]string, 0, len(s.Labels))
			for k := range s.Labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b.WriteByte('{')
			for i, k := range keys {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(k)
				b.WriteString("=\"")
				b.WriteString(escapeLabel(s.Labels[k]))
				b.WriteByte('"')
			}
			b.WriteByte('}')
		}
		fmt.Fprintf(&b, " %g\n", s.Value)
	}
	return b.String()
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
