package encode

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/alkaid/miniprometheus/internal/model"
)

const (
	MaxPoints   = 120
	BlockBound  = int64(2 * 60 * 60 * 1000)
	headerMagic = uint32(0x4D504331) // MPC1
)

var ErrCorrupt = errors.New("chunk: corrupt payload")

type Stats struct {
	Points     int
	RawBytes   int
	CompBytes  int
	DoDZero    int
	XORZero    int
	XORReuse   int
	XORReset   int
}

func (s Stats) BytesPerSample() float64 {
	if s.Points == 0 {
		return 0
	}
	return float64(s.CompBytes) / float64(s.Points)
}

func (s Stats) Ratio() float64 {
	if s.RawBytes == 0 {
		return 0
	}
	return float64(s.CompBytes) / float64(s.RawBytes)
}

func Encode(samples []model.Sample) ([]byte, Stats, error) {
	if len(samples) == 0 {
		return nil, Stats{}, nil
	}
	if len(samples) > MaxPoints {
		return nil, Stats{}, fmt.Errorf("chunk: %d points exceeds %d", len(samples), MaxPoints)
	}
	ts := make([]int64, len(samples))
	vs := make([]float64, len(samples))
	for i, s := range samples {
		ts[i] = s.T
		vs[i] = s.V
	}
	tw := NewWriter(len(samples) * 3)
	vw := NewWriter(len(samples) * 4)
	WriteDoD(tw, ts)
	WriteGorilla(vw, vs)
	tb := tw.Bytes()
	vb := vw.Bytes()
	out := make([]byte, 4+2+4+4+len(tb)+len(vb))
	binary.BigEndian.PutUint32(out[0:4], headerMagic)
	binary.BigEndian.PutUint16(out[4:6], uint16(len(samples)))
	binary.BigEndian.PutUint32(out[6:10], uint32(len(tb)))
	binary.BigEndian.PutUint32(out[10:14], uint32(len(vb)))
	copy(out[14:], tb)
	copy(out[14+len(tb):], vb)
	st := Stats{
		Points:    len(samples),
		RawBytes:  len(samples) * 16,
		CompBytes: len(tb) + len(vb),
	}
	return out, st, nil
}

func Decode(b []byte) ([]model.Sample, error) {
	if len(b) == 0 {
		return nil, nil
	}
	if len(b) < 14 {
		return nil, ErrCorrupt
	}
	if binary.BigEndian.Uint32(b[0:4]) != headerMagic {
		return nil, ErrCorrupt
	}
	n := int(binary.BigEndian.Uint16(b[4:6]))
	tlen := int(binary.BigEndian.Uint32(b[6:10]))
	vlen := int(binary.BigEndian.Uint32(b[10:14]))
	if n < 0 || 14+tlen+vlen > len(b) {
		return nil, ErrCorrupt
	}
	tr := NewReader(b[14 : 14+tlen])
	vr := NewReader(b[14+tlen : 14+tlen+vlen])
	ts, ok := ReadDoD(tr, n)
	if !ok {
		return nil, ErrCorrupt
	}
	vs, ok := ReadGorilla(vr, n)
	if !ok {
		return nil, ErrCorrupt
	}
	out := make([]model.Sample, n)
	for i := 0; i < n; i++ {
		out[i] = model.Sample{T: ts[i], V: vs[i]}
	}
	return out, nil
}

func ShouldCut(prev []model.Sample, nextT int64) bool {
	if len(prev) == 0 {
		return false
	}
	if len(prev) >= MaxPoints {
		return true
	}
	t0 := prev[0].T
	return nextT/BlockBound != t0/BlockBound
}
