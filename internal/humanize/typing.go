package humanize

import (
	"math"
	"math/rand"
	"time"
)

type TypingDelays struct {
	rng *rand.Rand
}

func NewTypingDelays(seed int64) *TypingDelays {
	return &TypingDelays{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (d *TypingDelays) Next(base time.Duration) time.Duration {
	mean := float64(base)
	stddev := mean / 2
	value := d.rng.NormFloat64()*stddev + mean
	value = math.Max(float64(20*time.Millisecond), value)
	return time.Duration(value)
}
