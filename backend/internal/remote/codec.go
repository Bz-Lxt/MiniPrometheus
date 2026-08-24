package remote

import (
	"encoding/binary"
	"math"

	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/golang/snappy"
)

type TimeSeries struct {
	Labels  model.Labels  `json:"labels"`
	Samples []model.Sample `json:"samples"`
}

type WriteRequest struct {
	Series []TimeSeries `json:"series"`
}

func Encode(req WriteRequest) ([]byte, error) {
	raw := marshal(req)
	return snappy.Encode(nil, raw), nil
}

func Decode(b []byte) (WriteRequest, error) {
	raw, err := snappy.Decode(nil, b)
	if err != nil {
		return WriteRequest{}, err
	}
	return unmarshal(raw)
}

func marshal(req WriteRequest) []byte {
	n := 4
	for _, s := range req.Series {
		n += 4
		for _, l := range s.Labels {
			n += 4 + len(l.Name) + 4 + len(l.Value)
		}
		n += 4 + len(s.Samples)*16
	}
	b := make([]byte, n)
	binary.BigEndian.PutUint32(b[0:4], uint32(len(req.Series)))
	off := 4
	putStr := func(s string) {
		binary.BigEndian.PutUint32(b[off:], uint32(len(s)))
		off += 4
		copy(b[off:], s)
		off += len(s)
	}
	for _, s := range req.Series {
		binary.BigEndian.PutUint32(b[off:], uint32(len(s.Labels)))
		off += 4
		for _, l := range s.Labels {
			putStr(l.Name)
			putStr(l.Value)
		}
		binary.BigEndian.PutUint32(b[off:], uint32(len(s.Samples)))
		off += 4
		for _, p := range s.Samples {
			binary.BigEndian.PutUint64(b[off:], uint64(p.T))
			off += 8
			binary.BigEndian.PutUint64(b[off:], math.Float64bits(p.V))
			off += 8
		}
	}
	return b
}

func unmarshal(b []byte) (WriteRequest, error) {
	if len(b) < 4 {
		return WriteRequest{}, ErrShort
	}
	n := int(binary.BigEndian.Uint32(b[:4]))
	off := 4
	readStr := func() (string, bool) {
		if off+4 > len(b) {
			return "", false
		}
		ln := int(binary.BigEndian.Uint32(b[off:]))
		off += 4
		if off+ln > len(b) {
			return "", false
		}
		s := string(b[off : off+ln])
		off += ln
		return s, true
	}
	out := make([]TimeSeries, 0, n)
	for i := 0; i < n; i++ {
		if off+4 > len(b) {
			return WriteRequest{}, ErrShort
		}
		nl := int(binary.BigEndian.Uint32(b[off:]))
		off += 4
		ls := make(model.Labels, 0, nl)
		for j := 0; j < nl; j++ {
			k, ok := readStr()
			if !ok {
				return WriteRequest{}, ErrShort
			}
			v, ok := readStr()
			if !ok {
				return WriteRequest{}, ErrShort
			}
			ls = append(ls, model.Label{Name: k, Value: v})
		}
		if off+4 > len(b) {
			return WriteRequest{}, ErrShort
		}
		ns := int(binary.BigEndian.Uint32(b[off:]))
		off += 4
		ss := make([]model.Sample, 0, ns)
		for j := 0; j < ns; j++ {
			if off+16 > len(b) {
				return WriteRequest{}, ErrShort
			}
			t := int64(binary.BigEndian.Uint64(b[off:]))
			off += 8
			v := math.Float64frombits(binary.BigEndian.Uint64(b[off:]))
			off += 8
			ss = append(ss, model.Sample{T: t, V: v})
		}
		out = append(out, TimeSeries{Labels: model.Normalize(ls), Samples: ss})
	}
	return WriteRequest{Series: out}, nil
}

var ErrShort = errShort{}

type errShort struct{}

func (errShort) Error() string { return "remote: short buffer" }
