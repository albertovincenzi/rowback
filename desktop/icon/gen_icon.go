// gen_icon renders a 1024×1024 app icon for Partial Restore using signed-distance
// fields for crisp anti-aliased edges (no external assets). Output: icon_1024.png.
//
// Motif: a database cylinder on a blue squircle, with one "extracted" row slice
// lifted out and a green restore badge — i.e. pulling specific rows back out.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const S = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, S, S))

	// Squircle background mask + diagonal blue gradient.
	bgR := 0.2235 * S
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := sdRoundRect(fx-S/2, fy-S/2, S/2-2, S/2-2, bgR)
			a := cov(d)
			if a <= 0 {
				continue
			}
			t := (fx/S + fy/S) / 2
			top := rgb(0x35, 0x84, 0xff)
			bot := rgb(0x12, 0x46, 0xc4)
			c := lerp(top, bot, t)
			// subtle top sheen
			sheen := math.Max(0, 1-fy/(S*0.62))
			c = lerp(c, rgb(0x9a, 0xc4, 0xff), 0.10*sheen)
			set(img, x, y, c, a)
		}
	}

	// Database cylinder geometry.
	cx := float64(S) * 0.50
	cyTop := float64(S) * 0.34
	cyBot := float64(S) * 0.66
	rx := float64(S) * 0.235
	ry := float64(S) * 0.072
	white := rgb(0xf3, 0xf7, 0xfd)
	shadow := rgb(0x0c, 0x2c, 0x72)

	// soft drop shadow under the cylinder
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := sdEllipse(fx-cx, fy-(cyBot+24), rx*1.02, ry*1.15)
			a := cov(d) * 0.35
			if a > 0 {
				blend(img, x, y, shadow, a)
			}
		}
	}

	// cylinder body + caps (skip the lifted top slice region; drawn separately)
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			body := sdRoundRect(fx-cx, fy-(cyTop+cyBot)/2, rx, (cyBot-cyTop)/2, 2)
			// body is the rectangle between the two cap centers
			bodyRect := -math.Max(math.Abs(fx-cx)-rx, math.Abs(fy-(cyTop+cyBot)/2)-(cyBot-cyTop)/2)
			capB := -sdEllipse(fx-cx, fy-cyBot, rx, ry)
			d := -math.Max(math.Max(bodyRect, capB), -1e9)
			_ = body
			a := cov(d)
			if a > 0 {
				c := white
				// horizontal band shading for volume
				shade := 0.08 * math.Sin((fy-cyTop)/(cyBot-cyTop)*math.Pi)
				c = lerp(c, rgb(0xc8, 0xd6, 0xee), math.Max(0, shade*4))
				blend(img, x, y, c, a)
			}
		}
	}

	// two divider bands on the cylinder
	for _, by := range []float64{cyTop + (cyBot-cyTop)*0.40, cyTop + (cyBot-cyTop)*0.66} {
		drawBand(img, cx, by, rx, ry, rgb(0x6f, 0x90, 0xc8))
	}

	// the "extracted" top slice, lifted up and to the right (highlighted)
	lift := float64(S) * 0.085
	sx := cx + float64(S)*0.045
	syTop := cyTop - lift
	accent := rgb(0x2f, 0xd9, 0x7a) // green = restored
	drawDisc(img, sx, syTop, rx*0.92, ry*0.92, accent, rgb(0x16, 0x9a, 0x53))
	// connector hint (dotted-ish) between hole and slice
	drawDisc(img, cx, cyTop, rx, ry, rgb(0xd7, 0xe2, 0xf5), rgb(0x9a, 0xb2, 0xd8))

	// restore badge: green circle with up-arrow, bottom-right
	bx, by, br := float64(S)*0.74, float64(S)*0.74, float64(S)*0.165
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := sdCircle(fx-bx, fy-by, br)
			a := cov(d)
			if a > 0 {
				ring := cov(sdCircle(fx-bx, fy-by, br) * -1) // inner
				_ = ring
				blend(img, x, y, accent, a)
			}
			// white up-arrow glyph
			if arrowUp(fx-bx, fy-by, br) {
				ga := cov(sdCircle(fx-bx, fy-by, br))
				if ga > 0 {
					blend(img, x, y, rgb(0xff, 0xff, 0xff), ga)
				}
			}
		}
	}

	f, err := os.Create("icon_1024.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// ---- drawing helpers ----

func drawBand(img *image.RGBA, cx, cy, rx, ry float64, c color.RGBA) {
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			outer := -sdEllipse(fx-cx, fy-cy, rx, ry)
			a := cov(outer) * 0.5
			if a > 0 {
				blend(img, x, y, c, a*0.4)
			}
		}
	}
}

func drawDisc(img *image.RGBA, cx, cy, rx, ry float64, fill, edge color.RGBA) {
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := sdEllipse(fx-cx, fy-cy, rx, ry)
			a := cov(d)
			if a > 0 {
				c := fill
				if d > -6 { // edge ring
					c = lerp(fill, edge, 0.6)
				}
				blend(img, x, y, c, a)
			}
		}
	}
}

func arrowUp(px, py, r float64) bool {
	// vertical shaft
	shaftW := r * 0.16
	if px > -shaftW && px < shaftW && py > -r*0.06 && py < r*0.46 {
		return true
	}
	// triangular head pointing UP: a point at the top, widening downward
	headTop := -r * 0.52
	headBot := -r * 0.02
	headHalf := r * 0.40
	if py >= headTop && py <= headBot {
		half := (py - headTop) / (headBot - headTop) * headHalf
		if px > -half && px < half {
			return true
		}
	}
	return false
}

// ---- SDF primitives (negative = inside) ----

func sdCircle(px, py, r float64) float64 { return math.Hypot(px, py) - r }

func sdEllipse(px, py, rx, ry float64) float64 {
	// approximate ellipse SDF (good enough at this scale)
	return (math.Hypot(px/rx, py/ry) - 1) * math.Min(rx, ry)
}

func sdRoundRect(px, py, hw, hh, r float64) float64 {
	qx := math.Abs(px) - hw + r
	qy := math.Abs(py) - hh + r
	ax := math.Max(qx, 0)
	ay := math.Max(qy, 0)
	return math.Hypot(ax, ay) + math.Min(math.Max(qx, qy), 0) - r
}

// cov maps a signed distance to ~1px anti-aliased coverage.
func cov(d float64) float64 {
	a := 0.5 - d
	if a < 0 {
		return 0
	}
	if a > 1 {
		return 1
	}
	return a
}

// ---- color helpers ----

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func lerp(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		255,
	}
}

func set(img *image.RGBA, x, y int, c color.RGBA, a float64) {
	c.A = uint8(a * 255)
	img.SetRGBA(x, y, c)
}

func blend(img *image.RGBA, x, y int, c color.RGBA, a float64) {
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	o := img.RGBAAt(x, y)
	img.SetRGBA(x, y, color.RGBA{
		uint8(float64(c.R)*a + float64(o.R)*(1-a)),
		uint8(float64(c.G)*a + float64(o.G)*(1-a)),
		uint8(float64(c.B)*a + float64(o.B)*(1-a)),
		255,
	})
}
