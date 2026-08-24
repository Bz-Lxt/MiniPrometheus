package shard

import "github.com/alkaid/miniprometheus/internal/model"

func Index(ls model.Labels, n int) int {
	if n <= 1 {
		return 0
	}
	return int(ls.Hash() % uint64(n))
}
