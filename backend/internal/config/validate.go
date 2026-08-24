package config

import "fmt"

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return fmt.Errorf("http addr required")
	}
	if c.Role == "storage" && c.DataDir == "" {
		return fmt.Errorf("data dir required")
	}
	if c.Role == "gateway" && len(c.Shards) == 0 {
		return fmt.Errorf("gateway requires MP_SHARDS")
	}
	if c.ShardCount < 1 {
		return fmt.Errorf("shard count must be >= 1")
	}
	if c.MaxQueries < 1 {
		return fmt.Errorf("max queries must be >= 1")
	}
	return nil
}
