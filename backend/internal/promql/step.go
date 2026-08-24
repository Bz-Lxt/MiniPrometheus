package promql

func IterateSteps(start, end, step int64, fn func(t int64) error) error {
	start, end, step = AlignRange(start, end, step)
	for t := start; t <= end; t += step {
		if err := fn(t); err != nil {
			return err
		}
	}
	return nil
}

func LastPoint(s Series) (SamplePoint, bool) {
	if len(s.Points) == 0 {
		return SamplePoint{}, false
	}
	return s.Points[len(s.Points)-1], true
}

func PointAtOrBefore(s Series, t int64) (SamplePoint, bool) {
	var last SamplePoint
	ok := false
	for _, p := range s.Points {
		if p.T > t {
			break
		}
		last, ok = p, true
	}
	return last, ok
}
