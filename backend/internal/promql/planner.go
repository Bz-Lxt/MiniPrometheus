package promql

import (
	"fmt"

	"github.com/alkaid/miniprometheus/internal/index"
)

type PlanKind int

const (
	PlanSelect PlanKind = iota
	PlanMatrix
	PlanCall
	PlanAgg
	PlanBinOp
	PlanLiteral
)

type Plan struct {
	Kind     PlanKind
	Op       string
	Detail   string
	Matchers []*index.Matcher
	Range    int64
	Param    float64
	HasParam bool
	By       []string
	Without  []string
	Kids     []*Plan
	Lit      float64
}

func PlanOf(n Node) (*Plan, error) {
	switch t := n.(type) {
	case *NumberLiteral:
		return &Plan{Kind: PlanLiteral, Op: "literal", Detail: t.String(), Lit: t.Val}, nil
	case *Selector:
		kind := PlanSelect
		op := "vectorSelector"
		if t.Range > 0 {
			kind = PlanMatrix
			op = "matrixSelector"
		}
		return &Plan{Kind: kind, Op: op, Detail: t.String(), Matchers: t.Matchers, Range: t.Range}, nil
	case *Call:
		kids := make([]*Plan, 0, len(t.Args))
		var param float64
		var hasParam bool
		for i, a := range t.Args {
			if lit, ok := a.(*NumberLiteral); ok && i == 0 && t.Func == "histogram_quantile" {
				param = lit.Val
				hasParam = true
				continue
			}
			p, err := PlanOf(a)
			if err != nil {
				return nil, err
			}
			kids = append(kids, p)
		}
		return &Plan{Kind: PlanCall, Op: t.Func, Detail: t.String(), Kids: kids, Param: param, HasParam: hasParam}, nil
	case *Aggregate:
		kid, err := PlanOf(t.Expr)
		if err != nil {
			return nil, err
		}
		p := &Plan{Kind: PlanAgg, Op: t.Op, Detail: t.String(), Kids: []*Plan{kid}, By: t.By, Without: t.Without}
		if t.Param != nil {
			if lit, ok := t.Param.(*NumberLiteral); ok {
				p.Param = lit.Val
				p.HasParam = true
			} else {
				return nil, fmt.Errorf("aggregate param must be scalar")
			}
		}
		return p, nil
	case *Binary:
		l, err := PlanOf(t.Left)
		if err != nil {
			return nil, err
		}
		r, err := PlanOf(t.Right)
		if err != nil {
			return nil, err
		}
		return &Plan{Kind: PlanBinOp, Op: t.Op.String(), Detail: t.String(), Kids: []*Plan{l, r}}, nil
	default:
		return nil, fmt.Errorf("unknown node %T", n)
	}
}

func (p *Plan) collectMatchers() []*index.Matcher {
	if p == nil {
		return nil
	}
	if len(p.Matchers) > 0 {
		return p.Matchers
	}
	var out []*index.Matcher
	for _, k := range p.Kids {
		out = append(out, k.collectMatchers()...)
	}
	return out
}
