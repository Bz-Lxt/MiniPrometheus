package promql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/model"
)

var funcs = map[string]bool{
	"rate": true, "irate": true, "increase": true, "delta": true,
	"avg_over_time": true, "max_over_time": true, "min_over_time": true,
	"sum_over_time": true, "count_over_time": true, "histogram_quantile": true,
}

var aggs = map[string]bool{
	"sum": true, "avg": true, "min": true, "max": true, "count": true, "topk": true, "quantile": true,
}

type Parser struct {
	lx  *Lexer
	cur Token
}

func Parse(src string) (Node, error) {
	p := &Parser{lx: NewLexer(src)}
	p.next()
	n, err := p.parseCompare()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind != TokEOF {
		return nil, p.err("trailing input %q", p.cur.Lit)
	}
	return n, nil
}

func (p *Parser) next() { p.cur = p.lx.Next() }

func (p *Parser) err(f string, args ...any) error {
	return &ParseError{Line: p.cur.Line, Col: p.cur.Col, Msg: fmt.Sprintf(f, args...)}
}

func (p *Parser) parseCompare() (Node, error) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	for {
		switch p.cur.Kind {
		case TokEq, TokNe, TokGt, TokLt, TokGte, TokLte:
			op := p.cur.Kind
			pos := pos{p.cur.Line, p.cur.Col}
			p.next()
			right, err := p.parseAdd()
			if err != nil {
				return nil, err
			}
			left = &Binary{pos: pos, Op: op, Left: left, Right: right}
		default:
			return left, nil
		}
	}
}

func (p *Parser) parseAdd() (Node, error) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokAdd || p.cur.Kind == TokSub {
		op := p.cur.Kind
		pos := pos{p.cur.Line, p.cur.Col}
		p.next()
		right, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		left = &Binary{pos: pos, Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMul() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == TokMul || p.cur.Kind == TokDiv || p.cur.Kind == TokMod || p.cur.Kind == TokPow {
		op := p.cur.Kind
		pos := pos{p.cur.Line, p.cur.Col}
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &Binary{pos: pos, Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Node, error) {
	if p.cur.Kind == TokError {
		return nil, p.err("%s", p.cur.Lit)
	}
	if p.cur.Kind == TokNumber {
		v, err := strconv.ParseFloat(p.cur.Lit, 64)
		if err != nil {
			return nil, p.err("bad number %q", p.cur.Lit)
		}
		n := &NumberLiteral{pos: pos{p.cur.Line, p.cur.Col}, Val: v}
		p.next()
		return n, nil
	}
	if p.cur.Kind == TokLParen {
		p.next()
		n, err := p.parseCompare()
		if err != nil {
			return nil, err
		}
		if p.cur.Kind != TokRParen {
			return nil, p.err("expected )")
		}
		p.next()
		return n, nil
	}
	if p.cur.Kind != TokIdent {
		return nil, p.err("expected expression, got %s", p.cur.Kind)
	}
	name := p.cur.Lit
	line, col := p.cur.Line, p.cur.Col
	p.next()
	if aggs[name] {
		return p.parseAgg(name, line, col)
	}
	if funcs[name] && p.cur.Kind == TokLParen {
		return p.parseCall(name, line, col)
	}
	return p.parseSelector(name, line, col)
}

func (p *Parser) parseCall(name string, line, col int) (Node, error) {
	p.next()
	var args []Node
	for p.cur.Kind != TokRParen && p.cur.Kind != TokEOF {
		a, err := p.parseCompare()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		if p.cur.Kind == TokComma {
			p.next()
			continue
		}
		break
	}
	if p.cur.Kind != TokRParen {
		return nil, p.err("expected ) after %s(", name)
	}
	p.next()
	return &Call{pos: pos{line, col}, Func: name, Args: args}, nil
}

func (p *Parser) parseAgg(name string, line, col int) (Node, error) {
	a := &Aggregate{pos: pos{line, col}, Op: name}
	if p.cur.Kind == TokBy || p.cur.Kind == TokWithout {
		without := p.cur.Kind == TokWithout
		p.next()
		labs, err := p.parseLabelList()
		if err != nil {
			return nil, err
		}
		if without {
			a.Without = labs
		} else {
			a.By = labs
		}
	}
	if p.cur.Kind != TokLParen {
		return nil, p.err("expected ( after %s", name)
	}
	p.next()
	first, err := p.parseCompare()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == TokComma {
		p.next()
		second, err := p.parseCompare()
		if err != nil {
			return nil, err
		}
		a.Param = first
		a.Expr = second
	} else {
		a.Expr = first
	}
	if p.cur.Kind != TokRParen {
		return nil, p.err("expected ) after aggregate")
	}
	p.next()
	return a, nil
}

func (p *Parser) parseLabelList() ([]string, error) {
	if p.cur.Kind != TokLParen {
		return nil, p.err("expected (label list)")
	}
	p.next()
	var out []string
	for p.cur.Kind == TokIdent {
		out = append(out, p.cur.Lit)
		p.next()
		if p.cur.Kind == TokComma {
			p.next()
			continue
		}
		break
	}
	if p.cur.Kind != TokRParen {
		return nil, p.err("expected ) after label list")
	}
	p.next()
	return out, nil
}

func (p *Parser) parseSelector(name string, line, col int) (Node, error) {
	s := &Selector{pos: pos{line, col}, Name: name}
	if name != "" {
		s.Matchers = append(s.Matchers, &index.Matcher{Name: model.MetricName, Type: index.MatchEqual, Value: name})
	}
	if p.cur.Kind == TokLBrace {
		p.next()
		for p.cur.Kind != TokRBrace && p.cur.Kind != TokEOF {
			if p.cur.Kind != TokIdent {
				return nil, p.err("expected label name")
			}
			ln := p.cur.Lit
			p.next()
			var mt index.MatchType
			switch p.cur.Kind {
			case TokEq:
				mt = index.MatchEqual
			case TokNe:
				mt = index.MatchNotEqual
			case TokRe:
				mt = index.MatchRegexp
			case TokNre:
				mt = index.MatchNotRegexp
			default:
				return nil, p.err("expected matcher operator")
			}
			p.next()
			if p.cur.Kind != TokString {
				return nil, p.err("expected matcher string")
			}
			s.Matchers = append(s.Matchers, &index.Matcher{Name: ln, Type: mt, Value: p.cur.Lit})
			p.next()
			if p.cur.Kind == TokComma {
				p.next()
			}
		}
		if p.cur.Kind != TokRBrace {
			return nil, p.err("expected }")
		}
		p.next()
	}
	if p.cur.Kind == TokDuration {
		d, err := ParseDuration(p.cur.Lit)
		if err != nil {
			return nil, p.err("%s", err.Error())
		}
		s.Range = d
		p.next()
	}
	return s, nil
}

func ParseDuration(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	i := 0
	for i < len(s) {
		start := i
		for i < len(s) && (unicode.IsDigit(rune(s[i])) || s[i] == '.') {
			i++
		}
		if start == i {
			return 0, fmt.Errorf("bad duration %q", s)
		}
		num, err := strconv.ParseFloat(s[start:i], 64)
		if err != nil {
			return 0, err
		}
		if i >= len(s) {
			return 0, fmt.Errorf("duration missing unit")
		}
		u := s[i]
		i++
		var d time.Duration
		switch u {
		case 's':
			d = time.Duration(num * float64(time.Second))
		case 'm':
			d = time.Duration(num * float64(time.Minute))
		case 'h':
			d = time.Duration(num * float64(time.Hour))
		case 'd':
			d = time.Duration(num * float64(24*time.Hour))
		default:
			return 0, fmt.Errorf("unknown duration unit %q", u)
		}
		total += d
	}
	return total.Milliseconds(), nil
}
