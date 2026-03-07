package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	flashTotalFrames = 20
)

// flashState tracks the border flash animation when focus changes.
// Both the panel gaining focus and the panel losing focus animate
// simultaneously with opposing wipe directions.
type flashState struct {
	active      bool
	gainPanel   int // panel gaining focus (wipe: dim -> bright)
	losePanel   int // panel losing focus (wipe: bright -> dim)
	frame       int
	totalFrames int
	id          uint64
	reverse     bool // sweep bottom-right to top-left (shift-tab)
}

// flashCtx holds precomputed values for rendering a single flash frame,
// reducing parameter threading across border-building functions.
type flashCtx struct {
	totalWidth  int
	totalHeight int
	beamCenter  float64
	dim         colorful.Color // unfocused border color
	bright      colorful.Color // focused border color
	glint       colorful.Color // bright peak swept across (sunlight on glass)
	titleDim    colorful.Color // unfocused title foreground
	titleBright colorful.Color // focused title foreground
	gaining     bool           // true = dim-to-bright wipe, false = bright-to-dim
	reverse     bool           // sweep bottom-right to top-left
}

// flashBeamCenter returns the beam center position in diagonal-space for the
// given animation frame. The diagonal range is [0, 2] (top-left to
// bottom-right). The beam overshoots on both ends for smooth entry/exit.
func flashBeamCenter(frame, totalFrames int) float64 {
	const (
		start = -0.3
		end   = 2.6
	)
	if totalFrames <= 1 {
		return (start + end) / 2
	}
	progress := float64(frame) / float64(totalFrames-1)
	return start + progress*(end-start)
}

// flashDiagonalValue computes the normalized diagonal position for a border
// character at (x, y) within a panel of the given total dimensions (including
// borders). Returns a value in [0, 2] where 0 is the top-left corner and 2
// is the bottom-right corner.
func flashDiagonalValue(x, y, width, height int) float64 {
	var xNorm, yNorm float64
	if width > 1 {
		xNorm = float64(x) / float64(width-1)
	}
	if height > 1 {
		yNorm = float64(y) / float64(height-1)
	}
	return xNorm + yNorm
}

// smoothstep performs Hermite interpolation between 0 and 1 when x is
// between edge0 and edge1. Clamps to 0 below edge0 and 1 above edge1.
func smoothstep(edge0, edge1, x float64) float64 {
	t := (x - edge0) / (edge1 - edge0)
	t = max(0, min(1, t))
	return t * t * (3 - 2*t)
}

// glintPeak returns the intensity of the glint highlight at diagonal position
// d, given a beam center and radius. Uses a quartic bump kernel for a smooth
// peak that reaches 1.0 at the center and 0.0 at the edges.
func glintPeak(d, center, radius float64) float64 {
	t := (d - center) / radius
	if t < -1 || t > 1 {
		return 0
	}
	v := 1 - t*t
	return v * v
}

// flashCharColor computes the color for a border character. The base
// transition sweeps between dim (unfocused) and bright (focused) colors.
// A narrow glint highlight is overlaid at the beam center, creating a
// sunlight-on-glass shimmer effect.
func flashCharColor(d float64, ctx flashCtx) colorful.Color {
	const (
		transitionHW = 0.15
		glintRadius  = 0.25
	)
	t := smoothstep(ctx.beamCenter-transitionHW, ctx.beamCenter+transitionHW, d)
	if ctx.reverse {
		t = 1 - t
	}

	var base colorful.Color
	if ctx.gaining {
		base = ctx.bright.BlendLab(ctx.dim, t)
	} else {
		base = ctx.dim.BlendLab(ctx.bright, t)
	}

	g := glintPeak(d, ctx.beamCenter, glintRadius)
	if g > 0 {
		return base.BlendRgb(ctx.glint, g)
	}
	return base
}

// flashTitleColor computes the foreground color for a title character using
// the same transition and glint logic as border characters.
func flashTitleColor(d float64, ctx flashCtx) colorful.Color {
	const (
		transitionHW = 0.15
		glintRadius  = 0.25
	)
	t := smoothstep(ctx.beamCenter-transitionHW, ctx.beamCenter+transitionHW, d)
	if ctx.reverse {
		t = 1 - t
	}

	var base colorful.Color
	if ctx.gaining {
		base = ctx.titleBright.BlendLab(ctx.titleDim, t)
	} else {
		base = ctx.titleDim.BlendLab(ctx.titleBright, t)
	}

	g := glintPeak(d, ctx.beamCenter, glintRadius)
	if g > 0 {
		return base.BlendRgb(ctx.glint, g)
	}
	return base
}

// flashColor converts a go-colorful color to a Lipgloss-compatible color
// by calling lipgloss.Color with the hex representation.
func flashColor(c colorful.Color) color.Color {
	return lipgloss.Color(c.Hex())
}

// renderFlashBorders renders a panel with per-character colored borders for
// the diagonal flash animation. The title is split into three parts:
//   - titlePrefix: pre-styled text that keeps its own coloring (e.g., dot indicator)
//   - titleLabel: raw text that gets per-character flash coloring (e.g., "snappy")
//   - titleSuffix: pre-styled text that keeps its own coloring (e.g., spinner)
//
// Content is padded and wrapped with manually-built borders, bypassing
// Lipgloss border rendering to allow individual character coloring.
func renderFlashBorders(content, titlePrefix, titleLabel, titleSuffix string, contentWidth int, flash flashState, gaining bool, s modelStyles) string {
	padStyle := lipgloss.NewStyle().Padding(0, 1).Width(contentWidth + 2)
	padded := padStyle.Render(content)
	bodyLines := strings.Split(padded, "\n")

	beamCenter := flashBeamCenter(flash.frame, flash.totalFrames)
	if !gaining {
		beamCenter -= 0.3 // losing panel trails behind
	}
	if flash.reverse {
		beamCenter = 2.3 - beamCenter
	}

	ctx := flashCtx{
		totalWidth:  contentWidth + 4,
		totalHeight: len(bodyLines) + 2,
		beamCenter:  beamCenter,
		dim:         s.flashDim,
		bright:      s.flashBright,
		glint:       s.flashGlint,
		titleDim:    s.flashTitleDim,
		titleBright: s.flashTitleBright,
		gaining:     gaining,
		reverse:     flash.reverse,
	}

	// Compute full title width for truncation and centering.
	prefixW := lipgloss.Width(titlePrefix)
	suffixW := lipgloss.Width(titleSuffix)
	fullTitleWidth := prefixW + len(titleLabel) + suffixW

	// Truncate label if necessary, matching borderTitle behavior.
	if maxTitle := ctx.totalWidth - 4; maxTitle > 0 && fullTitleWidth > maxTitle {
		avail := maxTitle - prefixW - suffixW
		if avail > 0 {
			titleLabel = ansi.Truncate(titleLabel, avail, "")
		} else {
			titleLabel = ""
		}
		fullTitleWidth = prefixW + len(titleLabel) + suffixW
	}

	border := lipgloss.RoundedBorder()

	var out strings.Builder

	buildFlashTopBorder(&out, titlePrefix, titleLabel, titleSuffix, fullTitleWidth, border, ctx)
	out.WriteByte('\n')

	for i, line := range bodyLines {
		y := i + 1
		leftD := flashDiagonalValue(0, y, ctx.totalWidth, ctx.totalHeight)
		rightD := flashDiagonalValue(ctx.totalWidth-1, y, ctx.totalWidth, ctx.totalHeight)
		leftC := flashCharColor(leftD, ctx)
		rightC := flashCharColor(rightD, ctx)

		out.WriteString(lipgloss.NewStyle().Foreground(flashColor(leftC)).Render(border.Left))
		out.WriteString(line)
		out.WriteString(lipgloss.NewStyle().Foreground(flashColor(rightC)).Render(border.Right))
		if i < len(bodyLines)-1 {
			out.WriteByte('\n')
		}
	}
	out.WriteByte('\n')

	buildFlashBottomBorder(&out, border, ctx)

	return out.String()
}

// buildFlashTopBorder writes the top border line with per-character coloring.
// The title prefix and suffix are written as-is (pre-styled). The title label
// is rendered per-character with flash-interpolated foreground colors.
func buildFlashTopBorder(b *strings.Builder, titlePrefix, titleLabel, titleSuffix string, titleWidth int, border lipgloss.Border, ctx flashCtx) {
	totalFill := max(ctx.totalWidth-titleWidth-4, 0)
	leftFill := totalFill / 2
	rightFill := totalFill - leftFill

	y := 0
	x := 0

	writeFlashChar(b, border.TopLeft, x, y, ctx)
	x++

	for range leftFill {
		writeFlashChar(b, border.Top, x, y, ctx)
		x++
	}

	writeFlashChar(b, " ", x, y, ctx)
	x++

	// Title prefix (pre-styled, e.g., dot indicator).
	b.WriteString(titlePrefix)
	x += lipgloss.Width(titlePrefix)

	// Title label (per-character flash coloring).
	for _, r := range titleLabel {
		d := flashDiagonalValue(x, y, ctx.totalWidth, ctx.totalHeight)
		tc := flashTitleColor(d, ctx)
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(flashColor(tc)).Render(string(r)))
		x++
	}

	// Title suffix (pre-styled, e.g., spinner).
	b.WriteString(titleSuffix)
	x += lipgloss.Width(titleSuffix)

	writeFlashChar(b, " ", x, y, ctx)
	x++

	for range rightFill {
		writeFlashChar(b, border.Top, x, y, ctx)
		x++
	}

	writeFlashChar(b, border.TopRight, x, y, ctx)
}

// buildFlashBottomBorder writes the bottom border line with per-character coloring.
func buildFlashBottomBorder(b *strings.Builder, border lipgloss.Border, ctx flashCtx) {
	y := ctx.totalHeight - 1

	writeFlashChar(b, border.BottomLeft, 0, y, ctx)

	for x := 1; x < ctx.totalWidth-1; x++ {
		writeFlashChar(b, border.Bottom, x, y, ctx)
	}

	writeFlashChar(b, border.BottomRight, ctx.totalWidth-1, y, ctx)
}

// writeFlashChar writes a single character with flash-computed coloring.
func writeFlashChar(b *strings.Builder, char string, x, y int, ctx flashCtx) {
	d := flashDiagonalValue(x, y, ctx.totalWidth, ctx.totalHeight)
	c := flashCharColor(d, ctx)
	b.WriteString(lipgloss.NewStyle().Foreground(flashColor(c)).Render(char))
}
