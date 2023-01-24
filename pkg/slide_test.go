package slide

import (
	"github.com/jgbaldwinbrown/fasttsv"
	"testing"
	"strings"
	"reflect"
)

var in1 = `chr1	0	1	1
chr1	1	2	1
chr1	2	3	1
chr1	3	4	1
chr1	4	5	1`

var in2 = `chr1	0	1	1
chr1	1	2	1
chr1	2	3	1
chr1	3	4	1
chr1	4	5	1
chr2	0	1	1
chr2	1	2	1
chr2	2	3	1
chr2	3	4	1
chr2	4	5	1`

var in3 = `chr1	0	3	4
chr1	1	2	1
chr1	2	3	1
chr1	3	4	1
chr1	4	5	1`

var expect1 = []BedEntry {
	BedEntry {
		Chrom: "chr1",
		Left: 0,
		Right: 2,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 1,
		Right: 3,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 2,
		Right: 4,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 3,
		Right: 5,
		Val: 1,
	},
}

var expect2 = []BedEntry {
	BedEntry {
		Chrom: "chr1",
		Left: 0,
		Right: 2,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 1,
		Right: 3,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 2,
		Right: 4,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 3,
		Right: 5,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr2",
		Left: 0,
		Right: 2,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr2",
		Left: 1,
		Right: 3,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr2",
		Left: 2,
		Right: 4,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr2",
		Left: 3,
		Right: 5,
		Val: 1,
	},
}

var expect3 = []BedEntry {
	BedEntry {
		Chrom: "chr1",
		Left: 0,
		Right: 2,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 1,
		Right: 3,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 2,
		Right: 4,
		Val: 1,
	},
	BedEntry {
		Chrom: "chr1",
		Left: 3,
		Right: 5,
		Val: 1,
	},
}

type SlideTester struct {
	Name string
	In string
	Winsize float64
	Winstep float64
	Expect []BedEntry
	Func func(in string, winsize float64, winstep float64) []BedEntry
}

var Tests = []SlideTester {
	SlideTester {
		Name: "simple",
		In: in1,
		Expect: expect1,
		Winsize: 2,
		Winstep: 1,
		Func: MeansTest,
	},
	SlideTester {
		Name: "2chrom",
		In: in2,
		Expect: expect2,
		Winsize: 2,
		Winstep: 1,
		Func: MeansTest,
	},
	SlideTester {
		Name: "2span",
		In: in3,
		Expect: expect3,
		Winsize: 2,
		Winstep: 1,
		Func: MeansTest,
	},
}

func MeansTest(in string, size float64, step float64) []BedEntry {
	b := NewBedScanner(fasttsv.NewScanner(strings.NewReader(in)))
	s := NewSlider(b, size, step)
	var out []BedEntry

	for s.Step() {
		e, err := s.MeanEntry()
		if err != nil {
			panic(err)
		}
		out = append(out, e)
	}
	return out
}

func TestSlide(t *testing.T) {
	for _, test := range Tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			out := test.Func(test.In, test.Winsize, test.Winstep)
			if !reflect.DeepEqual(out, test.Expect) {
				t.Errorf("out %v != expect %v", out, test.Expect)
			}
		})
	}
}
