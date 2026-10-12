package main

import (
	"fmt"
	"math"
	"time"
)

func (a *App) drawModal() {
	t := easeOut(a.modalT)
	p, u := a.pal, a.u
	W, H := float64(a.W), float64(a.H)
	a.c.FillRect(0, 0, W, H, p.Scrim, p.ScrimA*t)
	a.hit(0, 0, W, H, idScrim, "", "", curArrow)

	var w, h float64
	var title string
	switch a.modal {
	case modalSettings:
		w, h, title = 780, 548, "Ayarlar"
	case modalStats:
		w, h, title = 780, 500, "İstatistikler"
	default:
		w, h, title = 620, 420, "Klavye kısayolları"
	}
	m := math.Min(1, math.Min((W-40*u)/(w*u), (H-40*u)/(h*u)))
	mu := m * u
	a.mu = mu
	pw, ph := math.Round(w*mu), math.Round(h*mu)
	x := math.Round((W - pw) / 2)
	y := math.Round((H-ph)/2 + (1-t)*14*u)
	a.c.Shadow(x, y+10*mu, pw, ph, 16*mu, 36*mu, p.Shadow, p.ShadowA*t*1.25)
	a.c.RoundRect(x, y, pw, ph, 16*mu, p.Raised, t, p.Line, t, 1)
	a.hit(x, y, x+pw, y+ph, idPanel, "", "", curArrow)

	a.fadeOn, a.fadeT, a.fadeBg, a.mA = true, t, p.Raised, t
	defer func() { a.fadeOn, a.fadeT, a.mA = false, 1, 1 }()

	a.text(title, x+28*mu, y+38*mu, alignL, fz(faceDisp, 20*mu, 600), p.Text)
	cs := 34 * mu
	a.iconButton(idModalClose, icClose, x+pw-18*mu-cs, y+38*mu, cs, "Kapat", "Esc", false)

	switch a.modal {
	case modalSettings:
		a.drawSettings(x, y, pw, ph)
	case modalStats:
		a.drawStats(x, y, pw, ph)
	default:
		a.drawHelp(x, y, pw, ph)
	}
}

// ---------------------------------------------------------------------------
// Ayarlar
// ---------------------------------------------------------------------------

func (a *App) section(s string, x, y float64) {
	a.text(s, x, y, alignL, fz(faceText, 13*a.mu, 600), a.pal.Text2)
}

func (a *App) label(s string, x, y float64) {
	a.text(s, x, y, alignL, fz(faceText, 14*a.mu, 400), a.pal.Text)
}

func (a *App) stepLimits(row int) (v, lo, hi int) {
	s := a.S
	switch row {
	case 0:
		return s.FocusMin, 5, 120
	case 1:
		return s.ShortMin, 1, 30
	case 2:
		return s.LongMin, 5, 60
	case 3:
		return s.LongEvery, 2, 8
	}
	return s.Goal, 1, 24
}

func (a *App) stepper(row int, right, cy float64, value string) {
	p, mu := a.pal, a.mu
	bs, vw := 28*mu, 108*mu
	x0 := right - bs - vw - bs
	v, lo, hi := a.stepLimits(row)
	for k := 0; k < 2; k++ {
		id := idStepBase + row*2 + k
		bx, icon := x0, icMinus
		dis := v <= lo
		if row == 0 && k == 0 && v < 5 {
			dis = true
		}
		if k == 1 {
			bx, icon, dis = right-bs, icPlus, v >= hi
		}
		h := a.hov[id]
		if dis {
			h = 0
		}
		a.c.RoundRect(bx, cy-bs/2, bs, bs, 8*mu, p.Surface2.Mix(p.Line, h*0.6), a.mA, p.Line, 0, 0)
		col, al := p.Text2.Mix(p.Text, h), a.mA
		if dis {
			col, al = p.Text3, a.mA*0.5
		}
		a.c.FillPaths(iconPaths(icon, bx+bs/2, cy, 15*mu), col, al)
		cur := curHand
		if dis {
			cur = curArrow
		}
		a.hit(bx, cy-bs/2, bx+bs, cy+bs/2, id, "", "", cur)
	}
	a.text(value, x0+bs+vw/2, cy, alignC, fz(faceText, 14*mu, 600), p.Text)
}

func (a *App) toggleSwitch(k int, left, right, cy float64) {
	p, mu := a.pal, a.mu
	id := idToggleBase + k
	tw, th := 42*mu, 24*mu
	x := right - tw
	v := a.tgl[id]
	h := a.hov[id]
	track := p.Line.Mix(p.Text3, 0.25+h*0.3).Mix(p.Accent, v)
	a.c.RoundRect(x, cy-th/2, tw, th, th/2, track, a.mA, p.Line, 0, 0)
	kn := 18 * mu
	kx := x + 3*mu + kn/2 + v*(tw-6*mu-kn)
	a.c.Disc(kx, cy+0.8*mu, kn/2+0.6, black, 0.12*a.mA)
	a.c.Disc(kx, cy, kn/2, white.Mix(p.Text2, (1-v)*p.Dark*0.25), a.mA)
	a.hit(left, cy-th/2-6*mu, right+4*mu, cy+th/2+6*mu, id, "", "", curHand)
}

func (a *App) segGroup(g int, right, cy, w float64, opts []string, sel int) {
	p, mu := a.pal, a.mu
	h, pad := 30*mu, 3*mu
	x := right - w
	a.c.RoundRect(x, cy-h/2, w, h, 9*mu, p.Well, a.mA, p.Line, a.mA*0.7, 1)
	ow := (w - 2*pad) / float64(len(opts))
	for i, o := range opts {
		ox := x + pad + float64(i)*ow
		id := idSegBase + g*10 + i
		col := p.Text2.Mix(p.Text, a.hov[id])
		wt := 400
		if i == sel {
			a.c.Shadow(ox, cy-h/2+pad+1*mu, ow, h-2*pad, 6*mu, 5*mu, p.Shadow, p.ShadowA*0.6*(1-p.Dark)*a.mA)
			a.c.RoundRect(ox, cy-h/2+pad, ow, h-2*pad, 6.5*mu, p.Knob, a.mA, p.Line, a.mA*0.6*p.Dark, 1)
			col, wt = p.Text, 600
		}
		a.text(o, ox+ow/2, cy, alignC, fz(faceText, 13*mu, wt), col)
		a.hit(ox, cy-h/2, ox+ow, cy+h/2, id, "", "", curHand)
	}
}

func (a *App) slider(right, cy, w float64) {
	p, mu := a.pal, a.mu
	sx0 := right - 46*mu - w
	sx1 := sx0 + w
	a.sliderX0, a.sliderX1 = sx0, sx1
	v := float64(a.S.Volume) / 100
	dis := a.S.Sound == 3
	fill := p.Accent
	if dis {
		fill = p.Text3
	}
	al := a.mA
	if dis {
		al *= 0.5
	}
	a.c.RoundRect(sx0, cy-2*mu, w, 4*mu, 2*mu, p.Line, al, p.Line, 0, 0)
	if v > 0 {
		a.c.RoundRect(sx0, cy-2*mu, math.Max(4*mu, w*v), 4*mu, 2*mu, fill, al, p.Line, 0, 0)
	}
	h := a.hov[idSlider]
	if a.dragID == idSlider {
		h = 1
	}
	kx := sx0 + w*v
	kr := (8 + 1.5*h) * mu
	a.c.Disc(kx, cy+1*mu, kr+0.8, black, 0.14*al)
	a.c.Disc(kx, cy, kr, white.Mix(p.Text2, p.Dark*0.15), al)
	a.c.Ring(kx, cy, kr-0.5, 1, p.Line, al*(1-p.Dark))
	a.text(fmt.Sprintf("%%%d", a.S.Volume), right, cy, alignR, fz(faceText, 13.5*mu, 600), p.Text2)
	cur := curHand
	if dis {
		cur = curArrow
	}
	a.hit(sx0-10*mu, cy-14*mu, sx1+10*mu, cy+14*mu, idSlider, "", "", cur)
}

func (a *App) drawSettings(x, y, w, h float64) {
	p, mu, s := a.pal, a.mu, a.S
	pad, gutter := 28*mu, 44*mu
	colW := (w - 2*pad - gutter) / 2
	lx := x + pad
	rx := lx + colW + gutter
	a.c.FillRect(x+w/2, y+70*mu, x+w/2+1, y+h-62*mu, p.Line, a.mA*0.8)

	rh := 44 * mu
	sy := y + 82*mu
	r0 := sy + 32*mu

	a.section("Zamanlayıcı", lx, sy)
	a.label("Odak süresi", lx, r0)
	a.stepper(0, lx+colW, r0, fmt.Sprintf("%d dk", s.FocusMin))
	a.label("Kısa mola", lx, r0+rh)
	a.stepper(1, lx+colW, r0+rh, fmt.Sprintf("%d dk", s.ShortMin))
	a.label("Uzun mola", lx, r0+2*rh)
	a.stepper(2, lx+colW, r0+2*rh, fmt.Sprintf("%d dk", s.LongMin))
	a.label("Uzun mola sıklığı", lx, r0+3*rh)
	a.stepper(3, lx+colW, r0+3*rh, fmt.Sprintf("%d odakta bir", s.LongEvery))

	sy2 := r0 + 3*rh + 42*mu
	r1 := sy2 + 32*mu
	a.section("Otomatik başlatma", lx, sy2)
	a.label("Molaları otomatik başlat", lx, r1)
	a.toggleSwitch(0, lx, lx+colW, r1)
	a.label("Odakları otomatik başlat", lx, r1+rh)
	a.toggleSwitch(1, lx, lx+colW, r1+rh)

	sy3 := r1 + rh + 42*mu
	a.section("Hedef", lx, sy3)
	a.label("Günlük hedef", lx, sy3+32*mu)
	a.stepper(4, lx+colW, sy3+32*mu, fmt.Sprintf("%d pomodoro", s.Goal))

	a.section("Görünüm", rx, sy)
	a.label("Tema", rx, r0)
	a.segGroup(0, rx+colW, r0, 204*mu, []string{"Sistem", "Açık", "Koyu"}, s.Theme)
	a.label("Arka plan", rx, r0+rh)
	a.segGroup(1, rx+colW, r0+rh, 204*mu, []string{"Düz", "Sade", "Şehir"}, s.Background)

	sy4 := r0 + rh + 42*mu
	q0 := sy4 + 32*mu
	a.section("Ses ve bildirim", rx, sy4)
	a.label("Bildirim sesi", rx, q0)
	a.segGroup(2, rx+colW, q0, 236*mu, soundName[:], s.Sound)
	a.label("Ses düzeyi", rx, q0+rh)
	a.slider(rx+colW, q0+rh, 150*mu)
	a.label("Masaüstü bildirimi", rx, q0+2*rh)
	a.toggleSwitch(2, rx, rx+colW, q0+2*rh)

	sy5 := q0 + 2*rh + 42*mu
	a.section("Pencere", rx, sy5)
	a.label("Her zaman üstte", rx, sy5+32*mu)
	a.toggleSwitch(3, rx, rx+colW, sy5+32*mu)

	fy := y + h - 30*mu
	a.c.FillRect(x+1, fy-26*mu, x+w-1, fy-26*mu+1, p.Line, a.mA*0.8)
	a.text("Değişiklikler otomatik kaydedilir.", lx, fy, alignL, fz(faceText, 12.5*mu, 400), p.Text3)
	lf := fz(faceText, 13*mu, 600)
	lw := a.measure(lf, "Klavye kısayolları")
	col := p.Ink
	if a.hov[idHelpLink] > 0 {
		col = p.Ink.Mix(p.Text, a.hov[idHelpLink]*0.5)
	}
	a.c.FillPaths(iconPaths(icKeyboard, rx+colW-lw-14*mu, fy, 17*mu), col, a.mA)
	a.text("Klavye kısayolları", rx+colW, fy, alignR, lf, col)
	a.hit(rx+colW-lw-26*mu, fy-14*mu, rx+colW+4*mu, fy+14*mu, idHelpLink, "", "", curHand)
}

// ---------------------------------------------------------------------------
// İstatistikler
// ---------------------------------------------------------------------------

func barPts(x, y, w, h, r float64) []pt {
	r = math.Min(r, math.Min(w/2, h))
	var out []pt
	out = append(out, pt{x, y + h})
	for i := 0; i <= 6; i++ {
		a := math.Pi + float64(i)/6*math.Pi/2
		out = append(out, pt{x + r + r*math.Cos(a), y + r + r*math.Sin(a)})
	}
	for i := 0; i <= 6; i++ {
		a := -math.Pi/2 + float64(i)/6*math.Pi/2
		out = append(out, pt{x + w - r + r*math.Cos(a), y + r + r*math.Sin(a)})
	}
	out = append(out, pt{x + w, y + h})
	return out
}

func niceStep(maxMin int) (top, step int) {
	for _, s := range []int{15, 30, 60, 90, 120, 180, 240, 360} {
		if (maxMin+s-1)/s <= 4 {
			return ((maxMin + s - 1) / s) * s, s
		}
	}
	s := 480
	return ((maxMin + s - 1) / s) * s, s
}

func (a *App) drawStats(x, y, w, h float64) {
	p, mu := a.pal, a.mu
	now := a.now
	day := a.S.Day(now)
	streak := a.S.Streak(now)
	tc, _ := a.S.Totals()
	pad := 28 * mu

	ky, kh, gap := y+70*mu, 86*mu, 12*mu
	kw := (w - 2*pad - 3*gap) / 4
	streakS := fmt.Sprintf("%d gün", streak)
	tiles := []struct{ v, sub, l string }{
		{fmt.Sprint(day.Count), fmt.Sprintf("/ %d", a.S.Goal), "Bugün tamamlanan"},
		{fmtDur(day.Sec), "", "Bugünkü odak süresi"},
		{streakS, "", "Kesintisiz seri"},
		{fmt.Sprint(tc), "", "Toplam pomodoro"},
	}
	vf := fz(faceDisp, 26*mu, 600)
	for i, t := range tiles {
		tx := x + pad + float64(i)*(kw+gap)
		a.c.RoundRect(tx, ky, kw, kh, 12*mu, p.Well, a.mA, p.Line, a.mA*0.7, 1)
		a.text(t.v, tx+16*mu, ky+34*mu, alignL, vf, p.Text)
		if t.sub != "" {
			a.text(t.sub, tx+16*mu+a.measure(vf, t.v)+6*mu, ky+38*mu, alignL, fz(faceText, 14*mu, 400), p.Text3)
		}
		a.text(t.l, tx+16*mu, ky+64*mu, alignL, fz(faceText, 12.5*mu, 400), p.Text2)
	}

	top := ky + kh + 16*mu
	chH := y + h - 26*mu - top
	chW := math.Round((w - 2*pad - 16*mu) * 0.60)
	cx0 := x + pad
	a.c.RoundRect(cx0, top, chW, chH, 12*mu, p.Well, a.mA, p.Line, a.mA*0.7, 1)
	a.text("Son 7 gün", cx0+18*mu, top+26*mu, alignL, fz(faceText, 14*mu, 600), p.Text)
	a.text("Odak süresi, dakika", cx0+chW-18*mu, top+26*mu, alignR, fz(faceText, 12*mu, 400), p.Text3)

	days := a.S.LastDays(now, 7)
	maxMin := 60
	for _, d := range days {
		if m := d.Sec / 60; m > maxMin {
			maxMin = m
		}
	}
	topV, step := niceStep(maxMin)
	px0, px1 := cx0+46*mu, cx0+chW-18*mu
	py0, py1 := top+56*mu, top+chH-38*mu
	sf := fz(faceText, 11.5*mu, 400)
	for v := 0; v <= topV; v += step {
		yv := math.Round(py1 - float64(v)/float64(topV)*(py1-py0))
		al := 0.55
		if v == 0 {
			al = 1
		}
		a.c.FillRect(px0, yv, px1, yv+1, p.Line, a.mA*al)
		a.text(fmt.Sprint(v), px0-10*mu, yv, alignR, sf, p.Text3)
	}
	slot := (px1 - px0) / 7
	bw := math.Min(30*mu, slot*0.52)
	type lbl struct {
		x, y float64
		s    string
	}
	var labels []lbl
	for i, d := range days {
		bx := px0 + slot*(float64(i)+0.5)
		bh := float64(d.Sec) / 60 / float64(topV) * (py1 - py0)
		today := i == len(days)-1
		hovered := a.chartHover == i
		al := 0.5
		if today || hovered {
			al = 1
		}
		if bh > 0.5 {
			a.c.FillPaths([][]pt{barPts(bx-bw/2, py1-bh, bw, bh, 4*mu)}, p.Accent, al*a.mA)
			if today || hovered {
				labels = append(labels, lbl{bx, py1 - bh - 11*mu, fmt.Sprint(d.Sec / 60)})
			}
		}
		dc, dw := p.Text3, 400
		name := dayShort[d.Date.Weekday()]
		if today {
			dc, dw, name = p.Text, 600, "Bugün"
		}
		a.text(name, bx, py1+18*mu, alignC, fz(faceText, 12*mu, dw), dc)
		a.hit(bx-slot/2, py0-10*mu, bx+slot/2, py1+28*mu, idChartBar+i, "", "", curArrow)
	}
	for _, l := range labels {
		a.text(l.s, l.x, l.y, alignC, fz(faceText, 12*mu, 600), p.Text2)
	}

	// bugünkü oturumlar
	lx0 := cx0 + chW + 16*mu
	lw := x + w - pad - lx0
	a.c.RoundRect(lx0, top, lw, chH, 12*mu, p.Well, a.mA, p.Line, a.mA*0.7, 1)
	a.text("Bugünkü oturumlar", lx0+18*mu, top+26*mu, alignL, fz(faceText, 14*mu, 600), p.Text)
	recs := a.S.TodayRecords(now)
	if len(recs) == 0 {
		a.text("Henüz oturum yok.", lx0+18*mu, top+62*mu, alignL, fz(faceText, 13*mu, 400), p.Text2)
		a.text("Bitirdiğin her odak burada listelenir.", lx0+18*mu, top+84*mu, alignL, fz(faceText, 12.5*mu, 400), p.Text3)
	} else {
		rowH := 32 * mu
		ry := top + 60*mu
		maxRows := int((top + chH - 14*mu - ry + rowH/2) / rowH)
		if maxRows < 1 {
			maxRows = 1
		}
		more := 0
		if len(recs) > maxRows {
			more = len(recs) - (maxRows - 1)
			recs = recs[:maxRows-1]
		}
		tf := fz(faceText, 12.5*mu, 400)
		nf := fz(faceText, 13*mu, 400)
		for i, r := range recs {
			tm := time.Unix(r.End, 0).In(now.Location()).Format("15:04")
			task := r.Task
			if task == "" {
				task = "Adsız odak"
			}
			dur := fmtDur(r.Sec)
			nc := p.Text
			if !r.Full {
				dur += ", yarım"
				nc = p.Text2
			}
			dw := a.measure(tf, dur)
			tx := lx0 + 18*mu + a.measure(tf, "00:00") + 12*mu
			a.text(tm, lx0+18*mu, ry, alignL, tf, p.Text3)
			a.text(a.ellipsize(nf, task, lx0+lw-18*mu-dw-12*mu-tx), tx, ry, alignL, nf, nc)
			a.text(dur, lx0+lw-18*mu, ry, alignR, tf, p.Text2)
			if i < len(recs)-1 || more > 0 {
				a.c.FillRect(lx0+18*mu, ry+rowH/2, lx0+lw-18*mu, ry+rowH/2+1, p.Line, a.mA*0.6)
			}
			ry += rowH
		}
		if more > 0 {
			a.text(fmt.Sprintf("%d oturum daha", more), lx0+18*mu, ry, alignL, tf, p.Text3)
		}
	}

	// grafik ipucu (üstte kalsın diye ayrı katman)
	if a.chartHover >= 0 && a.chartHover < len(days) && a.modalOpen {
		a.flush()
		d := days[a.chartHover]
		bx := px0 + slot*(float64(a.chartHover)+0.5)
		line1 := fmt.Sprintf("%s, %d %s", dayLong[d.Date.Weekday()], d.Date.Day(), monShort[d.Date.Month()-1])
		line2 := fmt.Sprintf("%d pomodoro, %s odak", d.Count, fmtDur(d.Sec))
		f1, f2 := fz(faceText, 12.5*mu, 600), fz(faceText, 12.5*mu, 400)
		tw := math.Max(a.measure(f1, line1), a.measure(f2, line2))
		tW, tH := tw+24*mu, 50*mu
		bh := float64(d.Sec) / 60 / float64(topV) * (py1 - py0)
		tx := clampf(bx-tW/2, cx0+8*mu, cx0+chW-tW-8*mu)
		ty := math.Max(top+8*mu, py1-bh-tH-26*mu)
		a.c.Shadow(tx, ty+3*mu, tW, tH, 8*mu, 10*mu, black, 0.25*a.mA)
		a.c.RoundRect(tx, ty, tW, tH, 8*mu, p.Tip, a.mA, p.Tip, 0, 0)
		a.text(line1, tx+12*mu, ty+16*mu, alignL, f1, p.Tip.Mix(p.TipText, a.mA))
		a.text(line2, tx+12*mu, ty+34*mu, alignL, f2, p.Tip.Mix(p.TipText, a.mA*0.8))
	}
}

// ---------------------------------------------------------------------------
// Kısayollar
// ---------------------------------------------------------------------------

var shortcuts = [][2]string{
	{"Boşluk", "Başlat / duraklat"},
	{"R", "Sıfırla"},
	{"N", "Sonrakine geç"},
	{"1  2  3", "Odak, kısa mola, uzun mola"},
	{"↑  ↓", "Süreyi 1 dk değiştir"},
	{"E", "Görevi düzenle"},
	{"Esc", "Kapat, vazgeç"},
	{"S", "Ayarlar"},
	{"I", "İstatistikler"},
	{"D", "Açık / koyu tema"},
	{"C", "Kompakt pencere"},
	{"P", "Her zaman üstte"},
	{"M", "Sesi aç / kapat"},
	{"F1", "Bu liste"},
}

func (a *App) drawHelp(x, y, w, h float64) {
	p, mu := a.pal, a.mu
	pad := 28 * mu
	colW := (w - 2*pad - 32*mu) / 2
	kf := fz(faceText, 12.5*mu, 600)
	df := fz(faceText, 13.5*mu, 400)
	rows := (len(shortcuts) + 1) / 2
	for i, sc := range shortcuts {
		col, row := i/rows, i%rows
		cx := x + pad + float64(col)*(colW+32*mu)
		cy := y + 92*mu + float64(row)*40*mu
		kw := math.Max(30*mu, a.measure(kf, sc[0])+20*mu)
		a.c.RoundRect(cx, cy-13*mu, kw, 26*mu, 6*mu, p.Well, a.mA, p.Line, a.mA, 1)
		a.c.FillRect(cx+3*mu, cy+12*mu, cx+kw-3*mu, cy+13*mu, p.Line, a.mA)
		a.text(sc[0], cx+kw/2, cy, alignC, kf, p.Text)
		a.text(sc[1], cx+104*mu, cy, alignL, df, p.Text2)
	}
	a.text("Görev kutusuna yazarken kısayollar devre dışıdır.", x+pad, y+h-32*mu, alignL, fz(faceText, 12.5*mu, 400), p.Text3)
}
