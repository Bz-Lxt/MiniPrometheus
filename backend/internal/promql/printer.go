package promql

import "strings"

func PrintProfile(n ProfileNode, indent int) string {
	var b strings.Builder
	pad := strings.Repeat("  ", indent)
	b.WriteString(pad)
	b.WriteString(n.Op)
	b.WriteString("  ")
	b.WriteString(formatMS(n.DurationMS))
	b.WriteString("  in=")
	b.WriteString(itoa(n.InSeries))
	b.WriteString(" out=")
	b.WriteString(itoa(n.OutSeries))
	b.WriteString(" scan=")
	b.WriteString(itoa(n.Samples))
	if n.HitIndex {
		b.WriteString(" idx")
	}
	b.WriteByte('\n')
	for _, c := range n.Children {
		b.WriteString(PrintProfile(c, indent+1))
	}
	return b.String()
}

func formatMS(v float64) string {
	return strings.TrimRight(strings.TrimRight(sprintf(v), "0"), ".") + "ms"
}

func sprintf(v float64) string {
	s := ""
	neg := v < 0
	if neg {
		v = -v
	}
	ip := int(v)
	fp := int((v - float64(ip)) * 1000)
	s = itoa(ip) + "." + pad3(fp)
	if neg {
		s = "-" + s
	}
	return s
}

func pad3(n int) string {
	if n < 0 {
		n = 0
	}
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
