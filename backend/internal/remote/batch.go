package remote

const defaultBatch = 1000

func Split(req WriteRequest, max int) []WriteRequest {
	if max <= 0 {
		max = defaultBatch
	}
	if len(req.Series) <= max {
		return []WriteRequest{req}
	}
	var out []WriteRequest
	for i := 0; i < len(req.Series); i += max {
		j := i + max
		if j > len(req.Series) {
			j = len(req.Series)
		}
		out = append(out, WriteRequest{Series: req.Series[i:j]})
	}
	return out
}

func SampleCount(req WriteRequest) int {
	n := 0
	for _, s := range req.Series {
		n += len(s.Samples)
	}
	return n
}
