package promql

import "strings"

func Pretty(n Node) string {
	if n == nil {
		return "<nil>"
	}
	return indentNode(n, 0)
}

func indentNode(n Node, depth int) string {
	pad := strings.Repeat("  ", depth)
	switch t := n.(type) {
	case *NumberLiteral:
		return pad + "literal " + t.String()
	case *Selector:
		return pad + "select " + t.String()
	case *Call:
		var b strings.Builder
		b.WriteString(pad + "call " + t.Func + "\n")
		for _, a := range t.Args {
			b.WriteString(indentNode(a, depth+1))
			b.WriteByte('\n')
		}
		return strings.TrimRight(b.String(), "\n")
	case *Aggregate:
		var b strings.Builder
		b.WriteString(pad + "agg " + t.Op)
		if len(t.By) > 0 {
			b.WriteString(" by(" + strings.Join(t.By, ",") + ")")
		}
		b.WriteByte('\n')
		b.WriteString(indentNode(t.Expr, depth+1))
		return b.String()
	case *Binary:
		return pad + "binop " + t.Op.String() + "\n" + indentNode(t.Left, depth+1) + "\n" + indentNode(t.Right, depth+1)
	default:
		return pad + n.String()
	}
}

func PlanKindName(k PlanKind) string {
	switch k {
	case PlanSelect:
		return "select"
	case PlanMatrix:
		return "matrix"
	case PlanCall:
		return "call"
	case PlanAgg:
		return "agg"
	case PlanBinOp:
		return "binop"
	case PlanLiteral:
		return "literal"
	default:
		return "unknown"
	}
}

func IsAggOp(name string) bool { return aggs[name] }

func IsFuncOp(name string) bool { return funcs[name] }
