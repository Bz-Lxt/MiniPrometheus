package clock

import (
	"sync"
	"time"
)

var beijing *time.Location
var locOnce sync.Once

func Beijing() *time.Location {
	locOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		beijing = loc
	})
	return beijing
}

func Now() time.Time {
	return time.Now().In(Beijing())
}

func NowUnixMilli() int64 {
	return Now().UnixMilli()
}

func Format(tsMilli int64) string {
	return time.UnixMilli(tsMilli).In(Beijing()).Format("2006-01-02 15:04:05")
}
