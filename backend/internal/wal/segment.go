package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func ListSegments(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".wal" {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func SegmentName(seq int) string {
	return fmt.Sprintf("%08d.wal", seq)
}
