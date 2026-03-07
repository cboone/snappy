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
	totalWidth   int
	panelY       int // panel's top-left Y in global screen coordinates
	screenHeight int // total terminal height
	beamCenter   float64
	dim          colorful.Color // unfocused border color
	bright       colorful.Color // focused border color
	glint        colorful.Color // bright peak swept across (sunlight on glass)
	titleDim     colorful.Color // unfocused title foreground
	titleBright  colorful.Color // focused title foreground
	gaining      bool           // true = dim-to-bright wipe, false = bright-to-dim
	reverse      bool           // sweep bottom-right to top-left
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

// flashVerticalValue computes the normalized vertical position for a border
// row at localY within a panel whose top edge is at panelY in global screen
// coordinates. Returns a value in [0, 2] where 0 is the top of the screen
// and 2 is the bottom, matching the beam center range.
func flashVerticalValue(localY, panelY, screenHeight int) float64 {
	globalY := panelY + localY
	if screenHeight <= 1 {
		return 0
	}
	return 2.0 * float64(globalY) / float64(screenHeight-1)
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
// the vertical flash animation. The title is split into three parts:
//   - titlePrefix: pre-styled text that keeps its own coloring (e.g., dot indicator)
//   - titleLabel: raw text that gets per-character flash coloring (e.g., "snappy")
//   - titleSuffix: pre-styled text that keeps its own coloring (e.g., spinner)
//
// panelY is the panel's top-left Y in global screen coordinates.
// screenHeight is the total terminal height, used to normalize the vertical
// sweep so the glint flows continuously between panels.
//
// Content is padded and wrapped with manually-built borders, bypassing
// Lipgloss border rendering to allow individual character coloring.
func renderFlashBorders(content, titlePrefix, titleLabel, titleSuffix string, contentWidth int, flash flashState, gaining bool, s modelStyles, panelY, screenHeight int) string {
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
		totalWidth:   contentWidth + 4,
		panelY:       panelY,
		screenHeight: screenHeight,
		beamCenter:   beamCenter,
		dim:          s.flashDim,
		bright:       s.flashBright,
		glint:        s.flashGlint,
		titleDim:     s.flashTitleDim,
		titleBright:  s.flashTitleBright,
		gaining:      gaining,
		reverse:      flash.reverse,
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
		d := flashVerticalValue(i+1, ctx.panelY, ctx.screenHeight)
		c := flashCharColor(d, ctx)
		styled := lipgloss.NewStyle().Foreground(flashColor(c)).Render

		out.WriteString(styled(border.Left))
		out.WriteString(line)
		out.WriteString(styled(border.Right))
		if i < len(bodyLines)-1 {
			out.WriteByte('\n')
		}
	}
	out.WriteByte('\n')

	buildFlashBottomBorder(&out, len(bodyLines)+2, border, ctx)

	return out.String()
}

// buildFlashTopBorder writes the top border line with uniform row coloring.
// The title prefix and suffix are written as-is (pre-styled). The title label
// is rendered per-character with the row's flash-interpolated title color.
func buildFlashTopBorder(b *strings.Builder, titlePrefix, titleLabel, titleSuffix string, titleWidth int, border lipgloss.Border, ctx flashCtx) {
	d := flashVerticalValue(0, ctx.panelY, ctx.screenHeight)
	borderStyle := lipgloss.NewStyle().Foreground(flashColor(flashCharColor(d, ctx)))
	titleColor := flashTitleColor(d, ctx)

	totalFill := max(ctx.totalWidth-titleWidth-4, 0)
	leftFill := totalFill / 2
	rightFill := totalFill - leftFill

	b.WriteString(borderStyle.Render(border.TopLeft))
	b.WriteString(borderStyle.Render(strings.Repeat(border.Top, leftFill)))
	b.WriteString(borderStyle.Render(" "))

	// Title prefix (pre-styled, e.g., dot indicator).
	b.WriteString(titlePrefix)

	// Title label (uniform row color from vertical position).
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(flashColor(titleColor))
	b.WriteString(titleStyle.Render(titleLabel))

	// Title suffix (pre-styled, e.g., spinner).
	b.WriteString(titleSuffix)

	b.WriteString(borderStyle.Render(" "))
	b.WriteString(borderStyle.Render(strings.Repeat(border.Top, rightFill)))
	b.WriteString(borderStyle.Render(border.TopRight))
}

// buildFlashBottomBorder writes the bottom border line with uniform row coloring.
func buildFlashBottomBorder(b *strings.Builder, panelHeight int, border lipgloss.Border, ctx flashCtx) {
	d := flashVerticalValue(panelHeight-1, ctx.panelY, ctx.screenHeight)
	borderStyle := lipgloss.NewStyle().Foreground(flashColor(flashCharColor(d, ctx)))

	b.WriteString(borderStyle.Render(border.BottomLeft))
	b.WriteString(borderStyle.Render(strings.Repeat(border.Bottom, ctx.totalWidth-2)))
	b.WriteString(borderStyle.Render(border.BottomRight))
}
