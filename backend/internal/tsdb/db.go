package tsdb

import (
	"path/filepath"
	"time"

	"github.com/alkaid/miniprometheus/internal/block"
	"github.com/alkaid/miniprometheus/internal/head"
	"github.com/alkaid/miniprometheus/internal/index"
	"github.com/alkaid/miniprometheus/internal/logger"
	"github.com/alkaid/miniprometheus/internal/model"
	"github.com/alkaid/miniprometheus/internal/wal"
)

type DB struct {
	Head   *head.Head
	Blocks *block.Store
	WAL    *wal.WAL
	dir    string
}

func Open(dir string) (*DB, error) {
	if _, err := wal.RepairTail(filepath.Join(dir, "wal")); err != nil {
		logger.L().Warn("wal repair", "err", err.Error())
	}
	w, err := wal.Open(filepath.Join(dir, "wal"))
	if err != nil {
		return nil, err
	}
	h := head.New(w)
	if n, err := wal.Replay(filepath.Join(dir, "wal"), h); err != nil {
		logger.L().Warn("replay", "err", err.Error(), "n", n)
	}
	bs, err := block.OpenStore(filepath.Join(dir, "blocks"))
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	return &DB{Head: h, Blocks: bs, WAL: w, dir: dir}, nil
}

func (d *DB) Close() error {
	d.Head.FlushAll()
	return d.WAL.Close()
}

func (d *DB) Query(ms []*index.Matcher, mint, maxt int64) ([]model.TimeSeries, index.LookupStat) {
	hs, st := d.Head.Query(ms, mint, maxt)
	bs := d.Blocks.Query(ms, mint, maxt)
	return merge(hs, bs), st
}

func merge(a, b []model.TimeSeries) []model.TimeSeries {
	type acc struct {
		ls model.Labels
		m  map[int64]float64
	}
	idx := map[string]*acc{}
	order := []string{}
	add := func(part []model.TimeSeries) {
		for _, s := range part {
			k := s.Labels.String()
			x, ok := idx[k]
			if !ok {
				x = &acc{ls: s.Labels, m: map[int64]float64{}}
				idx[k] = x
				order = append(order, k)
			}
			for _, p := range s.Samples {
				x.m[p.T] = p.V
			}
		}
	}
	add(a)
	add(b)
	out := make([]model.TimeSeries, 0, len(order))
	for _, k := range order {
		x := idx[k]
		ts := make([]int64, 0, len(x.m))
		for t := range x.m {
			ts = append(ts, t)
		}
		for i := 0; i < len(ts); i++ {
			for j := i + 1; j < len(ts); j++ {
				if ts[j] < ts[i] {
					ts[i], ts[j] = ts[j], ts[i]
				}
			}
		}
		ss := make([]model.Sample, 0, len(ts))
		for _, t := range ts {
			ss = append(ss, model.Sample{T: t, V: x.m[t]})
		}
		out = append(out, model.TimeSeries{Labels: x.ls, Samples: ss})
	}
	return out
}

func (d *DB) Compact() (*block.Block, error) {
	b, err := block.CompactHead(d.Head, d.Blocks)
	if err != nil || b == nil {
		return b, err
	}
	_ = block.WriteSparseIndex(b.Dir, b.Series())
	return b, nil
}

func (d *DB) Expire(ret time.Duration) {
	d.Blocks.Expire(time.Now().Add(-ret).UnixMilli())
}
