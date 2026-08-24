package block

import (
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/alkaid/miniprometheus/internal/model"
)

type SparsePosting struct {
	Hash   uint64
	Offset uint32
}

func WriteSparseIndex(dir string, series []SeriesChunks) error {
	f, err := os.Create(filepath.Join(dir, "index.bin"))
	if err != nil {
		return err
	}
	defer f.Close()
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(series)))
	if _, err := f.Write(hdr[:]); err != nil {
		return err
	}
	for i, s := range series {
		var rec [16]byte
		binary.BigEndian.PutUint64(rec[0:8], s.Labels.Hash())
		binary.BigEndian.PutUint32(rec[8:12], uint32(i))
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(s.Chunks)))
		if _, err := f.Write(rec[:]); err != nil {
			return err
		}
	}
	return nil
}

func ReadSparseIndex(dir string) ([]SparsePosting, error) {
	b, err := os.ReadFile(filepath.Join(dir, "index.bin"))
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, nil
	}
	n := int(binary.BigEndian.Uint32(b[:4]))
	out := make([]SparsePosting, 0, n)
	off := 4
	for i := 0; i < n && off+16 <= len(b); i++ {
		out = append(out, SparsePosting{
			Hash:   binary.BigEndian.Uint64(b[off:]),
			Offset: binary.BigEndian.Uint32(b[off+8:]),
		})
		off += 16
	}
	_ = model.MetricName
	return out, nil
}
