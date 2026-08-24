package promql

type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokString
	TokNumber
	TokDuration
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokLBrack
	TokRBrack
	TokComma
	TokEq
	TokNe
	TokRe
	TokNre
	TokAdd
	TokSub
	TokMul
	TokDiv
	TokMod
	TokPow
	TokGte
	TokLte
	TokGt
	TokLt
	TokBy
	TokWithout
	TokError
)

type Token struct {
	Kind TokenKind
	Lit  string
	Line int
	Col  int
}

func (k TokenKind) String() string {
	names := map[TokenKind]string{
		TokEOF: "EOF", TokIdent: "IDENT", TokString: "STRING", TokNumber: "NUMBER",
		TokDuration: "DURATION", TokEq: "=", TokNe: "!=", TokRe: "=~", TokNre: "!~",
		TokAdd: "+", TokSub: "-", TokMul: "*", TokDiv: "/", TokMod: "%", TokPow: "^",
		TokBy: "by", TokWithout: "without",
	}
	if s, ok := names[k]; ok {
		return s
	}
	return "?"
}
