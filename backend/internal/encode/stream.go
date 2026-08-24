package encode

import "github.com/alkaid/miniprometheus/internal/model"

func EncodeAll(samples []model.Sample) ([][]byte, Stats, error) {
	var blobs [][]byte
	var acc Stats
	var cur []model.Sample
	flush := func() error {
		if len(cur) == 0 {
			return nil
		}
		b, st, err := Encode(cur)
		if err != nil {
			return err
		}
		blobs = append(blobs, b)
		acc.Points += st.Points
		acc.RawBytes += st.RawBytes
		acc.CompBytes += st.CompBytes
		cur = cur[:0]
		return nil
	}
	for _, s := range samples {
		if ShouldCut(cur, s.T) {
			if err := flush(); err != nil {
				return nil, acc, err
			}
		}
		cur = append(cur, s)
	}
	if err := flush(); err != nil {
		return nil, acc, err
	}
	return blobs, acc, nil
}

func DecodeAll(blobs [][]byte) ([]model.Sample, error) {
	var out []model.Sample
	for _, b := range blobs {
		ss, err := Decode(b)
		if err != nil {
			return nil, err
		}
		out = append(out, ss...)
	}
	return out, nil
}
