package promql

type FuncMeta struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Args    []string `json:"args"`
	Doc     string   `json:"doc"`
	Range   bool     `json:"needs_range"`
}

func Catalog() []FuncMeta {
	return []FuncMeta{
		{Name: "rate", Kind: "func", Args: []string{"matrix"}, Doc: "每秒平均增长率，处理计数器复位", Range: true},
		{Name: "irate", Kind: "func", Args: []string{"matrix"}, Doc: "最近两点的瞬时速率", Range: true},
		{Name: "increase", Kind: "func", Args: []string{"matrix"}, Doc: "窗口内增量", Range: true},
		{Name: "delta", Kind: "func", Args: []string{"matrix"}, Doc: "窗口首末差值（gauge）", Range: true},
		{Name: "avg_over_time", Kind: "func", Args: []string{"matrix"}, Doc: "窗口均值", Range: true},
		{Name: "max_over_time", Kind: "func", Args: []string{"matrix"}, Doc: "窗口最大", Range: true},
		{Name: "min_over_time", Kind: "func", Args: []string{"matrix"}, Doc: "窗口最小", Range: true},
		{Name: "sum_over_time", Kind: "func", Args: []string{"matrix"}, Doc: "窗口求和", Range: true},
		{Name: "count_over_time", Kind: "func", Args: []string{"matrix"}, Doc: "窗口点数", Range: true},
		{Name: "histogram_quantile", Kind: "func", Args: []string{"scalar", "vector"}, Doc: "经典直方图分位", Range: false},
		{Name: "sum", Kind: "agg", Args: []string{"vector"}, Doc: "求和，支持 by/without"},
		{Name: "avg", Kind: "agg", Args: []string{"vector"}, Doc: "平均"},
		{Name: "min", Kind: "agg", Args: []string{"vector"}, Doc: "最小"},
		{Name: "max", Kind: "agg", Args: []string{"vector"}, Doc: "最大"},
		{Name: "count", Kind: "agg", Args: []string{"vector"}, Doc: "序列条数"},
		{Name: "topk", Kind: "agg", Args: []string{"k", "vector"}, Doc: "最大 k 条"},
		{Name: "quantile", Kind: "agg", Args: []string{"q", "vector"}, Doc: "分位聚合"},
	}
}
