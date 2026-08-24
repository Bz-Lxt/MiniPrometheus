package bitmap

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitmapVsSorted(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	gen := func() []uint32 {
		n := 200 + rng.Intn(400)
		out := make([]uint32, n)
		for i := range out {
			out[i] = uint32(rng.Intn(80_000))
		}
		return out
	}
	for i := 0; i < 20; i++ {
		a, b := gen(), gen()
		bmA, bmB := FromIDs(a), FromIDs(b)
		sa, sb := NewSorted(a), NewSorted(b)
		require.Equal(t, sa.And(sb).IDs(), bmA.And(bmB).ToArray())
		require.Equal(t, sa.Or(sb).IDs(), bmA.Or(bmB).ToArray())
		require.Equal(t, sa.AndNot(sb).IDs(), bmA.AndNot(bmB).ToArray())
	}
}
