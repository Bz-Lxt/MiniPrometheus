package bitmap

func RunsOf(ids []uint32) [][2]uint32 {
	if len(ids) == 0 {
		return nil
	}
	var out [][2]uint32
	start, prev := ids[0], ids[0]
	for i := 1; i < len(ids); i++ {
		if ids[i] == prev+1 {
			prev = ids[i]
			continue
		}
		out = append(out, [2]uint32{start, prev})
		start, prev = ids[i], ids[i]
	}
	out = append(out, [2]uint32{start, prev})
	return out
}

func ExpandRuns(runs [][2]uint32) []uint32 {
	n := 0
	for _, r := range runs {
		if r[1] >= r[0] {
			n += int(r[1] - r[0] + 1)
		}
	}
	out := make([]uint32, 0, n)
	for _, r := range runs {
		for v := r[0]; v <= r[1]; v++ {
			out = append(out, v)
			if v == 0xffffffff {
				break
			}
		}
	}
	return out
}
