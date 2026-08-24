package promql

import "fmt"

func Unsupported(feature string, line, col int) error {
	return &ParseError{Line: line, Col: col, Msg: fmt.Sprintf("unsupported in MiniQL subset: %s", feature)}
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return containsStr(s, "timeout") || containsStr(s, "deadline")
}

func IsLimit(err error) bool {
	if err == nil {
		return false
	}
	return containsStr(err.Error(), "max samples")
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexStr(s, sub) >= 0
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
