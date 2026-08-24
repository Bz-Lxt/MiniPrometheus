package promql

import (
	"fmt"
	"strings"

	"github.com/alkaid/miniprometheus/internal/index"
)

type Node interface {
	Pos() (int, int)
	String() string
}

type pos struct{ Line, Col int }

func (p pos) Pos() (int, int) { return p.Line, p.Col }

type NumberLiteral struct {
	pos
	Val float64
}

func (n *NumberLiteral) String() string { return fmt.Sprintf("%v", n.Val) }

type Selector struct {
	pos
	Name     string
	Matchers []*index.Matcher
	Range    int64
}

func (s *Selector) String() string {
	var b strings.Builder
	b.WriteString(s.Name)
	if len(s.Matchers) > 0 || s.Name == "" {
		b.WriteByte('{')
		for i, m := range s.Matchers {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(m.String())
		}
		b.WriteByte('}')
	}
	if s.Range > 0 {
		b.WriteString(fmt.Sprintf("[%dms]", s.Range))
	}
	return b.String()
}

type Call struct {
	pos
	Func string
	Args []Node
}

func (c *Call) String() string {
	var b strings.Builder
	b.WriteString(c.Func)
	b.WriteByte('(')
	for i, a := range c.Args {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(a.String())
	}
	b.WriteByte(')')
	return b.String()
}

type Aggregate struct {
	pos
	Op      string
	By      []string
	Without []string
	Param   Node
	Expr    Node
}

func (a *Aggregate) String() string {
	var b strings.Builder
	b.WriteString(a.Op)
	if len(a.By) > 0 {
		b.WriteString(" by (")
		b.WriteString(strings.Join(a.By, ","))
		b.WriteString(") ")
	}
	if len(a.Without) > 0 {
		b.WriteString(" without (")
		b.WriteString(strings.Join(a.Without, ","))
		b.WriteString(") ")
	}
	b.WriteByte('(')
	if a.Param != nil {
		b.WriteString(a.Param.String())
		b.WriteByte(',')
	}
	b.WriteString(a.Expr.String())
	b.WriteByte(')')
	return b.String()
}

type Binary struct {
	pos
	Op    TokenKind
	Left  Node
	Right Node
}

func (b *Binary) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Op.String(), b.Right.String())
}

type ParseError struct {
	Line, Col int
	Msg       string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %d:%d: %s", e.Line, e.Col, e.Msg)
}
