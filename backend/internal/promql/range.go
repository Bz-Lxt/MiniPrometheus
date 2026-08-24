package promql

func AlignRange(start, end, step int64) (int64, int64, int64) {
	if step <= 0 {
		step = 15_000
	}
	if end < start {
		start, end = end, start
	}
	if end == start {
		end = start + step
	}
	start = start - (start % step)
	return start, end, step
}

func StepPoints(start, end, step int64) int {
	if step <= 0 || end < start {
		return 0
	}
	return int((end-start)/step) + 1
}

func ClampEval(mint, maxt, eval int64) (int64, int64, int64) {
	if eval == 0 {
		eval = maxt
	}
	if mint == 0 {
		mint = eval
	}
	if maxt == 0 {
		maxt = eval
	}
	return mint, maxt, eval
}
