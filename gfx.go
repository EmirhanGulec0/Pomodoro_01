package main

import (
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Yazılım çizim katmanı (yalnızca standart kütüphane).
// Piksel biçimi 0x00RRGGBB — Windows 32bpp DIB ile birebir uyumlu.
// ---------------------------------------------------------------------------

type pt struct{ x, y float64 }

type RGB struct{ R, G, B float64 }

func hex(h uint32) RGB { return RGB{float64(h >> 16 & 255), float64(h >> 8 & 255), float64(h & 255)} }

func (c RGB) Mix(o RGB, t float64) RGB {
	return RGB{c.R + (o.R-c.R)*t, c.G + (o.G-c.G)*t, c.B + (o.B-c.B)*t}
}

func (c RGB) i() (int, int, int) {
	return c8(int(c.R + 0.5)), c8(int(c.G + 0.5)), c8(int(c.B + 0.5))
}

func (c RGB) q() [3]int { r, g, b := c.i(); return [3]int{r, g, b} }

func pack(r, g, b int) uint32 { return uint32(r)<<16 | uint32(g)<<8 | uint32(b) }

func c8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func smoothstep(e0, e1, x float64) float64 {
	t := clampf((x-e0)/(e1-e0), 0, 1)
	return t * t * (3 - 2*t)
}

func easeOut(t float64) float64 { t = clampf(t, 0, 1); return 1 - (1-t)*(1-t)*(1-t) }

func easeInOut(t float64) float64 {
	t = clampf(t, 0, 1)
	if t < 0.5 {
		return 4 * t * t * t
	}
	f := -2*t + 2
	return 1 - f*f*f/2
}

func blendPix(p *uint32, r, g, b, a int) {
	if a <= 0 {
		return
	}
	if a >= 255 {
		*p = pack(r, g, b)
		return
	}
	a += a >> 7
	d := *p
	dr, dg, db := int(d>>16&255), int(d>>8&255), int(d&255)
	dr += ((r - dr) * a) >> 8
	dg += ((g - dg) * a) >> 8
	db += ((b - db) * a) >> 8
	*p = pack(dr, dg, db)
}

func addPix(p *uint32, r, g, b int) {
	d := *p
	nr := int(d>>16&255) + r
	ng := int(d>>8&255) + g
	nb := int(d&255) + b
	if nr > 255 {
		nr = 255
	}
	if ng > 255 {
		ng = 255
	}
	if nb > 255 {
		nb = 255
	}
	*p = pack(nr, ng, nb)
}

type edge struct {
	x0, y0, y1, dxdy float64
	dir              int
}

type cross struct {
	x float64
	d int
}

type Canvas struct {
	W, H   int
	Pix    []uint32
	cov    []float32
	edges  []edge
	active []int
	xs     []cross
}

func NewCanvas(w, h int) *Canvas { return NewCanvasOn(w, h, make([]uint32, w*h)) }

func NewCanvasOn(w, h int, pix []uint32) *Canvas {
	return &Canvas{W: w, H: h, Pix: pix, cov: make([]float32, w+4)}
}

func (c *Canvas) bounds(x0, y0, x1, y1 float64) (int, int, int, int) {
	ix0, iy0 := int(math.Floor(x0)), int(math.Floor(y0))
	ix1, iy1 := int(math.Ceil(x1)), int(math.Ceil(y1))
	if ix0 < 0 {
		ix0 = 0
	}
	if iy0 < 0 {
		iy0 = 0
	}
	if ix1 > c.W-1 {
		ix1 = c.W - 1
	}
	if iy1 > c.H-1 {
		iy1 = c.H - 1
	}
	return ix0, iy0, ix1, iy1
}

// FillRect: tam piksel dikdörtgen.
func (c *Canvas) FillRect(x0, y0, x1, y1 float64, col RGB, a float64) {
	X0, Y0 := int(math.Round(x0)), int(math.Round(y0))
	X1, Y1 := int(math.Round(x1)), int(math.Round(y1))
	if X0 < 0 {
		X0 = 0
	}
	if Y0 < 0 {
		Y0 = 0
	}
	if X1 > c.W {
		X1 = c.W
	}
	if Y1 > c.H {
		Y1 = c.H
	}
	A := int(a * 255)
	if A <= 0 || X0 >= X1 {
		return
	}
	r, g, b := col.i()
	v := pack(r, g, b)
	for y := Y0; y < Y1; y++ {
		row := c.Pix[y*c.W : y*c.W+c.W]
		if A >= 255 {
			for x := X0; x < X1; x++ {
				row[x] = v
			}
		} else {
			for x := X0; x < X1; x++ {
				blendPix(&row[x], r, g, b, A)
			}
		}
	}
}

func rrDist(px, py, mx, my, hw, hh, rad float64) float64 {
	qx := math.Abs(px-mx) - (hw - rad)
	qy := math.Abs(py-my) - (hh - rad)
	switch {
	case qx <= 0 && qy <= 0:
		return math.Max(qx, qy) - rad
	case qx <= 0:
		return qy - rad
	case qy <= 0:
		return qx - rad
	}
	return math.Sqrt(qx*qx+qy*qy) - rad
}

// RoundRect: kenar yumuşatmalı yuvarlatılmış dikdörtgen; isteğe bağlı iç kenarlık.
func (c *Canvas) RoundRect(x, y, w, h, rad float64, fill RGB, fa float64, border RGB, ba, bw float64) {
	if w <= 0 || h <= 0 {
		return
	}
	hw, hh := w/2, h/2
	rad = math.Min(rad, math.Min(hw, hh))
	mx, my := x+hw, y+hh
	fr, fg, fb := fill.i()
	br, bgc, bb := border.i()
	FA, BA := fa*255, ba*255
	X0, Y0, X1, Y1 := c.bounds(x-1, y-1, x+w+1, y+h+1)
	for py := Y0; py <= Y1; py++ {
		row := py * c.W
		fy := float64(py) + 0.5
		for px := X0; px <= X1; px++ {
			d := rrDist(float64(px)+0.5, fy, mx, my, hw, hh, rad)
			outer := clampf(0.5-d, 0, 1)
			if outer <= 0 {
				continue
			}
			inner := outer
			if bw > 0 {
				inner = clampf(0.5-(d+bw), 0, 1)
			}
			if FA > 0 && inner > 0 {
				blendPix(&c.Pix[row+px], fr, fg, fb, int(FA*inner))
			}
			if BA > 0 && bw > 0 && outer > inner {
				blendPix(&c.Pix[row+px], br, bgc, bb, int(BA*(outer-inner)))
			}
		}
	}
}

// Shadow: yuvarlatılmış dikdörtgenin yumuşak gölgesi (iç kısım atlanır).
func (c *Canvas) Shadow(x, y, w, h, rad, blur float64, col RGB, a float64) {
	if a <= 0 {
		return
	}
	hw, hh := w/2, h/2
	rad = math.Min(rad, math.Min(hw, hh))
	mx, my := x+hw, y+hh
	r, g, b := col.i()
	X0, Y0, X1, Y1 := c.bounds(x-blur, y-blur, x+w+blur, y+h+blur)
	for py := Y0; py <= Y1; py++ {
		row := py * c.W
		fy := float64(py) + 0.5
		for px := X0; px <= X1; px++ {
			d := rrDist(float64(px)+0.5, fy, mx, my, hw, hh, rad)
			if d > blur {
				continue
			}
			t := 1 - clampf(d/blur, 0, 1)
			blendPix(&c.Pix[row+px], r, g, b, int(255*a*t*t))
		}
	}
}

func (c *Canvas) Disc(cx, cy, rad float64, col RGB, a float64) {
	r, g, b := col.i()
	X0, Y0, X1, Y1 := c.bounds(cx-rad-1, cy-rad-1, cx+rad+1, cy+rad+1)
	for y := Y0; y <= Y1; y++ {
		for x := X0; x <= X1; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			cv := rad - math.Sqrt(dx*dx+dy*dy) + 0.5
			if cv > 0 {
				if cv > 1 {
					cv = 1
				}
				blendPix(&c.Pix[y*c.W+x], r, g, b, int(255*a*cv))
			}
		}
	}
}

func (c *Canvas) Ring(cx, cy, rad, thick float64, col RGB, a float64) {
	r, g, b := col.i()
	ext := rad + thick/2 + 1.5
	X0, Y0, X1, Y1 := c.bounds(cx-ext, cy-ext, cx+ext, cy+ext)
	lo := math.Max(0, rad-thick/2-1.5)
	lo2, hi2 := lo*lo, ext*ext
	for y := Y0; y <= Y1; y++ {
		dy := float64(y) + 0.5 - cy
		for x := X0; x <= X1; x++ {
			dx := float64(x) + 0.5 - cx
			d2 := dx*dx + dy*dy
			if d2 < lo2 || d2 > hi2 {
				continue
			}
			off := math.Abs(math.Sqrt(d2) - rad)
			cv := thick/2 + 0.5 - off
			if cv > 0 {
				if cv > 1 {
					cv = 1
				}
				blendPix(&c.Pix[y*c.W+x], r, g, b, int(255*a*cv))
			}
		}
	}
}

// Glow: toplamalı radyal ışıma (koyu tema için).
func (c *Canvas) Glow(cx, cy, rad float64, col RGB, strength float64) {
	if strength <= 0 {
		return
	}
	X0, Y0, X1, Y1 := c.bounds(cx-rad, cy-rad, cx+rad, cy+rad)
	inv := 1 / rad
	for y := Y0; y <= Y1; y++ {
		dy := float64(y) + 0.5 - cy
		for x := X0; x <= X1; x++ {
			dx := float64(x) + 0.5 - cx
			d := math.Sqrt(dx*dx+dy*dy) * inv
			if d >= 1 {
				continue
			}
			t := 1 - d
			f := t * t * strength
			addPix(&c.Pix[y*c.W+x], int(col.R*f), int(col.G*f), int(col.B*f))
		}
	}
}

// ---------------------------------------------------------------------------
// Çokgen dolgusu: sıfırdan farklı sarım (nonzero), 4 alt tarama, yatayda tam kapsama.
// Tüm alt çokgenler aynı yöne çevrildiğinden üst üste binenler birleşir (union).
// ---------------------------------------------------------------------------

func signedArea(p []pt) float64 {
	s := 0.0
	for i := range p {
		q := p[(i+1)%len(p)]
		s += p[i].x*q.y - q.x*p[i].y
	}
	return s
}

func (c *Canvas) addSpan(xa, xb float64, w float32) {
	if xb <= 0 || xa >= float64(c.W) || xb <= xa {
		return
	}
	if xa < 0 {
		xa = 0
	}
	if xb > float64(c.W) {
		xb = float64(c.W)
	}
	ia, ib := int(xa), int(xb)
	if ib >= c.W {
		ib = c.W - 1
	}
	if ia == ib {
		c.cov[ia] += float32(xb-xa) * w
		return
	}
	c.cov[ia] += float32(float64(ia+1)-xa) * w
	for i := ia + 1; i < ib; i++ {
		c.cov[i] += w
	}
	c.cov[ib] += float32(xb-float64(ib)) * w
}

func (c *Canvas) FillPaths(polys [][]pt, col RGB, alpha float64) {
	if alpha <= 0 {
		return
	}
	c.edges = c.edges[:0]
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		if len(poly) < 3 {
			continue
		}
		sgn := 1
		if signedArea(poly) < 0 {
			sgn = -1
		}
		for i := range poly {
			p, q := poly[i], poly[(i+1)%len(poly)]
			minX, maxX = math.Min(minX, p.x), math.Max(maxX, p.x)
			minY, maxY = math.Min(minY, p.y), math.Max(maxY, p.y)
			if p.y == q.y {
				continue
			}
			d := sgn
			if p.y > q.y {
				p, q = q, p
				d = -d
			}
			c.edges = append(c.edges, edge{p.x, p.y, q.y, (q.x - p.x) / (q.y - p.y), d})
		}
	}
	if len(c.edges) == 0 {
		return
	}
	sort.Slice(c.edges, func(i, j int) bool { return c.edges[i].y0 < c.edges[j].y0 })
	X0, Y0, X1, Y1 := c.bounds(minX, minY, maxX, maxY)
	if X0 > X1 || Y0 > Y1 {
		return
	}
	r, g, b := col.i()
	A := float32(alpha * 255)
	c.active = c.active[:0]
	ei := 0
	const ss = 4
	for y := Y0; y <= Y1; y++ {
		for i := X0; i <= X1+1 && i < len(c.cov); i++ {
			c.cov[i] = 0
		}
		any := false
		for s := 0; s < ss; s++ {
			sy := float64(y) + (float64(s)+0.5)/ss
			for ei < len(c.edges) && c.edges[ei].y0 <= sy {
				c.active = append(c.active, ei)
				ei++
			}
			xs := c.xs[:0]
			k := 0
			for _, idx := range c.active {
				e := &c.edges[idx]
				if e.y1 <= sy {
					continue
				}
				c.active[k] = idx
				k++
				xs = append(xs, cross{e.x0 + (sy-e.y0)*e.dxdy, e.dir})
			}
			c.active = c.active[:k]
			for i := 1; i < len(xs); i++ {
				for j := i; j > 0 && xs[j].x < xs[j-1].x; j-- {
					xs[j], xs[j-1] = xs[j-1], xs[j]
				}
			}
			wind, start := 0, 0.0
			for _, cr := range xs {
				prev := wind
				wind += cr.d
				if prev == 0 && wind != 0 {
					start = cr.x
				} else if prev != 0 && wind == 0 {
					c.addSpan(start, cr.x, 1.0/ss)
					any = true
				}
			}
			c.xs = xs
		}
		if !any {
			continue
		}
		row := y * c.W
		for x := X0; x <= X1; x++ {
			cv := c.cov[x]
			if cv <= 0.002 {
				continue
			}
			if cv > 1 {
				cv = 1
			}
			blendPix(&c.Pix[row+x], r, g, b, int(A*cv))
		}
	}
}

// ---------------------------------------------------------------------------
// Yol yardımcıları
// ---------------------------------------------------------------------------

func circlePts(cx, cy, r float64) []pt {
	n := int(clampf(r*1.3, 10, 120))
	out := make([]pt, n)
	for i := 0; i < n; i++ {
		a := float64(i) / float64(n) * 2 * math.Pi
		out[i] = pt{cx + r*math.Cos(a), cy + r*math.Sin(a)}
	}
	return out
}

// clockPt: saat açısı (0 = tepe, saat yönünde artar).
func clockPt(cx, cy, r, a float64) pt { return pt{cx + r*math.Sin(a), cy - r*math.Cos(a)} }

// arcStroke: yuvarlak uçlu yay (saat açıları, a0 → a1).
func arcStroke(cx, cy, r, w, a0, a1 float64) [][]pt {
	sweep := a1 - a0
	if sweep <= 0 {
		return nil
	}
	n := int(clampf(math.Abs(sweep)*r/3, 6, 400))
	band := make([]pt, 0, 2*n+2)
	for i := 0; i <= n; i++ {
		a := a0 + sweep*float64(i)/float64(n)
		band = append(band, clockPt(cx, cy, r+w/2, a))
	}
	for i := n; i >= 0; i-- {
		a := a0 + sweep*float64(i)/float64(n)
		band = append(band, clockPt(cx, cy, r-w/2, a))
	}
	out := [][]pt{band}
	if sweep < 2*math.Pi-1e-6 {
		p0, p1 := clockPt(cx, cy, r, a0), clockPt(cx, cy, r, a1)
		out = append(out, circlePts(p0.x, p0.y, w/2), circlePts(p1.x, p1.y, w/2))
	}
	return out
}

func segQuad(p, q pt, w float64) []pt {
	dx, dy := q.x-p.x, q.y-p.y
	l := math.Hypot(dx, dy)
	if l < 1e-9 {
		return nil
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	return []pt{{p.x + nx, p.y + ny}, {q.x + nx, q.y + ny}, {q.x - nx, q.y - ny}, {p.x - nx, p.y - ny}}
}

// stroke: yuvarlak uç ve birleşimli çoklu çizgi.
func stroke(pts []pt, w float64, closed bool) [][]pt {
	var out [][]pt
	n := len(pts)
	for i := 0; i+1 < n; i++ {
		if q := segQuad(pts[i], pts[i+1], w); q != nil {
			out = append(out, q)
		}
	}
	if closed && n > 2 {
		if q := segQuad(pts[n-1], pts[0], w); q != nil {
			out = append(out, q)
		}
	}
	for _, p := range pts {
		out = append(out, circlePts(p.x, p.y, w/2))
	}
	return out
}

func roundRectPts(x, y, w, h, r float64) []pt {
	r = math.Min(r, math.Min(w, h)/2)
	var out []pt
	corner := func(cx, cy, a0 float64) {
		for i := 0; i <= 6; i++ {
			a := a0 + float64(i)/6*math.Pi/2
			out = append(out, pt{cx + r*math.Cos(a), cy + r*math.Sin(a)})
		}
	}
	corner(x+w-r, y+r, -math.Pi/2)
	corner(x+w-r, y+h-r, 0)
	corner(x+r, y+h-r, math.Pi/2)
	corner(x+r, y+r, math.Pi)
	return out
}

// dither: banding'i önlemek için küçük konum gürültüsü (-0.5..0.5).
func dither(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&1023)/1023 - 0.5
}
