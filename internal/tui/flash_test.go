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

func TestGlintPeak(t *testing.T) {
	tests := []struct {
		name   string
		d      float64
		center float64
		radius float64
		want   float64
	}{
		{"at center", 1.0, 1.0, 0.25, 1.0},
		{"at left edge", 0.75, 1.0, 0.25, 0},
		{"at right edge", 1.25, 1.0, 0.25, 0},
		{"outside left", 0.5, 1.0, 0.25, 0},
		{"outside right", 1.5, 1.0, 0.25, 0},
		{"halfway left", 0.875, 1.0, 0.25, 0.5625}, // (1 - 0.25)^2 = 0.5625
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glintPeak(tt.d, tt.center, tt.radius)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("glintPeak(%g, %g, %g) = %g, want %g", tt.d, tt.center, tt.radius, got, tt.want)
			}
		})
	}
}

func TestFlashCharColor(t *testing.T) {
	dim := colorful.Color{R: 0, G: 0, B: 0}
	bright := colorful.Color{R: 1, G: 1, B: 1}
	glint := colorful.Color{R: 1, G: 0.84, B: 0} // gold

	tests := []struct {
		name    string
		d       float64
		gaining bool
		reverse bool
		wantR   float64
	}{
		// Far from beam: no glint, pure transition.
		{"gaining, far left of beam", 0, true, false, 1},         // bright
		{"gaining, far right of beam", 2, true, false, 0},        // dim
		{"losing, far left of beam", 0, false, false, 0},         // dim
		{"losing, far right of beam", 2, false, false, 1},        // bright
		{"reverse gaining, far left of beam", 0, true, true, 0},  // dim
		{"reverse gaining, far right of beam", 2, true, true, 1}, // bright
		{"reverse losing, far left of beam", 0, false, true, 1},  // bright
		{"reverse losing, far right of beam", 2, false, true, 0}, // dim
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := flashCtx{
				beamCenter: 1.0,
				dim:        dim,
				bright:     bright,
				glint:      glint,
				gaining:    tt.gaining,
				reverse:    tt.reverse,
			}
			got := flashCharColor(tt.d, ctx)
			if math.Abs(got.R-tt.wantR) > 0.01 {
				t.Errorf("flashCharColor(%g, gaining=%v, reverse=%v).R = %g, want %g", tt.d, tt.gaining, tt.reverse, got.R, tt.wantR)
			}
		})
	}
}

func TestFlashCharColorGlintAtBeamCenter(t *testing.T) {
	dim := colorful.Color{R: 0, G: 0, B: 0}
	bright := colorful.Color{R: 0.5, G: 0.5, B: 0.5}
	glint := colorful.Color{R: 1, G: 1, B: 0} // yellow

	ctx := flashCtx{
		beamCenter: 1.0,
		dim:        dim,
		bright:     bright,
		glint:      glint,
		gaining:    true,
	}
	got := flashCharColor(1.0, ctx)

	// At beam center the glint peaks at full intensity, so the color
	// should be very close to the glint color, not the transition base.
	if got.R < 0.9 {
		t.Errorf("at beam center, R = %g, want >= 0.9 (glint dominant)", got.R)
	}
}

func TestFlashTitleColor(t *testing.T) {
	dim := colorful.Color{R: 0, G: 0, B: 0}
	bright := colorful.Color{R: 1, G: 1, B: 1}
	glint := colorful.Color{R: 1, G: 0.84, B: 0}

	tests := []struct {
		name    string
		d       float64
		gaining bool
		reverse bool
		wantR   float64
	}{
		{"gaining, far left of beam", 0, true, false, 1},
		{"gaining, far right of beam", 2, true, false, 0},
		{"losing, far left of beam", 0, false, false, 0},
		{"losing, far right of beam", 2, false, false, 1},
		{"reverse gaining, far left of beam", 0, true, true, 0},
		{"reverse gaining, far right of beam", 2, true, true, 1},
		{"reverse losing, far left of beam", 0, false, true, 1},
		{"reverse losing, far right of beam", 2, false, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := flashCtx{
				beamCenter:  1.0,
				titleDim:    dim,
				titleBright: bright,
				glint:       glint,
				gaining:     tt.gaining,
				reverse:     tt.reverse,
			}
			got := flashTitleColor(tt.d, ctx)
			if math.Abs(got.R-tt.wantR) > 0.01 {
				t.Errorf("flashTitleColor(%g, gaining=%v, reverse=%v).R = %g, want %g", tt.d, tt.gaining, tt.reverse, got.R, tt.wantR)
			}
		})
	}
}
