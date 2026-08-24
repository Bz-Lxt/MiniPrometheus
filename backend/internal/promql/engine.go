package promql

import (
	"context"
	"fmt"
	"time"

	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
)

type SeriesSource func(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat)

type Engine struct {
	Src     SeriesSource
	Timeout time.Duration
	MaxScan int
}

type Result struct {
	Value   Value
	Profile ProfileNode
	Partial bool
	Err     error
}

func (e *Engine) Exec(ctx context.Context, expr string, start, end, step int64, evalTime int64) Result {
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}
	ast, err := Parse(expr)
	if err != nil {
		return Result{Err: err}
	}
	plan, err := PlanOf(ast)
	if err != nil {
		return Result{Err: err}
	}
	if end > start && step > 0 {
		orig := e.Src
		e.Src = cacheSource(orig, start-10*60*1000, end)
		res := e.evalRange(ctx, plan, start, end, step)
		e.Src = orig
		return res
	}
	if evalTime == 0 {
		evalTime = end
		if evalTime == 0 {
			evalTime = time.Now().UnixMilli()
		}
	}
	v, prof, err := e.eval(ctx, plan, evalTime, evalTime, 0)
	return Result{Value: v, Profile: prof, Err: err}
}

func (e *Engine) evalRange(ctx context.Context, plan *Plan, start, end, step int64) Result {
	var acc []Series
	var last ProfileNode
	scanned := 0
	for t := start; t <= end; t += step {
		if err := ctx.Err(); err != nil {
			return Result{Err: fmt.Errorf("query timeout: %w", err)}
		}
		v, prof, err := e.eval(ctx, plan, t, t, 0)
		if err != nil {
			return Result{Err: err}
		}
		last = prof
		scanned += prof.Samples
		if e.MaxScan > 0 && scanned > e.MaxScan {
			return Result{Err: fmt.Errorf("max samples exceeded (%d)", e.MaxScan)}
		}
		if v.Kind == ValScalar {
			if len(acc) == 0 {
				acc = []Series{{Points: nil}}
			}
			acc[0].Points = append(acc[0].Points, SamplePoint{T: t, V: v.Scalar.V})
			continue
		}
		for _, s := range v.Series {
			if len(s.Points) == 0 {
				continue
			}
			p := s.Points[len(s.Points)-1]
			p.T = t
			found := false
			for i := range acc {
				if acc[i].Labels.Equal(s.Labels) {
					acc[i].Points = append(acc[i].Points, p)
					found = true
					break
				}
			}
			if !found {
				acc = append(acc, Series{Labels: s.Labels, Points: []SamplePoint{p}})
			}
		}
	}
	last.Samples = scanned
	return Result{Value: Value{Kind: ValMatrix, Series: acc}, Profile: last}
}

func (e *Engine) eval(ctx context.Context, p *Plan, mint, maxt int64, nextID int) (Value, ProfileNode, error) {
	start := time.Now()
	node := ProfileNode{ID: nextID, Op: p.Op, Detail: p.Detail}
	switch p.Kind {
	case PlanLiteral:
		node.DurationMS = ms(start)
		return Value{Kind: ValScalar, Scalar: SamplePoint{T: maxt, V: p.Lit}}, node, nil
	case PlanSelect, PlanMatrix:
		selMin := mint
		if p.Range > 0 {
			selMin = maxt - p.Range
		} else if p.Kind == PlanSelect {
			const lookback = int64(5 * 60 * 1000)
			if maxt-selMin < lookback {
				selMin = maxt - lookback
			}
		}
		ts, st := e.Src(p.Matchers, selMin, maxt)
		ss := make([]Series, 0, len(ts))
		samples := 0
		for _, t := range ts {
			pts := make([]SamplePoint, 0, len(t.Samples))
			for _, s := range t.Samples {
				pts = append(pts, SamplePoint{T: s.T, V: s.V})
			}
			samples += len(pts)
			ss = append(ss, Series{Labels: t.Labels, Points: pts})
		}
		kind := ValVector
		if p.Kind == PlanMatrix {
			kind = ValMatrix
		} else {
			ss = toInstant(ss, maxt)
		}
		node.InSeries = st.Series
		node.OutSeries = len(ss)
		node.Samples = samples
		node.HitIndex = st.HitIndex
		node.DurationMS = ms(start)
		return Value{Kind: kind, Series: ss}, node, nil
	case PlanCall:
		var kids []ProfileNode
		var in Value
		id := nextID + 1
		for _, k := range p.Kids {
			v, pn, err := e.eval(ctx, k, mint, maxt, id)
			if err != nil {
				return Value{}, node, err
			}
			id += countNodes(pn)
			kids = append(kids, pn)
			in = v
		}
		out := evalCall(p.Op, in, p.Param)
		node.Children = kids
		node.InSeries = inSeries(kids)
		node.OutSeries = len(out.Series)
		node.Samples = sumSamples(kids)
		node.DurationMS = ms(start)
		return out, node, nil
	case PlanAgg:
		v, pn, err := e.eval(ctx, p.Kids[0], mint, maxt, nextID+1)
		if err != nil {
			return Value{}, node, err
		}
		out := evalAgg(p.Op, v, p.By, p.Without, p.Param, p.HasParam)
		node.Children = []ProfileNode{pn}
		node.InSeries = pn.OutSeries
		node.OutSeries = len(out.Series)
		node.Samples = pn.Samples
		node.DurationMS = ms(start)
		return out, node, nil
	case PlanBinOp:
		l, lp, err := e.eval(ctx, p.Kids[0], mint, maxt, nextID+1)
		if err != nil {
			return Value{}, node, err
		}
		r, rp, err := e.eval(ctx, p.Kids[1], mint, maxt, nextID+1+countNodes(lp))
		if err != nil {
			return Value{}, node, err
		}
		out := evalBinOp(p.Op, l, r)
		node.Children = []ProfileNode{lp, rp}
		node.InSeries = lp.OutSeries + rp.OutSeries
		node.OutSeries = len(out.Series)
		if out.Kind == ValScalar {
			node.OutSeries = 1
		}
		node.Samples = lp.Samples + rp.Samples
		node.DurationMS = ms(start)
		return out, node, nil
	default:
		return Value{}, node, fmt.Errorf("unknown plan kind")
	}
}

func ms(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000.0
}

func countNodes(n ProfileNode) int {
	c := 1
	for _, ch := range n.Children {
		c += countNodes(ch)
	}
	return c
}

func inSeries(ks []ProfileNode) int {
	n := 0
	for _, k := range ks {
		n += k.OutSeries
	}
	return n
}

func sumSamples(ks []ProfileNode) int {
	n := 0
	for _, k := range ks {
		n += k.Samples
	}
	return n
}
