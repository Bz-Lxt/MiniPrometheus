package wal

import (
	"encoding/binary"
	"hash/crc32"
	"math"

	"github.com/alkaid/miniprometheus/internal/model"
)

const (
	RecSeries uint8 = 1
	RecSample uint8 = 2
)

func encodeSeries(id uint32, ls model.Labels) []byte {
	n := 4
	for _, l := range ls {
		n += 4 + len(l.Name) + 4 + len(l.Value)
	}
	b := make([]byte, 1+4+n)
	b[0] = RecSeries
	binary.BigEndian.PutUint32(b[1:5], uint32(n))
	off := 5
	binary.BigEndian.PutUint32(b[off:], id)
	off += 4
	for _, l := range ls {
		binary.BigEndian.PutUint32(b[off:], uint32(len(l.Name)))
		off += 4
		copy(b[off:], l.Name)
		off += len(l.Name)
		binary.BigEndian.PutUint32(b[off:], uint32(len(l.Value)))
		off += 4
		copy(b[off:], l.Value)
		off += len(l.Value)
	}
	return appendCRC(b)
}

func encodeSample(id uint32, t int64, v float64) []byte {
	b := make([]byte, 1+4+4+8+8)
	b[0] = RecSample
	binary.BigEndian.PutUint32(b[1:5], 20)
	binary.BigEndian.PutUint32(b[5:9], id)
	binary.BigEndian.PutUint64(b[9:17], uint64(t))
	binary.BigEndian.PutUint64(b[17:25], math.Float64bits(v))
	return appendCRC(b)
}

func appendCRC(b []byte) []byte {
	sum := crc32.ChecksumIEEE(b)
	out := make([]byte, len(b)+4)
	copy(out, b)
	binary.BigEndian.PutUint32(out[len(b):], sum)
	return out
}

func checkCRC(raw []byte) bool {
	if len(raw) < 4 {
		return false
	}
	got := binary.BigEndian.Uint32(raw[len(raw)-4:])
	return crc32.ChecksumIEEE(raw[:len(raw)-4]) == got
}

func decodePayload(rec []byte) (typ uint8, payload []byte, ok bool) {
	if !checkCRC(rec) || len(rec) < 9 {
		return 0, nil, false
	}
	body := rec[:len(rec)-4]
	typ = body[0]
	n := binary.BigEndian.Uint32(body[1:5])
	if int(n) != len(body)-5 {
		return 0, nil, false
	}
	return typ, body[5:], true
}

func decodeSeries(p []byte) (uint32, model.Labels, bool) {
	if len(p) < 4 {
		return 0, nil, false
	}
	id := binary.BigEndian.Uint32(p[:4])
	off := 4
	var ls model.Labels
	for off < len(p) {
		if off+4 > len(p) {
			return 0, nil, false
		}
		nl := int(binary.BigEndian.Uint32(p[off:]))
		off += 4
		if off+nl+4 > len(p) {
			return 0, nil, false
		}
		name := string(p[off : off+nl])
		off += nl
		vl := int(binary.BigEndian.Uint32(p[off:]))
		off += 4
		if off+vl > len(p) {
			return 0, nil, false
		}
		val := string(p[off : off+vl])
		off += vl
		ls = append(ls, model.Label{Name: name, Value: val})
	}
	return id, model.Normalize(ls), true
}

func decodeSample(p []byte) (uint32, int64, float64, bool) {
	if len(p) < 20 {
		return 0, 0, 0, false
	}
	id := binary.BigEndian.Uint32(p[:4])
	t := int64(binary.BigEndian.Uint64(p[4:12]))
	v := math.Float64frombits(binary.BigEndian.Uint64(p[12:20]))
	return id, t, v, true
}
