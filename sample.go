package main

import "time"

// fakeRecords: önizleme ve testler için örnek oturum geçmişi.
func fakeRecords(s *Settings, now time.Time) {
	tasks := []string{"Devre analizi ödevi", "Sinyaller ve sistemler", "TEG proje raporu", "Mikroişlemciler lab", ""}
	mins := [][]int{{25, 25, 25}, {25, 25}, {}, {25, 25, 25, 25, 25}, {25, 25, 25}, {25, 25, 25, 25}, {25, 25, 25}}
	for i, list := range mins {
		d := dayStart(now).AddDate(0, 0, -(6 - i))
		for k, m := range list {
			end := d.Add(time.Duration(9+k) * time.Hour).Add(time.Duration(k*7) * time.Minute)
			if i == 6 {
				end = now.Add(-time.Duration(len(list)-k) * 40 * time.Minute)
			}
			s.Records = append(s.Records, Record{End: end.Unix(), Sec: m * 60, Task: tasks[(i+k)%len(tasks)], Full: true})
		}
	}
	s.Records = append(s.Records, Record{End: now.Add(-10 * time.Minute).Unix(), Sec: 12 * 60, Task: "Devre analizi ödevi", Full: false})
}
