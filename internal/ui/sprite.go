package ui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed sprites/Idle-Anim.png
var idlePNG []byte

//go:embed sprites/Walk-Anim.png
var walkPNG []byte

// SpriteAnim identifies which animation to play.
type SpriteAnim int

const (
	SpriteIdle SpriteAnim = iota
	SpriteWalk
)

// spriteSheet holds pre-rendered frames for one animation.
type spriteSheet struct {
	frames   []string
	interval time.Duration
}

// SpriteModel is a Bubble Tea sub-model that renders an animated sprite.
type SpriteModel struct {
	sheets map[SpriteAnim]*spriteSheet
	anim   SpriteAnim
	frame  int
}

type spriteTickMsg struct{}

// NewSprite creates a SpriteModel with pre-rendered frames from embedded PNGs.
func NewSprite() SpriteModel {
	m := SpriteModel{
		sheets: make(map[SpriteAnim]*spriteSheet),
		anim:   SpriteIdle,
	}

	// Idle: 32x40 per frame, 2 frames, row 1 (south-east) — auto-crop transparent padding
	idleFrames := cropFrames(decodeSheet(idlePNG, 32, 40, 2, 1))
	idleRendered := make([]string, len(idleFrames))
	for i, img := range idleFrames {
		idleRendered[i] = renderFrame(img)
	}
	m.sheets[SpriteIdle] = &spriteSheet{
		frames:   idleRendered,
		interval: 800 * time.Millisecond,
	}

	// Walk: 32x40 per frame, 4 frames, row 1 (south-east) — auto-crop transparent padding
	walkFrames := cropFrames(decodeSheet(walkPNG, 32, 40, 4, 1))
	walkRendered := make([]string, len(walkFrames))
	for i, img := range walkFrames {
		walkRendered[i] = renderFrame(img)
	}
	m.sheets[SpriteWalk] = &spriteSheet{
		frames:   walkRendered,
		interval: 300 * time.Millisecond,
	}

	return m
}

// SetAnim switches the active animation and resets to frame 0.
func (m *SpriteModel) SetAnim(anim SpriteAnim) {
	if m.anim != anim {
		m.anim = anim
		m.frame = 0
	}
}

// Tick returns the tea.Cmd that schedules the next animation frame.
func (m SpriteModel) Tick() tea.Cmd {
	sheet := m.sheets[m.anim]
	if sheet == nil {
		return nil
	}
	return tea.Tick(sheet.interval, func(time.Time) tea.Msg {
		return spriteTickMsg{}
	})
}

// Update handles spriteTickMsg to advance frames.
func (m SpriteModel) Update(msg tea.Msg) (SpriteModel, tea.Cmd) {
	if _, ok := msg.(spriteTickMsg); ok {
		sheet := m.sheets[m.anim]
		if sheet != nil && len(sheet.frames) > 0 {
			m.frame = (m.frame + 1) % len(sheet.frames)
		}
		return m, m.Tick()
	}
	return m, nil
}

// View returns the current frame as a pre-rendered string.
func (m SpriteModel) View() string {
	sheet := m.sheets[m.anim]
	if sheet == nil || len(sheet.frames) == 0 {
		return ""
	}
	return sheet.frames[m.frame]
}

// decodeSheet loads a PNG from bytes and extracts frames from the given direction row.
func decodeSheet(data []byte, frameW, frameH, numFrames, row int) []image.Image {
	reader := bytes.NewReader(data)
	sheet, _, err := image.Decode(reader)
	if err != nil {
		return make([]image.Image, numFrames)
	}

	yOff := row * frameH
	frames := make([]image.Image, numFrames)
	for i := 0; i < numFrames; i++ {
		rect := image.Rect(i*frameW, yOff, (i+1)*frameW, yOff+frameH)
		if sub, ok := sheet.(interface {
			SubImage(r image.Rectangle) image.Image
		}); ok {
			frames[i] = sub.SubImage(rect)
		}
	}
	return frames
}

// renderFrame converts an image to a terminal string using half-block characters.
func renderFrame(img image.Image) string {
	if img == nil {
		return ""
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	var sb strings.Builder
	for y := bounds.Min.Y; y < bounds.Min.Y+h; y += 2 {
		for x := bounds.Min.X; x < bounds.Min.X+w; x++ {
			topC := img.At(x, y)
			botC := img.At(x, y+1)
			topTrans := isTransparent(topC)
			botTrans := isTransparent(botC)

			switch {
			case topTrans && botTrans:
				sb.WriteString(" ")
			case !topTrans && botTrans:
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex(topC)))
				sb.WriteString(style.Render("\u2580"))
			case topTrans && !botTrans:
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex(botC)))
				sb.WriteString(style.Render("\u2584"))
			default:
				style := lipgloss.NewStyle().
					Foreground(lipgloss.Color(colorHex(botC))).
					Background(lipgloss.Color(colorHex(topC)))
				sb.WriteString(style.Render("\u2584"))
			}
		}
		if y+2 < bounds.Min.Y+h {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// cropFrames finds the union of visible (non-transparent) bounds across all
// frames and crops them uniformly so the sprite is as compact as possible
// without jitter between frames. Ensures height is even for half-block rendering.
func cropFrames(frames []image.Image) []image.Image {
	// Find union of visible bounds (relative to each frame's origin)
	minRX, minRY := 9999, 9999
	maxRX, maxRY := 0, 0
	for _, img := range frames {
		if img == nil {
			continue
		}
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if !isTransparent(img.At(x, y)) {
					rx, ry := x-b.Min.X, y-b.Min.Y
					if rx < minRX {
						minRX = rx
					}
					if rx > maxRX {
						maxRX = rx
					}
					if ry < minRY {
						minRY = ry
					}
					if ry > maxRY {
						maxRY = ry
					}
				}
			}
		}
	}
	if maxRX <= minRX || maxRY <= minRY {
		return frames
	}
	// Ensure even height for half-block rendering
	cropH := maxRY + 1 - minRY
	if cropH%2 != 0 {
		maxRY++
	}
	// Crop each frame to the union bounds
	cropped := make([]image.Image, len(frames))
	for i, img := range frames {
		if img == nil {
			continue
		}
		b := img.Bounds()
		rect := image.Rect(b.Min.X+minRX, b.Min.Y+minRY, b.Min.X+maxRX+1, b.Min.Y+maxRY+1)
		if sub, ok := img.(interface {
			SubImage(r image.Rectangle) image.Image
		}); ok {
			cropped[i] = sub.SubImage(rect)
		} else {
			cropped[i] = img
		}
	}
	return cropped
}

func colorHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func isTransparent(c color.Color) bool {
	_, _, _, a := c.RGBA()
	return a < 0x8000
}
