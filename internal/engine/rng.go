package engine

// RNG is a xorshift64* generator. It is deliberately hand-rolled rather than
// math/rand: --daily and --replay require that a given seed produce an
// identical word sequence across versions, platforms and Go releases, and the
// standard library makes no such promise.
type RNG struct{ s uint64 }

// NewRNG seeds the generator. A zero seed is remapped, since xorshift is stuck
// at zero.
func NewRNG(seed uint64) *RNG {
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15
	}
	return &RNG{s: seed}
}

func (r *RNG) Uint32() uint32 {
	x := r.s
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	r.s = x
	return uint32((x * 0x2545F4914F6CDD1D) >> 32)
}

// Intn returns a value in [0, n). Panics on n <= 0.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		panic("engine: RNG.Intn requires n > 0")
	}
	return int(r.Uint32() % uint32(n))
}

// Float32 returns a value in [0, 1].
func (r *RNG) Float32() float32 {
	return float32(r.Uint32()) / float32(^uint32(0))
}

// Range returns a value in [lo, hi].
func (r *RNG) Range(lo, hi float32) float32 {
	return lo + r.Float32()*(hi-lo)
}
