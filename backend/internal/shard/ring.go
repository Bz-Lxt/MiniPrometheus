package shard

import "github.com/alkaid/miniprometheus/internal/model"

type Ring struct {
	n int
}

func NewRing(n int) *Ring {
	if n < 1 {
		n = 1
	}
	return &Ring{n: n}
}

func (r *Ring) Owner(ls model.Labels) int {
	return Index(ls, r.n)
}

func (r *Ring) Count() int { return r.n }

func (r *Ring) Owns(ls model.Labels, self int) bool {
	return r.Owner(ls) == self
}
