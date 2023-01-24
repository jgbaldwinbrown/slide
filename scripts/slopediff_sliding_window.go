package main

import (
	"flag"
	"math"
	"strconv"
	"fmt"
	"github.com/jgbaldwinbrown/fasttsv"
	"github.com/montanaflynn/stats"
	"container/list"
	"os"
	"io"
)

type BedEntry struct {
	Chrom string
	Left float64
	Right float64
	Val float64
}

type BedScanner struct {
	Scanner *fasttsv.Scanner
	CurEntry BedEntry
	LastErr error
}

type LineWriter interface {
	Write([]string)
}

func NewBedScanner(s *fasttsv.Scanner) *BedScanner {
	b := &BedScanner{Scanner: s}
	b.Scan()
	return b
}

func (s *BedScanner) Scan() bool {
	ok := s.Scanner.Scan()
	if !ok {
		return ok
	}
	line := s.Scanner.Line()
	s.CurEntry.Chrom = line[0]
	s.CurEntry.Left, s.LastErr = strconv.ParseFloat(line[1], 64)
	if s.LastErr != nil {
		return false
	}
	s.CurEntry.Left--
	s.CurEntry.Right, s.LastErr = strconv.ParseFloat(line[1], 64)
	if s.LastErr != nil {
		return false
	}

	if line[5] == "-nan" {
		s.CurEntry.Val = math.NaN()
	} else {
		s.CurEntry.Val, s.LastErr = strconv.ParseFloat(line[5], 64)
	}
	if s.LastErr != nil {
		return false
	}
	return true
}

func (s *BedScanner) Entry() BedEntry {
	return s.CurEntry
}

type BedOutputScanner interface {
	Scan() bool
	Entry() BedEntry
}

type Slider struct {
	Chrom string
	Left float64
	Mid float64
	Right float64
	Size float64
	StepLen float64
	Items *list.List
	Scanner BedOutputScanner
}

func NewSlider(b BedOutputScanner, size float64, step float64) Slider {
	s := Slider{Size: size, StepLen: step, Scanner: b, Items: list.New()}
	s.Scanner.Scan()
	return s
}

func (s *Slider) StartChrom() {
	s.Chrom = s.Scanner.Entry().Chrom
	s.Left = 0
	s.Mid = (s.Left + s.Right) / 2
	s.Right = s.Size
	for s.Items.Front() != nil {
		s.Items.Remove(s.Items.Front())
	}
}

func (s *Slider) Step() bool {
	if s.Scanner.Entry().Chrom != s.Chrom {
		s.StartChrom()
	}
	// fmt.Println(1)
	for s.Items.Front() != nil && s.Items.Front().Value.(BedEntry).Right <= s.Left {
		s.Items.Remove(s.Items.Front())
		// fmt.Println(2)
	}
	// fmt.Println(3)
	for s.Scanner.Entry().Chrom == s.Chrom && s.Scanner.Entry().Right <= s.Left {
		// fmt.Println(4)
		ok := s.Scanner.Scan()
		if !ok {
			return ok
		}
		// fmt.Println(4.5)
	}

	// fmt.Println(4.6)

	for s.Scanner.Entry().Chrom == s.Chrom && s.Scanner.Entry().Left >= s.Right {
		// fmt.Println(4.7)
		s.Left += s.StepLen
		s.Right += s.StepLen
	}
	// fmt.Println(5)
	if s.Scanner.Entry().Chrom != s.Chrom {
	// fmt.Println(6)
		s.StartChrom()
	}
	// fmt.Println(7)

	for math.Max(s.Scanner.Entry().Left, s.Left) <= math.Min(s.Scanner.Entry().Right, s.Right) {
		// fmt.Println(8)
		s.Items.PushBack(s.Scanner.Entry())
		ok := s.Scanner.Scan()
		if !ok {
			return ok
		}
		for s.Scanner.Entry().Chrom != s.Chrom {
			// fmt.Println(9)
			s.StartChrom()
			// fmt.Println(10)
		}
		// fmt.Println(11)
	}

	// fmt.Println(12)
	return true
}

func (s *Slider) Mean() (float64, error) {
	var vals []float64
	for elem := s.Items.Back(); elem != nil; elem = elem.Prev() {
		v := elem.Value.(BedEntry).Val
		if ! math.IsNaN(v) {
			vals = append(vals, elem.Value.(BedEntry).Val)
		}
	}
	return stats.Mean(vals)
}

func (s *Slider) WriteWindow(w LineWriter) {
	mean, err := s.Mean()
	if err == nil {
		w.Write([]string{s.Chrom, fmt.Sprintf("%d", int(s.Left)), fmt.Sprintf("%d", int(s.Right)), "NA", "NA", fmt.Sprintf("%g", mean)})
	} else {
		w.Write([]string{s.Chrom, fmt.Sprintf("%d", int(s.Left)), fmt.Sprintf("%d", int(s.Right)), "NA", "NA", "NA"})
	}
}

func SlidingMeans(inconn io.Reader, outconn io.Writer, size float64, step float64) {
	b := NewBedScanner(fasttsv.NewScanner(inconn))
	s := NewSlider(b, size, step)
	w := fasttsv.NewWriter(outconn)
	defer w.Flush()

	for s.Step() {
		// fmt.Println(s)
		// fmt.Println(s.Scanner.Entry())
		s.WriteWindow(w)
	}
	// fmt.Printf("%s\n", b.LastErr)
}

func main() {
	winsize_strptr := flag.String("w", "", "Window size for sliding window")
	winstep_strptr := flag.String("s", "", "Window step for sliding window")
	flag.Parse()
	winsize, err := strconv.ParseFloat(*winsize_strptr, 64)
	if err != nil { panic(err) }
	winstep, err := strconv.ParseFloat(*winstep_strptr, 64)
	if err != nil { panic(err) }
	SlidingMeans(os.Stdin, os.Stdout, winsize, winstep)
}
