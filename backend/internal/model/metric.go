package model

import "strings"

func Metric(ls Labels) string { return ls.Get(MetricName) }

func DropName(ls Labels) Labels { return ls.Without(MetricName) }

func MatchJobInstance(ls Labels, job, instance string) bool {
	if job != "" && ls.Get("job") != job {
		return false
	}
	if instance != "" && ls.Get("instance") != instance {
		return false
	}
	return true
}

func ValidName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == ':' || r == '_' {
			continue
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func Quote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func Fingerprint(ls Labels) string {
	return ls.String()
}

func IsSyntheticJob(ls Labels) bool {
	j := ls.Get("job")
	return j == "api" || j == "worker" || j == "loadgen"
}

func HasLabel(ls Labels, name, value string) bool {
	return ls.Get(name) == value
}
