package promql

func Walk(n Node, fn func(Node)) {
	if n == nil {
		return
	}
	fn(n)
	switch t := n.(type) {
	case *Call:
		for _, a := range t.Args {
			Walk(a, fn)
		}
	case *Aggregate:
		if t.Param != nil {
			Walk(t.Param, fn)
		}
		Walk(t.Expr, fn)
	case *Binary:
		Walk(t.Left, fn)
		Walk(t.Right, fn)
	}
}

func CollectSelectors(n Node) []*Selector {
	var out []*Selector
	Walk(n, func(x Node) {
		if s, ok := x.(*Selector); ok {
			out = append(out, s)
		}
	})
	return out
}

func HasRange(n Node) bool {
	found := false
	Walk(n, func(x Node) {
		if s, ok := x.(*Selector); ok && s.Range > 0 {
			found = true
		}
	})
	return found
}
