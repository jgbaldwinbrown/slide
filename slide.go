package slide

import (
	"math"
	"strconv"
	"fmt"
	"github.com/jgbaldwinbrown/fasttsv"
	"github.com/montanaflynn/stats"
	"container/list"
	"io"
)

type ScannerState int64

const (
	AHEAD_OF_WINDOW ScannerState = iota
	BEHIND_WINDOW
	IN_WINDOW
	NEW_CHROM
	STALE_ENTRIES
	DONE
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

	if line[2] == "-nan" {
		s.CurEntry.Val = math.NaN()
	} else {
		s.CurEntry.Val, s.LastErr = strconv.ParseFloat(line[2], 64)
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

func Intersect(query_left, query_right, subject_left, subject_right float64) bool {
	right_edge := math.Min(query_right, subject_right)
	left_edge := math.Max(query_left, subject_left)
	return left_edge < right_edge
}

func Behind(query_left, query_right, subject_left, subject_right float64) bool {
	return query_right <= subject_left
}

func Ahead(query_left, query_right, subject_left, subject_right float64) bool {
	return query_left >= subject_right
}

func (s *Slider) State() ScannerState {
	if s.Scanner.Entry().Chrom != s.Chrom {
		return NEW_CHROM
	}

	if s.Items.Front() != nil && s.Items.Front().Value.(BedEntry).Right <= s.Left {
		return STALE_ENTRIES
	}

	if Intersect(s.Scanner.Entry().Left, s.Scanner.Entry().Right, s.Left, s.Right) {
		return IN_WINDOW
	}

	if Behind(s.Scanner.Entry().Left, s.Scanner.Entry().Right, s.Left, s.Right) {
		return BEHIND_WINDOW
	}

	if Ahead(s.Scanner.Entry().Left, s.Scanner.Entry().Right, s.Left, s.Right) {
		return AHEAD_OF_WINDOW
	}

	return DONE
}

func (s *Slider) Step() bool {
	if s.State() != NEW_CHROM {
		s.Left += s.StepLen
		s.Right += s.StepLen
	}
	for true {
		switch s.State() {
		case STALE_ENTRIES:
			s.Items.Remove(s.Items.Front())
		case NEW_CHROM:
			s.StartChrom()
		case IN_WINDOW:
			s.Items.PushBack(s.Scanner.Entry())
			ok := s.Scanner.Scan()
			if !ok {
				return ok
			}
		case BEHIND_WINDOW:
			ok := s.Scanner.Scan()
			if !ok {
				return ok
			}
		case AHEAD_OF_WINDOW:
			return true
		default:
			return true
		}
	}
	return false
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
		w.Write([]string{s.Chrom, fmt.Sprintf("%d", int(s.Left)), fmt.Sprintf("%d", int(s.Right)), fmt.Sprintf("%g", mean)})
	} else {
		w.Write([]string{s.Chrom, fmt.Sprintf("%d", int(s.Left)), fmt.Sprintf("%d", int(s.Right)), "NA"})
	}
}

func SlidingMeans(inconn io.Reader, outconn io.Writer, size float64, step float64) {
	b := NewBedScanner(fasttsv.NewScanner(inconn))
	s := NewSlider(b, size, step)
	w := fasttsv.NewWriter(outconn)
	defer w.Flush()

	for s.Step() {
		s.WriteWindow(w)
	}
}
