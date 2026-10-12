package main

import (
	"fmt"
	"math"
)

var white = RGB{255, 255, 255}
var black = RGB{0, 0, 0}

// ---------------------------------------------------------------------------
// Ortak yardımcılar
// ---------------------------------------------------------------------------

func fz(face uint8, size float64, weight int) Font {
	return Font{Face: face, Size: int(math.Round(size)), Weight: weight}
}

func (a *App) text(s string, x, y float64, al int8, f Font, c RGB) {
	if a.fadeOn {
		c = a.fadeBg.Mix(c, a.fadeT)
	}
	a.texts = append(a.texts, TextCmd{S: s, X: int(math.Round(x)), Y: int(math.Round(y)), Align: al, F: f, C: c})
}

func (a *App) flush() {
	if len(a.texts) > 0 {
		a.host.DrawText(a.texts)
		a.texts = a.texts[:0]
	}
}

func (a *App) hit(x0, y0, x1, y1 float64, id int, tip, key string, cur int) {
	a.hits = append(a.hits, Hit{x0, y0, x1, y1, id, tip, key, cur, a.tipBelow})
}

func (a *App) measure(f Font, s string) float64 { return float64(a.host.Measure(f, s)) }

func (a *App) ellipsize(f Font, s string, maxW float64) string {
	if a.measure(f, s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 {
		r = r[:len(r)-1]
		if a.measure(f, string(r)+"…") <= maxW {
			return string(r) + "…"
		}
	}
	return "…"
}

// Render: tüm sahneyi katman katman çizer (piksel → yazı → üst katman pikseli → yazı ...).
func (a *App) Render() {
	if a.c == nil {
		return
	}
	a.hits = a.hits[:0]
	a.texts = a.texts[:0]
	if a.compact {
		a.drawCompact()
		a.flush()
	} else {
		a.drawMain()
		a.flush()
		if a.modal != modalNone {
			a.drawModal()
			a.flush()
		}
		a.drawTooltip()
		a.flush()
		a.drawToast()
		a.flush()
	}
	a.dirty = false
	a.lastRender = a.t
	a.lastSecs = a.M.Secs()
	a.lastBlink = int((a.t - a.caretT0) / 0.53)
}

// ---------------------------------------------------------------------------
// Ana ekran
// ---------------------------------------------------------------------------

type mainLayout struct {
	cx, cy, R, thick     float64
	taskY, ctrlY, stripY float64
}

func (a *App) layout() mainLayout {
	u := a.u
	W, H := float64(a.W), float64(a.H)
	top, bottom := 66*u, 58*u
	avail := H - top - bottom
	fixed := (24 + 40 + 24 + 68) * u
	R := (avail - fixed - 12*u) / 2
	R = math.Min(R, W*0.2)
	R = clampf(R, 88*u, 200*u)
	total := 2*R + fixed
	y0 := top + math.Max(0, (avail-total)/2)
	cy := y0 + R
	taskY := cy + R + 24*u + 20*u
	ctrlY := taskY + 20*u + 24*u + 34*u
	return mainLayout{math.Round(W / 2), math.Round(cy), R, math.Max(4.5*u, R*0.042), taskY, ctrlY, H - 30*u}
}

func (a *App) drawMain() {
	L := a.layout()
	a.ensureBG(L.cx, L.cy, L.R)
	copy(a.c.Pix, a.bg.Pix)
	a.drawTopBar()
	a.drawRing(L)
	a.drawTask(L)
	a.drawControls(L)
	a.drawStrip(L)
}

func (a *App) iconButton(id, icon int, x, cy, size float64, tip, key string, active bool) {
	p, u := a.pal, a.u
	h := a.hov[id]
	if active {
		h = math.Max(h, 1)
	}
	if a.pressID == id && a.hoverID == id {
		h = 1.4
	}
	if h > 0 {
		fill := p.Surface2
		if h > 1 {
			fill = p.Surface2.Mix(p.Line, 0.5)
		}
		a.c.RoundRect(x, cy-size/2, size, size, 10*u*a.mu1(), fill, math.Min(1, h)*a.mA, p.Line, math.Min(1, h)*0.7*a.mA, 1)
	}
	col := p.Text2.Mix(p.Text, math.Min(1, h))
	a.c.FillPaths(iconPaths(icon, x+size/2, cy, size*0.52), col, a.mA)
	a.hit(x, cy-size/2, x+size, cy+size/2, id, tip, key, curHand)
}

// mu1: modal içindeysek modal ölçeği, değilsek 1.
func (a *App) mu1() float64 {
	if a.fadeOn && a.u > 0 {
		return a.mu / a.u
	}
	return 1
}

func (a *App) drawTopBar() {
	p, u := a.pal, a.u
	W := float64(a.W)
	y := math.Round(34 * u)
	a.tipBelow = true
	defer func() { a.tipBelow = false }()

	lx := 24 * u
	a.c.FillPaths(logoPaths(lx+9*u, y, 9*u), p.Accent, 1)
	if W > 820*u {
		a.text("Pomodoro", lx+26*u, y, alignL, fz(faceText, 14.5*u, 600), p.Text)
	}

	segW, segH, pad := 104*u, 32*u, 3*u
	total := 3*segW + 2*pad
	sx := math.Round(W/2 - total/2)
	a.c.RoundRect(sx, y-segH/2-pad, total, segH+2*pad, (segH+2*pad)/2, p.Well, 1, p.Line, 0.8, 1)
	ix := sx + pad + a.segX*segW
	a.c.Shadow(ix, y-segH/2+1.5*u, segW, segH, segH/2, 6*u, p.Shadow, p.ShadowA*0.7*(1-p.Dark))
	a.c.RoundRect(ix, y-segH/2, segW, segH, segH/2, p.Knob, 1, p.Line, 0.6*p.Dark, 1)
	keys := []string{"1", "2", "3"}
	for i := 0; i < 3; i++ {
		x0 := sx + pad + float64(i)*segW
		id := idSeg0 + i
		col := p.Text2.Mix(p.Text, a.hov[id])
		if Mode(i) == a.M.Mode {
			col = p.Ink
		}
		a.text(modeName[i], x0+segW/2, y, alignC, fz(faceText, 13.5*u, 600), col)
		a.hit(x0, y-segH/2, x0+segW, y+segH/2, id, modeName[i], keys[i], curHand)
	}

	bs, gap := 36*u, 6*u
	x := W - 20*u - bs
	themeIcon, themeTip := icMoon, "Koyu tema"
	if a.dark() {
		themeIcon, themeTip = icSun, "Açık tema"
	}
	for _, b := range []struct {
		id, icon int
		tip, key string
		active   bool
	}{
		{idCompact, icCompact, "Kompakt pencere", "C", false},
		{idTheme, themeIcon, themeTip, "D", false},
		{idSettings, icGear, "Ayarlar", "S", a.modal == modalSettings && a.modalOpen},
		{idStats, icChart, "İstatistikler", "I", a.modal == modalStats && a.modalOpen},
	} {
		a.iconButton(b.id, b.icon, x, y, bs, b.tip, b.key, b.active)
		x -= bs + gap
	}
}

func (a *App) drawRing(L mainLayout) {
	p, u := a.pal, a.u
	cx, cy, R, th := L.cx, L.cy, L.R, L.thick
	tau := 2 * math.Pi

	// dakika çentikleri (saat kadranı)
	ticks := make([][]pt, 0, 60)
	rOut := R - th*1.9
	for i := 0; i < 60; i++ {
		ang := float64(i) / 60 * tau
		ln, w := 4*u, 1.1*u
		if i%5 == 0 {
			ln, w = 7*u, 1.6*u
		}
		if q := segQuad(clockPt(cx, cy, rOut, ang), clockPt(cx, cy, rOut-ln, ang), w); q != nil {
			ticks = append(ticks, q)
		}
	}
	a.c.FillPaths(ticks, p.Text3, lerp(0.5, 0.36, p.Dark))

	a.c.Ring(cx, cy, R, th, p.Line, 1)
	frac := a.M.Frac() * easeOut(a.intro)
	if frac > 0.0005 {
		a.c.FillPaths(arcStroke(cx, cy, R, th, 0, frac*tau), p.Accent, 1)
		if p.Dark > 0 && frac < 0.999 {
			head := clockPt(cx, cy, R, frac*tau)
			a.c.Glow(head.x, head.y, th*5, p.Accent, 0.32*p.Dark)
		}
	}
	if a.pulse > 0.01 {
		a.c.Ring(cx, cy, R+(1-a.pulse)*28*u, th*a.pulse+0.8, p.Accent, a.pulse*0.55)
	}
	a.hit(cx-R, cy-R, cx+R, cy+R, idRing, "", "", curArrow)

	status := ""
	if a.M.Running {
		status = "Bitiş " + a.M.EndAt.Format("15:04")
	} else if a.M.Started() {
		status = "Duraklatıldı"
	}
	if status != "" {
		a.text(status, cx, cy-R*0.40, alignC, fz(faceText, 13*u, 400), p.Text3)
	}
	a.text(a.M.Clock(), cx, cy-R*0.05, alignC, fz(faceDisp, R*0.50, 300), p.Text)

	ly := cy + R*0.31
	a.text(modeName[a.M.Mode], cx, ly, alignC, fz(faceText, 14.5*u, 600), p.Ink)

	n := a.M.LongEvery
	gap, dr := 13*u, 3.4*u
	dy := cy + R*0.48
	for i := 0; i < n; i++ {
		x := cx + (float64(i)-float64(n-1)/2)*gap
		switch {
		case i < a.M.Cycle:
			a.c.Disc(x, dy, dr, p.Accent, 1)
		case i == a.M.Cycle && a.M.Mode == Focus:
			a.c.Ring(x, dy, dr-0.7*u, 1.4*u, p.Accent, 1)
		default:
			a.c.Disc(x, dy, dr, p.Line, 1)
		}
	}

	if a.adjustable() {
		al := a.ringHov
		bx, bs := R*0.62, 30*u
		for k, id := range []int{idMinus, idPlus} {
			x := cx - bx
			icon, tip, key := icMinus, "1 dk azalt", "↓"
			if k == 1 {
				x, icon, tip, key = cx+bx, icPlus, "1 dk artır", "↑"
			}
			if al > 0.01 {
				h := a.hov[id]
				a.c.Disc(x, ly, bs/2, p.Surface2.Mix(p.Line, h*0.7), al)
				a.c.FillPaths(iconPaths(icon, x, ly, 16*u), p.Text2.Mix(p.Text, h), al)
			}
			a.hit(x-bs/2, ly-bs/2, x+bs/2, ly+bs/2, id, tip, key, curHand)
		}
	}
}

func (a *App) drawTask(L mainLayout) {
	p, u := a.pal, a.u
	w := clampf(float64(a.W)*0.40, 280*u, 400*u)
	h := 40 * u
	x, y := math.Round(L.cx-w/2), math.Round(L.taskY-h/2)
	hv := a.hov[idTask]
	border, bw := p.Line.Mix(p.Text3, hv*0.55), 1.0
	if a.editing {
		border, bw = p.Accent, math.Max(1.5, 1.5*u)
	}
	a.c.RoundRect(x, y, w, h, h/2, p.Surface, 1, border, 1, bw)
	a.c.FillPaths(iconPaths(icPencil, x+21*u, L.taskY, 15*u), p.Text3.Mix(p.Text2, hv), 1)

	f := fz(faceText, 14*u, 400)
	tx, maxW := x+38*u, w-38*u-18*u
	a.taskTX, a.taskFont, a.taskShift = tx, f, 0
	switch {
	case a.editing:
		caretX := a.measure(f, string(a.edit[:a.caret]))
		shift := math.Max(0, caretX-maxW)
		a.taskShift = shift
		if len(a.edit) == 0 {
			a.text("Ne üzerinde çalışıyorsun?", tx, L.taskY, alignL, f, p.Text3)
		} else {
			a.texts = append(a.texts, TextCmd{S: string(a.edit), X: int(tx - shift), Y: int(L.taskY), Align: alignL, F: f, C: p.Text,
				Clip: true, CX0: int(tx), CX1: int(tx + maxW + 2)})
		}
		if int((a.t-a.caretT0)/0.53)%2 == 0 {
			cxp := math.Round(tx - shift + caretX)
			a.c.FillRect(cxp, L.taskY-9*u, cxp+math.Max(1, math.Round(1.4*u)), L.taskY+9*u, p.Text, 1)
		}
	case a.S.Task != "":
		a.text(a.ellipsize(f, a.S.Task, maxW), tx, L.taskY, alignL, f, p.Text)
	default:
		a.text("Ne üzerinde çalışıyorsun?", tx, L.taskY, alignL, f, p.Text3)
	}
	tip := "Görevi düzenle"
	if a.editing {
		tip = ""
	}
	a.hit(x, y, x+w, y+h, idTask, tip, "E", curText)
}

func (a *App) placeCaret(x float64) {
	best, bd := 0, math.Inf(1)
	for i := 0; i <= len(a.edit); i++ {
		px := a.taskTX - a.taskShift + a.measure(a.taskFont, string(a.edit[:i]))
		if d := math.Abs(px - x); d < bd {
			bd, best = d, i
		}
	}
	a.caret = best
	a.caretT0 = a.t
}

func (a *App) roundButton(id, icon int, cx, cy, d float64, primary bool, tip, key string) {
	p, u := a.pal, a.u
	h := a.hov[id]
	pressed := a.pressID == id && a.hoverID == id
	r := d / 2
	if pressed {
		r *= 0.95
	}
	if primary {
		if p.Dark > 0 {
			a.c.Glow(cx, cy+2*u, r*1.9, p.Accent, (0.14+0.08*h)*p.Dark)
		}
		a.c.Shadow(cx-r, cy-r+3*u, 2*r, 2*r, r, 12*u, p.Shadow, p.ShadowA*(1-p.Dark)*1.3)
		fill := p.Accent.Mix(white, h*0.12)
		if pressed {
			fill = p.Accent.Mix(black, 0.08)
		}
		a.c.Disc(cx, cy, r, fill, 1)
		ox := 0.0
		if icon == icPlay {
			ox = d * 0.025
		}
		a.c.FillPaths(iconPaths(icon, cx+ox, cy, d*0.40), p.OnAccent, 1)
	} else {
		a.c.Disc(cx, cy, r, p.Surface.Mix(p.Surface2, h), 1)
		a.c.Ring(cx, cy, r-0.5, 1, p.Line.Mix(p.Text3, h*0.5), 1)
		a.c.FillPaths(iconPaths(icon, cx, cy, d*0.42), p.Text2.Mix(p.Text, h), 1)
	}
	a.hit(cx-d/2, cy-d/2, cx+d/2, cy+d/2, id, tip, key, curHand)
}

func (a *App) playTip() string {
	switch {
	case a.M.Running:
		return "Duraklat"
	case a.M.Started():
		return "Devam et"
	}
	return "Başlat"
}

func (a *App) drawControls(L mainLayout) {
	u := a.u
	big, small := 68*u, 48*u
	off := big/2 + 26*u + small/2
	icon := icPlay
	if a.M.Running {
		icon = icPause
	}
	a.roundButton(idReset, icReset, L.cx-off, L.ctrlY, small, false, "Sıfırla", "R")
	a.roundButton(idPlay, icon, L.cx, L.ctrlY, big, true, a.playTip(), "Boşluk")
	a.roundButton(idSkip, icSkip, L.cx+off, L.ctrlY, small, false, "Sonrakine geç", "N")
}

func (a *App) toastAlpha() float64 {
	if a.toast == "" {
		return 0
	}
	age := a.t - a.toastT0
	return clampf(age/0.18, 0, 1) * clampf((2.9-age)/0.3, 0, 1)
}

func (a *App) drawStrip(L mainLayout) {
	p, u := a.pal, a.u
	// bildirim balonu aynı yerde görünür; şerit o sırada zemine karışır
	hide := a.toastAlpha()
	bgc := p.Bg1
	if a.S.Background == 0 {
		bgc = p.Bg0.Mix(p.Bg1, 0.45)
	}
	fade := func(c RGB) RGB { return c.Mix(bgc, hide) }
	day := a.S.Day(a.now)
	f := fz(faceText, 13*u, 400)
	var s string
	switch {
	case day.Count == 0 && day.Sec == 0:
		s = "Bugün henüz pomodoro yok"
	case day.Count == 0:
		s = fmt.Sprintf("Bugün %s odaklandın", fmtDur(day.Sec))
	default:
		s = fmt.Sprintf("Bugün %d pomodoro, %s odak", day.Count, fmtDur(day.Sec))
	}
	goal := fmt.Sprintf("%d/%d", day.Count, a.S.Goal)
	tw, gw := a.measure(f, s), 96*u
	g2 := a.measure(f, goal)
	total := tw + 18*u + gw + 10*u + g2
	x, y := math.Round(L.cx-total/2), math.Round(L.stripY)
	a.text(s, x, y, alignL, f, fade(p.Text2))
	bx := x + tw + 18*u
	a.c.RoundRect(bx, y-3*u, gw, 6*u, 3*u, p.Line, 1-hide, p.Line, 0, 0)
	if fr := math.Min(1, float64(day.Count)/float64(a.S.Goal)); fr > 0 {
		a.c.RoundRect(bx, y-3*u, math.Max(6*u, gw*fr), 6*u, 3*u, p.Accent, 1-hide, p.Line, 0, 0)
	}
	gc := p.Text2
	if day.Count >= a.S.Goal {
		gc = p.Ink
	}
	a.text(goal, bx+gw+10*u, y, alignL, f, fade(gc))
	a.hit(x-10*u, y-14*u, x+total+10*u, y+14*u, idGoal, "Günlük hedef", "I", curHand)
}

// ---------------------------------------------------------------------------
// Araç ipucu ve bildirim balonu
// ---------------------------------------------------------------------------

func (a *App) tipAlpha() float64 {
	if a.hoverID == 0 || a.pressID != 0 || a.dragID != 0 || a.editing {
		return 0
	}
	return clampf((a.t-a.hoverT0-0.5)/0.14, 0, 1)
}

func (a *App) drawTooltip() {
	al := a.tipAlpha()
	if al <= 0 {
		return
	}
	var h *Hit
	for i := len(a.hits) - 1; i >= 0; i-- {
		if a.hits[i].ID == a.hoverID {
			h = &a.hits[i]
			break
		}
	}
	if h == nil || h.Tip == "" {
		return
	}
	p, u := a.pal, a.u
	f := fz(faceText, 12.5*u, 400)
	fk := fz(faceText, 12*u, 600)
	tw := a.measure(f, h.Tip)
	kw := 0.0
	if h.Key != "" {
		kw = a.measure(fk, h.Key) + 12*u
	}
	w, ht := tw+kw+22*u, 28*u
	cx := (h.X0 + h.X1) / 2
	x := math.Round(clampf(cx-w/2, 8*u, float64(a.W)-w-8*u))
	y := h.Y0 - 8*u - ht
	if h.Below || y < 4*u {
		y = h.Y1 + 8*u
	}
	y = math.Round(y)
	a.c.Shadow(x, y+3*u, w, ht, 7*u, 10*u, black, 0.25*al)
	a.c.RoundRect(x, y, w, ht, 7*u, p.Tip, al, p.Tip, 0, 0)
	a.text(h.Tip, x+11*u, y+ht/2, alignL, f, p.Tip.Mix(p.TipText, al))
	if h.Key != "" {
		a.text(h.Key, x+11*u+tw+12*u, y+ht/2, alignL, fk, p.Tip.Mix(p.TipText, al*0.55))
	}
}

func (a *App) drawToast() {
	if a.toast == "" {
		return
	}
	p, u := a.pal, a.u
	al := a.toastAlpha()
	if al <= 0 {
		return
	}
	f := fz(faceText, 13.5*u, 400)
	tw := a.measure(f, a.toast)
	w, h := tw+40*u, 38*u
	x := math.Round((float64(a.W) - w) / 2)
	y := math.Round(a.layout().stripY - h/2 + (1-easeOut(al))*8*u)
	a.c.Shadow(x, y+6*u, w, h, h/2, 18*u, p.Shadow, p.ShadowA*al)
	a.c.RoundRect(x, y, w, h, h/2, p.Raised, al, p.Line, al, 1)
	a.c.Disc(x+18*u, y+h/2, 3.5*u, p.Accent, al)
	a.text(a.toast, x+28*u, y+h/2, alignL, f, p.Raised.Mix(p.Text, al))
}

// ---------------------------------------------------------------------------
// Kompakt pencere
// ---------------------------------------------------------------------------

func (a *App) drawCompact() {
	p, u := a.pal, a.u
	W, H := float64(a.W), float64(a.H)
	a.c.FillRect(0, 0, W, H, p.Surface, 1)
	a.c.RoundRect(0, 0, W, H, 0, p.Surface, 0, p.Line, 1, 1)
	barH := math.Round(3 * u)
	a.c.FillRect(1, H-barH-1, W-1, H-1, p.Line, 1)
	a.c.FillRect(1, H-barH-1, 1+(W-2)*a.M.Frac(), H-1, p.Accent, 1)

	left := 20 * u
	cy := H/2 - 2*u
	a.text(a.M.Clock(), left-1*u, cy-9*u, alignL, fz(faceDisp, 38*u, 300), p.Text)
	lf := fz(faceText, 12.5*u, 600)
	a.text(modeName[a.M.Mode], left, cy+22*u, alignL, lf, p.Ink)
	status := ""
	if a.M.Running {
		status = "Bitiş " + a.M.EndAt.Format("15:04")
	} else if a.M.Started() {
		status = "Duraklatıldı"
	}
	if status != "" {
		a.text(status, left+a.measure(lf, modeName[a.M.Mode])+10*u, cy+22*u, alignL, fz(faceText, 12.5*u, 400), p.Text3)
	}
	icon := icPlay
	if a.M.Running {
		icon = icPause
	}
	pd := 44 * u
	px := W - 18*u - pd/2
	a.roundButton(idPlay, icon, px, cy, pd, true, a.playTip(), "Boşluk")
	es := 32 * u
	a.iconButton(idExpand, icExpand, px-pd/2-10*u-es, cy, es, "Tam pencere", "Esc", false)
}
