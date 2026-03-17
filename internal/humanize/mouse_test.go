package humanize

import "testing"

func TestMousePathBuilderBuild(t *testing.T) {
	builder := NewMousePathBuilder(42)
	path := builder.Build(Point{X: 10, Y: 15}, Point{X: 240, Y: 180})

	if len(path) < 12 {
		t.Fatalf("expected at least 12 points, got %d", len(path))
	}

	last := path[len(path)-1]
	if last.X != 240 || last.Y != 180 {
		t.Fatalf("expected path to end at destination, got (%f,%f)", last.X, last.Y)
	}
}
