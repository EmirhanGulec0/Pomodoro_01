package main

import (
	"math"
	"math/rand"
)

// Arka plan stilleri: 0 düz, 1 sade (halka arkasında vurgu renginde hafif ışık), 2 şehir (+ siluet, yıldızlar).
// Arka plan bir kez önbelleğe çizilir; yalnızca boyut, tema/mod rengi ya da stil değişince yenilenir.

type bgKey struct {
	w, h, style   int
	bg0, bg1, acc [3]int
	dark          int
	cx, cy, r     int
}

func (a *App) ensureBG(cx, cy, R float64) {
	p := a.pal
	k := bgKey{a.W, a.H, a.S.Background, p.Bg0.q(), p.Bg1.q(), p.Accent.q(), int(p.Dark * 32), int(cx), int(cy), int(R)}
	if a.bg != nil && k == a.bgKey && a.bg.W == a.W && a.bg.H == a.H {
		return
	}
	a.bgKey = k
	if a.bg == nil || a.bg.W != a.W || a.bg.H != a.H {
		a.bg = NewCanvas(a.W, a.H)
	}
	a.bakeBG(cx, cy, R)
}

func (a *App) bakeBG(cx, cy, R float64) {
	p := a.pal
	W, H := a.W, a.H
	style := a.S.Background
	pix := a.bg.Pix
	glowR := R * 2.7
	inv := 1 / (glowR * glowR)
	gs := lerp(0.075, 0.12, p.Dark)
	flat := p.Bg0.Mix(p.Bg1, 0.45)
	for y := 0; y < H; y++ {
		base := flat
		if style > 0 {
			base = p.Bg0.Mix(p.Bg1, float64(y)/float64(H))
		}
		dy := float64(y) - cy
		row := y * W
		for x := 0; x < W; x++ {
			col := base
			if style > 0 {
				dx := float64(x) - cx
				if d2 := (dx*dx + dy*dy) * inv; d2 < 1 {
					g := 1 - d2
					col = col.Mix(p.Accent, g*g*gs)
				}
			}
			n := dither(x, y)
			pix[row+x] = pack(c8(int(col.R+0.5+n)), c8(int(col.G+0.5+n)), c8(int(col.B+0.5+n)))
		}
	}
	if style == 2 {
		a.bakeCity()
	}
}

func (a *App) bakeCity() {
	p := a.pal
	c := a.bg
	W, H := float64(a.W), float64(a.H)
	u := a.u
	rng := rand.New(rand.NewSource(7))

	// yıldızlar (yalnızca koyu temada görünür; ayrı tohum, siluet temadan bağımsız kalsın)
	if p.Dark > 0.02 {
		sr := rand.New(rand.NewSource(11))
		n := int(W * H / 4200)
		for i := 0; i < n; i++ {
			x := sr.Float64() * W
			y := sr.Float64() * H * 0.78
			b := (0.12 + 0.5*math.Pow(sr.Float64(), 2.4)) * p.Dark * (1 - y/(H*0.95))
			col := RGB{220, 228, 255}
			c.Disc(x, y, 0.55*u+0.25, col, b)
		}
	}

	// iki katmanlı siluet: uzak (zemine yakın ton) ve yakın (koyu)
	far := p.Bg1.Mix(p.Text3, lerp(0.09, 0.10, p.Dark))
	near := p.Bg1.Mix(p.Text3, lerp(0.17, 0.0, p.Dark)).Mix(RGB{0, 0, 0}, 0.35*p.Dark)
	layers := []struct {
		col        RGB
		hMin, hMax float64
		wMin, wMax float64
		lights     bool
	}{
		{far, 0.06, 0.125, 34, 70, false},
		{near, 0.03, 0.08, 46, 96, true},
	}
	lamp := hex(0xF2C27A)
	for _, L := range layers {
		x := -10 * u
		for x < W {
			w := (L.wMin + rng.Float64()*(L.wMax-L.wMin)) * u
			h := H * (L.hMin + rng.Float64()*(L.hMax-L.hMin))
			top := H - h
			c.FillRect(x, top, x+w-1*u, H, L.col, 1)
			switch rng.Intn(5) {
			case 0: // anten
				c.FillRect(x+w/2-0.75*u, top-14*u, x+w/2+0.75*u, top, L.col, 1)
			case 1: // basamaklı taç
				c.FillRect(x+w*0.2, top-8*u, x+w*0.8-1*u, top, L.col, 1)
			}
			if L.lights {
				for wy := top + 9*u; wy < H-8*u; wy += 13 * u {
					for wx := x + 7*u; wx < x+w-9*u; wx += 10 * u {
						if rng.Float64() < 0.10 && p.Dark > 0.02 {
							c.FillRect(wx, wy, wx+3*u, wy+4*u, lamp, 0.38*p.Dark)
						}
					}
				}
			}
			x += w
		}
	}
}
