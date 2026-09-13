package gifdecode

import "testing"

func BenchmarkDecodeSmall(b *testing.B) {
	data := makeTestGIF(4)
	opts := DefaultOptions()
	for b.Loop() {
		if _, err := Decode(data, opts); err != nil {
			b.Fatalf("decode failed: %v", err)
		}
	}
}

func BenchmarkDecodeMedium(b *testing.B) {
	data := makeMediumGIF()
	opts := DefaultOptions()
	for b.Loop() {
		if _, err := Decode(data, opts); err != nil {
			b.Fatalf("decode failed: %v", err)
		}
	}
}

func BenchmarkDecodeFixtures(b *testing.B) {
	fixtures := []struct {
		name string
		file string
	}{
		{name: "animexample2", file: "animexample2.gif"},
		{name: "youtube-loading-3", file: "youtube-loading-3.gif"},
		{name: "knowledge-human-pink", file: "knowledge-human-pink.gif"},
		{name: "animation-loading1", file: "animation-loading1.gif"},
		{name: "walk-cycle", file: "walk-cycle.gif"},
	}
	opts := DefaultOptions()
	for _, fixture := range fixtures {
		data := readFixture(b, fixture.file)
		b.Run(fixture.name, func(b *testing.B) {
			for b.Loop() {
				if _, err := Decode(data, opts); err != nil {
					b.Fatalf("decode failed: %v", err)
				}
			}
		})
	}
}
