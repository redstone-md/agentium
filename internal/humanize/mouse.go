package humanize

import (
	"math"
	"math/rand"
)

type MousePathBuilder struct {
	rng *rand.Rand
}

type Point struct {
	X float64
	Y float64
}

func NewMousePathBuilder(seed int64) *MousePathBuilder {
	return &MousePathBuilder{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (b *MousePathBuilder) Build(from, to Point) []Point {
	distance := math.Hypot(to.X-from.X, to.Y-from.Y)
	steps := int(distance/12) + 12
	if steps < 12 {
		steps = 12
	}

	jitter := math.Max(10, distance*0.12)
	cp1 := Point{
		X: from.X + (to.X-from.X)*0.25 + b.randOffset(jitter),
		Y: from.Y + (to.Y-from.Y)*0.10 + b.randOffset(jitter),
	}
	cp2 := Point{
		X: from.X + (to.X-from.X)*0.75 + b.randOffset(jitter),
		Y: from.Y + (to.Y-from.Y)*0.90 + b.randOffset(jitter),
	}

	path := make([]Point, 0, steps)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		path = append(path, cubicBezier(from, cp1, cp2, to, t))
	}

	return path
}

func (b *MousePathBuilder) randOffset(limit float64) float64 {
	return (b.rng.Float64()*2 - 1) * limit
}

func cubicBezier(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	return Point{
		X: uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X,
		Y: uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y,
	}
}
