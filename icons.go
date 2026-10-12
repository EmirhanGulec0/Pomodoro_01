package main

import "math"

// Çizgi ikonlar: 24 birimlik ızgarada tanımlı, 2 birim kalınlık, yuvarlak uçlar.

const (
	icPlay = iota
	icPause
	icReset
	icSkip
	icGear
	icChart
	icSun
	icMoon
	icCompact
	icExpand
	icClose
	icPlus
	icMinus
	icPencil
	icKeyboard
)

func iconPaths(id int, cx, cy, size float64) [][]pt {
	k := size / 24
	T := func(x, y float64) pt { return pt{cx + (x-12)*k, cy + (y-12)*k} }
	sw := math.Max(1.25, 2*k)
	line := func(xy ...float64) []pt {
		out := make([]pt, 0, len(xy)/2)
		for i := 0; i+1 < len(xy); i += 2 {
			out = append(out, T(xy[i], xy[i+1]))
		}
		return out
	}
	var out [][]pt
	add := func(p []pt, closed bool) { out = append(out, stroke(p, sw, closed)...) }
	fill := func(p []pt) { out = append(out, p) }
	arc := func(ccx, ccy, r, a0, a1 float64, n int) []pt { // standart açı (radyan), y aşağı
		p := make([]pt, 0, n+1)
		for i := 0; i <= n; i++ {
			a := a0 + (a1-a0)*float64(i)/float64(n)
			p = append(p, T(ccx+r*math.Cos(a), ccy+r*math.Sin(a)))
		}
		return p
	}
	deg := math.Pi / 180

	switch id {
	case icPlay:
		tri := line(8.5, 5.5, 18.5, 12, 8.5, 18.5)
		fill(tri)
		add(tri, true)
	case icPause:
		fill(roundRectPts(cx+(6.5-12)*k, cy+(5-12)*k, 4*k, 14*k, 1.2*k))
		fill(roundRectPts(cx+(13.5-12)*k, cy+(5-12)*k, 4*k, 14*k, 1.2*k))
	case icReset:
		// soldan başlayıp alt → sağ → üst → sol üste uzanan yay, ucunda ok
		p := arc(12, 12, 8.5, 180*deg, 180*deg-315*deg, 48)
		p = append(p, T(3.5, 8))
		add(p, false)
		add(line(3.5, 3.5, 3.5, 8, 8, 8), false)
	case icSkip:
		tri := line(5.5, 5.5, 15, 12, 5.5, 18.5)
		fill(tri)
		add(tri, true)
		add(line(18.5, 5.5, 18.5, 18.5), false)
	case icGear:
		var g []pt
		for i := 0; i < 8; i++ {
			a := float64(i) * 45 * deg
			for _, s := range [][2]float64{{-17, 7.2}, {-10, 9.6}, {10, 9.6}, {17, 7.2}} {
				aa := a + s[0]*deg
				g = append(g, T(12+s[1]*math.Cos(aa), 12+s[1]*math.Sin(aa)))
			}
		}
		add(g, true)
		c := arc(12, 12, 3, 0, 2*math.Pi, 20)
		add(c[:len(c)-1], true)
	case icChart:
		add(line(3.5, 3.5, 3.5, 19, 4.2, 20.2, 5, 20.5, 20.5, 20.5), false)
		add(line(8, 16.5, 8, 13.5), false)
		add(line(12.5, 16.5, 12.5, 6), false)
		add(line(17, 16.5, 17, 10), false)
	case icSun:
		c := arc(12, 12, 4, 0, 2*math.Pi, 24)
		add(c[:len(c)-1], true)
		for i := 0; i < 8; i++ {
			a := float64(i) * 45 * deg
			add([]pt{T(12+8*math.Cos(a), 12+8*math.Sin(a)), T(12+10*math.Cos(a), 12+10*math.Sin(a))}, false)
		}
	case icMoon:
		inner := arc(16.5, 7.5, 6.364, -135*deg, -315*deg, 24)
		outer := arc(12, 12, 9, 0, 270*deg, 40)
		p := append(inner, outer[1:]...)
		add(p, true)
	case icCompact:
		add(line(20.5, 9, 20.5, 6.5, 19.8, 5.2, 18.5, 4.5, 5.5, 4.5, 4.2, 5.2, 3.5, 6.5, 3.5, 15.5, 4.2, 16.8, 5.5, 17.5, 8.5, 17.5), false)
		add(roundRectPts(cx+(12-12)*k, cy+(12.5-12)*k, 9.5*k, 7.5*k, 1.8*k), true)
	case icExpand:
		add(line(14.5, 3.5, 20.5, 3.5, 20.5, 9.5), false)
		add(line(9.5, 20.5, 3.5, 20.5, 3.5, 14.5), false)
		add(line(20.5, 3.5, 14, 10), false)
		add(line(3.5, 20.5, 10, 14), false)
	case icClose:
		add(line(6.5, 6.5, 17.5, 17.5), false)
		add(line(17.5, 6.5, 6.5, 17.5), false)
	case icPlus:
		add(line(5.5, 12, 18.5, 12), false)
		add(line(12, 5.5, 12, 18.5), false)
	case icMinus:
		add(line(5.5, 12, 18.5, 12), false)
	case icPencil:
		add(line(16.5, 3.8, 20.2, 7.5, 8.2, 19.5, 3.5, 20.5, 4.5, 15.8), true)
		add(line(14, 6.3, 17.7, 10), false)
	case icKeyboard:
		add(roundRectPts(cx+(2.5-12)*k, cy+(5.5-12)*k, 19*k, 13*k, 2*k), true)
		for _, xy := range [][2]float64{{6.5, 9.5}, {10, 9.5}, {13.5, 9.5}, {17, 9.5}} {
			add([]pt{T(xy[0], xy[1]), T(xy[0]+0.01, xy[1])}, false)
		}
		add(line(7.5, 14.5, 16.5, 14.5), false)
	}
	return out
}

// logoPaths: uygulama işareti — açık uçlu amber halka.
func logoPaths(cx, cy, r float64) [][]pt {
	w := r * 0.36
	return arcStroke(cx, cy, r-w/2, w, 0.1, 2*math.Pi*0.80)
}

// DrawAppIcon: uygulama simgesi (pencere, görev çubuğu, tepsi, dosya simgesi).
func DrawAppIcon(c *Canvas) {
	s := float64(c.W)
	for i := range c.Pix {
		c.Pix[i] = 0
	}
	top, bot := hex(0x1C2433), hex(0x0B0E15)
	pad := s * 0.04
	if s <= 20 {
		pad = 0
	}
	rad := (s - 2*pad) * 0.24
	// maske: yuvarlak köşeli kare, içi dikey gradyan
	mask := NewCanvas(c.W, c.H)
	mask.RoundRect(pad, pad, s-2*pad, s-2*pad, rad, RGB{255, 255, 255}, 1, RGB{}, 0, 0)
	for y := 0; y < c.H; y++ {
		col := top.Mix(bot, float64(y)/s)
		r, g, b := col.i()
		for x := 0; x < c.W; x++ {
			m := int(mask.Pix[y*c.W+x] & 255)
			if m > 0 {
				c.Pix[y*c.W+x] = uint32(m)<<24 | pack(r, g, b)
			}
		}
	}
	// halka (alfa kanalını koruyarak çiz)
	ring := NewCanvas(c.W, c.H)
	cx, cy := s/2, s/2
	R := s * 0.30
	w := math.Max(1.6, s*0.105)
	ring.FillPaths(arcStroke(cx, cy, R, w, 0, 2*math.Pi*0.78), RGB{255, 255, 255}, 1)
	acc := hex(0xF2A93B)
	if s >= 32 {
		ring2 := NewCanvas(c.W, c.H)
		ring2.Ring(cx, cy, R, w*0.55, RGB{255, 255, 255}, 1)
		for i := range c.Pix {
			t := float64(ring2.Pix[i]&255) / 255 * 0.14
			if t > 0 && c.Pix[i]>>24 > 0 {
				blendPixKeepA(&c.Pix[i], 255, 255, 255, t)
			}
		}
	}
	ar, ag, ab := acc.i()
	for i := range c.Pix {
		t := float64(ring.Pix[i]&255) / 255
		if t > 0 && c.Pix[i]>>24 > 0 {
			blendPixKeepA(&c.Pix[i], ar, ag, ab, t)
		}
	}
}

func blendPixKeepA(p *uint32, r, g, b int, t float64) {
	a := *p & 0xFF000000
	d := *p
	dr, dg, db := float64(d>>16&255), float64(d>>8&255), float64(d&255)
	nr := int(dr + (float64(r)-dr)*t)
	ng := int(dg + (float64(g)-dg)*t)
	nb := int(db + (float64(b)-db)*t)
	*p = a | pack(c8(nr), c8(ng), c8(nb))
}
