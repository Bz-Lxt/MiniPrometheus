package promql

import "math"

func evalBinOp(op string, left, right Value) Value {
	if left.Kind == ValScalar && right.Kind == ValScalar {
		return Value{Kind: ValScalar, Scalar: SamplePoint{T: left.Scalar.T, V: applyOp(op, left.Scalar.V, right.Scalar.V)}}
	}
	if left.Kind == ValScalar {
		return mapScalar(op, left.Scalar.V, right, false)
	}
	if right.Kind == ValScalar {
		return mapScalar(op, right.Scalar.V, left, true)
	}
	rm := map[string]Series{}
	for _, s := range right.Series {
		rm[s.Labels.String()] = s
	}
	out := Value{Kind: ValVector}
	for _, s := range left.Series {
		o, ok := rm[s.Labels.String()]
		if !ok || len(s.Points) == 0 || len(o.Points) == 0 {
			continue
		}
		v := applyOp(op, s.Points[len(s.Points)-1].V, o.Points[len(o.Points)-1].V)
		if math.IsNaN(v) && (op == "==" || op == "!=" || op == ">" || op == "<" || op == ">=" || op == "<=") {
			continue
		}
		out.Series = append(out.Series, Series{
			Labels: s.Labels,
			Points: []SamplePoint{{T: s.Points[len(s.Points)-1].T, V: v}},
		})
	}
	return out
}

func mapScalar(op string, sc float64, vec Value, scalarRight bool) Value {
	out := Value{Kind: ValVector}
	for _, s := range vec.Series {
		if len(s.Points) == 0 {
			continue
		}
		lv, rv := s.Points[len(s.Points)-1].V, sc
		if !scalarRight {
			lv, rv = sc, lv
		}
		v := applyOp(op, lv, rv)
		out.Series = append(out.Series, Series{Labels: s.Labels, Points: []SamplePoint{{T: s.Points[len(s.Points)-1].T, V: v}}})
	}
	return out
}

func applyOp(op string, a, b float64) float64 {
	switch op {
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	case "/":
		if b == 0 {
			return math.NaN()
		}
		return a / b
	case "%":
		if b == 0 {
			return math.NaN()
		}
		return math.Mod(a, b)
	case "^":
		return math.Pow(a, b)
	case "==":
		return btof(a == b)
	case "!=":
		return btof(a != b)
	case ">":
		return btof(a > b)
	case "<":
		return btof(a < b)
	case ">=":
		return btof(a >= b)
	case "<=":
		return btof(a <= b)
	default:
		return math.NaN()
	}
}

func btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
