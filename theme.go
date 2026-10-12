package main

// Renk sistemi.
// Koyu tema: gece yarısı laciverti zemin; odak rengi sodyum sokak lambası amberi,
// kısa mola neon turkuaz, uzun mola ay ışığı lavantası.
// Açık tema: bulutlu gün beton grisi, aynı vurgu ailesinin koyulaştırılmış tonları.

type Pal struct {
	Bg0, Bg1 RGB // zemin gradyanı (üst → alt)
	Surface  RGB // kart / düğme yüzeyi
	Surface2 RGB // üzerine gelince
	Raised   RGB // modal paneli
	Well     RGB // içe gömülü kap (segment zemini)
	Knob     RGB // seçili segment / anahtar topuzu
	Line     RGB // kenarlık, halka izi
	Text     RGB
	Text2    RGB
	Text3    RGB
	Scrim    RGB
	OnAccent RGB // vurgu dolgusu üstündeki yazı/ikon
	Accent   RGB // geçerli modun dolgu rengi
	Ink      RGB // geçerli modun yazı rengi (zemin üstünde okunur)
	Tip      RGB // araç ipucu zemini
	TipText  RGB
	Shadow   RGB
	ScrimA   float64
	ShadowA  float64
	Dark     float64 // 1 = koyu, 0 = açık (geçişte ara değer)
}

var (
	darkAccent  = [3]uint32{0xF2A93B, 0x3CC6B0, 0x9A9CF5}
	lightAccent = [3]uint32{0xEFA232, 0x2DB39C, 0x8587EE}
	lightInk    = [3]uint32{0x9A5E00, 0x0A7A68, 0x4B4DC8}
)

func palette(dark bool, md Mode) Pal {
	if dark {
		return Pal{
			Bg0: hex(0x111621), Bg1: hex(0x0A0D13),
			Surface: hex(0x171C27), Surface2: hex(0x202735), Raised: hex(0x161B25),
			Well: hex(0x0E1118), Knob: hex(0x272F3E), Line: hex(0x2A3242),
			Text: hex(0xE9EDF4), Text2: hex(0xA3ABBB), Text3: hex(0x6C7589),
			Scrim: hex(0x04060A), OnAccent: hex(0x15120B),
			Accent: hex(darkAccent[md]), Ink: hex(darkAccent[md]),
			Tip: hex(0x2B3343), TipText: hex(0xE9EDF4), Shadow: hex(0x000000),
			ScrimA: 0.62, ShadowA: 0.45, Dark: 1,
		}
	}
	return Pal{
		Bg0: hex(0xF5F7FA), Bg1: hex(0xE8ECF1),
		Surface: hex(0xFFFFFF), Surface2: hex(0xF0F3F7), Raised: hex(0xFFFFFF),
		Well: hex(0xE9EDF2), Knob: hex(0xFFFFFF), Line: hex(0xD7DDE5),
		Text: hex(0x151922), Text2: hex(0x535C6C), Text3: hex(0x878FA0),
		Scrim: hex(0x0B0F16), OnAccent: hex(0x15120B),
		Accent: hex(lightAccent[md]), Ink: hex(lightInk[md]),
		Tip: hex(0x1F2430), TipText: hex(0xF3F5F8), Shadow: hex(0x1A2233),
		ScrimA: 0.30, ShadowA: 0.16, Dark: 0,
	}
}

func (p Pal) Mix(o Pal, t float64) Pal {
	return Pal{
		Bg0: p.Bg0.Mix(o.Bg0, t), Bg1: p.Bg1.Mix(o.Bg1, t),
		Surface: p.Surface.Mix(o.Surface, t), Surface2: p.Surface2.Mix(o.Surface2, t),
		Raised: p.Raised.Mix(o.Raised, t), Well: p.Well.Mix(o.Well, t), Knob: p.Knob.Mix(o.Knob, t),
		Line: p.Line.Mix(o.Line, t), Text: p.Text.Mix(o.Text, t), Text2: p.Text2.Mix(o.Text2, t),
		Text3: p.Text3.Mix(o.Text3, t), Scrim: p.Scrim.Mix(o.Scrim, t), OnAccent: p.OnAccent.Mix(o.OnAccent, t),
		Accent: p.Accent.Mix(o.Accent, t), Ink: p.Ink.Mix(o.Ink, t), Tip: p.Tip.Mix(o.Tip, t),
		TipText: p.TipText.Mix(o.TipText, t), Shadow: p.Shadow.Mix(o.Shadow, t),
		ScrimA: lerp(p.ScrimA, o.ScrimA, t), ShadowA: lerp(p.ShadowA, o.ShadowA, t), Dark: lerp(p.Dark, o.Dark, t),
	}
}
