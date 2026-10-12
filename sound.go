package main

import (
	"encoding/binary"
	"math"
)

var soundName = [4]string{"Zil", "Çan", "Dijital", "Kapalı"}

// MakeWAV: bildirim sesini bellekte 16-bit mono PCM WAV olarak üretir.
func MakeWAV(kind, volume int) []byte {
	const sr = 44100
	var dur float64
	switch kind {
	case 0:
		dur = 2.2
	case 1:
		dur = 1.9
	default:
		dur = 1.0
	}
	n := int(dur * sr)
	buf := make([]float64, n)

	// bell: inarmonik kısmi tonlarla tek vuruş
	bell := func(at, f, amp float64) {
		ratios := []float64{1, 2.0, 2.76, 4.07, 5.43}
		amps := []float64{1, 0.55, 0.32, 0.2, 0.12}
		decays := []float64{1.5, 0.95, 0.6, 0.42, 0.3}
		s0 := int(at * sr)
		for i := s0; i < n; i++ {
			t := float64(i-s0) / sr
			att := math.Min(1, t/0.004)
			v := 0.0
			for k := range ratios {
				v += amps[k] * math.Exp(-t/decays[k]) * math.Sin(2*math.Pi*f*ratios[k]*t)
			}
			buf[i] += amp * att * v
		}
	}
	// pluck: yumuşak marimba benzeri ton
	pluck := func(at, f, amp float64) {
		s0 := int(at * sr)
		for i := s0; i < n; i++ {
			t := float64(i-s0) / sr
			att := math.Min(1, t/0.006)
			env := math.Exp(-t / 0.55)
			v := math.Sin(2*math.Pi*f*t) + 0.18*math.Exp(-t/0.12)*math.Sin(2*math.Pi*f*4*t)
			buf[i] += amp * att * env * v
		}
	}
	// beep: kısa, yuvarlatılmış kare dalga
	beep := func(at, f, length, amp float64) {
		s0, s1 := int(at*sr), int((at+length)*sr)
		for i := s0; i < s1 && i < n; i++ {
			t := float64(i-s0) / sr
			env := math.Min(1, t/0.004) * math.Min(1, (length-t)/0.01)
			v := 0.0
			for h := 1; h <= 7; h += 2 {
				v += math.Sin(2*math.Pi*f*float64(h)*t) / float64(h)
			}
			buf[i] += amp * env * v
		}
	}

	switch kind {
	case 0:
		bell(0, 880, 1)
		bell(0.42, 1174.66, 0.85)
	case 1:
		pluck(0, 783.99, 0.8)
		pluck(0.15, 1046.5, 0.8)
		pluck(0.30, 1318.51, 0.8)
		pluck(0.45, 1567.98, 0.9)
	default:
		for g := 0; g < 2; g++ {
			for k := 0; k < 3; k++ {
				beep(float64(g)*0.5+float64(k)*0.12, 1760, 0.075, 0.6)
			}
		}
	}

	peak := 1e-9
	for _, v := range buf {
		peak = math.Max(peak, math.Abs(v))
	}
	gain := 0.92 * float64(clampi(volume, 0, 100)) / 100 / peak
	fade := int(0.05 * sr)

	data := make([]byte, 44+2*n)
	copy(data[0:], "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+2*n))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], sr)
	binary.LittleEndian.PutUint32(data[28:], sr*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(2*n))
	for i, v := range buf {
		v *= gain
		if i > n-fade {
			v *= float64(n-i) / float64(fade)
		}
		s := int16(math.Max(-32767, math.Min(32767, v*32767)))
		binary.LittleEndian.PutUint16(data[44+2*i:], uint16(s))
	}
	return data
}
