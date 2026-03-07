package tui

import (
	"math"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

func TestSmoothstep(t *testing.T) {
	tests := []struct {
		name         string
		edge0, edge1 float64
		x            float64
		want         float64
	}{
		{"below edge0", 0, 1, -0.5, 0},
		{"at edge0", 0, 1, 0, 0},
		{"midpoint", 0, 1, 0.5, 0.5},
		{"at edge1", 0, 1, 1, 1},
		{"above edge1", 0, 1, 1.5, 1},
		{"shifted range below", 2, 4, 1, 0},
		{"shifted range mid", 2, 4, 3, 0.5},
		{"shifted range above", 2, 4, 5, 1},
		{"quarter point", 0, 1, 0.25, 0.15625},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smoothstep(tt.edge0, tt.edge1, tt.x)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("smoothstep(%g, %g, %g) = %g, want %g", tt.edge0, tt.edge1, tt.x, got, tt.want)
			}
		})
	}
}

func TestFlashBeamCenter(t *testing.T) {
	tests := []struct {
		name        string
		frame       int
		totalFrames int
		want        float64
	}{
		{"first frame", 0, 10, -0.3},
		{"last frame", 9, 10, 2.6},
		{"single frame", 0, 1, 1.15},
		{"zero frames", 0, 0, 1.15},
		{"mid frame", 5, 11, 1.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flashBeamCenter(tt.frame, tt.totalFrames)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("flashBeamCenter(%d, %d) = %g, want %g", tt.frame, tt.totalFrames, got, tt.want)
			}
		})
	}
}

func TestFlashDiagonalValue(t *testing.T) {
	tests := []struct {
		name       string
		x, y, w, h int
		want       float64
	}{
		{"top-left corner", 0, 0, 10, 10, 0},
		{"bottom-right corner", 9, 9, 10, 10, 2},
		{"top-right corner", 9, 0, 10, 10, 1},
		{"bottom-left corner", 0, 9, 10, 10, 1},
		{"center", 5, 5, 11, 11, 1},
		{"width 1", 0, 5, 1, 10, 5.0 / 9.0},
		{"height 1", 5, 0, 10, 1, 5.0 / 9.0},
		{"width 1 height 1", 0, 0, 1, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flashDiagonalValue(tt.x, tt.y, tt.w, tt.h)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("flashDiagonalValue(%d, %d, %d, %d) = %g, want %g", tt.x, tt.y, tt.w, tt.h, got, tt.want)
			}
		})
	}
}

func TestFlashCharColor(t *testing.T) {
	dim := colorful.Color{R: 0, G: 0, B: 0}
	bright := colorful.Color{R: 1, G: 1, B: 1}

	tests := []struct {
		name    string
		d       float64
		gaining bool
		wantR   float64
	}{
		{"gaining, far left of beam", 0, true, 1},  // bright
		{"gaining, far right of beam", 2, true, 0}, // dim
		{"losing, far left of beam", 0, false, 0},  // dim
		{"losing, far right of beam", 2, false, 1}, // bright
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := flashCtx{
				beamCenter: 1.0,
				dim:        dim,
				bright:     bright,
				gaining:    tt.gaining,
			}
			got := flashCharColor(tt.d, ctx)
			if math.Abs(got.R-tt.wantR) > 0.01 {
				t.Errorf("flashCharColor(%g, gaining=%v).R = %g, want %g", tt.d, tt.gaining, got.R, tt.wantR)
			}
		})
	}
}

func TestFlashTitleColor(t *testing.T) {
	dim := colorful.Color{R: 0, G: 0, B: 0}
	bright := colorful.Color{R: 1, G: 1, B: 1}

	tests := []struct {
		name    string
		d       float64
		gaining bool
		wantR   float64
	}{
		{"gaining, far left of beam", 0, true, 1},
		{"gaining, far right of beam", 2, true, 0},
		{"losing, far left of beam", 0, false, 0},
		{"losing, far right of beam", 2, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := flashCtx{
				beamCenter:  1.0,
				titleDim:    dim,
				titleBright: bright,
				gaining:     tt.gaining,
			}
			got := flashTitleColor(tt.d, ctx)
			if math.Abs(got.R-tt.wantR) > 0.01 {
				t.Errorf("flashTitleColor(%g, gaining=%v).R = %g, want %g", tt.d, tt.gaining, got.R, tt.wantR)
			}
		})
	}
}
