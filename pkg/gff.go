package slide

import (
	"bufio"
	"encoding/csv"
	"strings"
	"math"
	"strconv"
	"fmt"
	"io"
	"regexp"
)

func handle(format string) func(...any) error {
	return func(args ...any) error {
		return fmt.Errorf(format, args...)
	}
}

var commentre = regexp.MustCompile(`^#`)

type GffFields struct {
	IsComment bool
	Source string
	Type string
	Score float64
	Strand byte
	Phase byte
	AttributeNames []string
	Attributes map[string]string
}

func GffComment() BedEntry {
	return BedEntry{
		Other: GffFields {
			IsComment: true,
		},
	}
}

func ParseGffAttributes(field string) ([]string, map[string]string, error) {
	var names []string
	vals := map[string]string{}

	fields := strings.Split(field, ";")
	for _, f := range fields {
		if len(f) < 1 {
			continue
		}
		eq := strings.Split(f, "=")
		if len(eq) != 2 {
			names = append(names, f)
			continue
		}
		names = append(names, eq[0])
		vals[eq[0]] = eq[1]
	}
	return names, vals, nil
}

func ParseGffLine(line []string) (BedEntry, error) {
	h := handle("ParseGffLine: %w")

	if len(line) > 0 && commentre.MatchString(line[0]) {
		return GffComment(), nil
	}

	if len(line) != 9 {
		return BedEntry{}, h(fmt.Errorf("len(line) %v != 9", len(line)))
	}

	var b BedEntry
	var g GffFields
	var e error

	b.Chrom = line[0]

	b.Left, e = strconv.ParseFloat(line[3], 64)
	if e != nil { return b, h(e) }
	b.Left--

	b.Right, e = strconv.ParseFloat(line[4], 64)
	if e != nil { return b, h(e) }

	g.Source = line[1]
	g.Type = line[2]
	g.Score, e = strconv.ParseFloat(line[5], 64)
	if e != nil { g.Score = math.NaN() }

	if len(line[6]) != 1 { return b, h(fmt.Errorf("len of strand field %v != 1", line[6])) }
	g.Strand = line[6][0]

	if len(line[7]) != 1 { return b, h(fmt.Errorf("len of phase field %v != 1", line[7])) }
	g.Phase = line[7][0]

	g.AttributeNames, g.Attributes, e = ParseGffAttributes(line[8])
	if e != nil { return b, h(e) }

	b.Other = g
	return b, nil
}

type GffScanner struct {
	cr *csv.Reader
	line []string
	e BedEntry
	err error
	closed bool
}

func NewGffScanner(r io.Reader) (*GffScanner) {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true
	cr.Comma = rune('\t')
	cr.ReuseRecord = false
	cr.FieldsPerRecord = -1
	return &GffScanner{cr: cr}
}

func (s *GffScanner) Scan() bool {
	var err error
	s.line, err = s.cr.Read()
	if err != nil {
		if err != io.EOF {
			s.err = err
		}
		s.closed = true
		return false
	}
	s.e, err = ParseGffLine(s.line)
	if err != nil {
		s.err = err
		s.closed = true
		return false
	}
	return true
}

func (s *GffScanner) Entry() BedEntry {
	return s.e
}

func (s *GffScanner) Error() error {
	return s.err
}

func (s *GffScanner) Line() []string {
	return s.line
}

func SlidingGffEntryCount(in BedOutputScanner, size float64, step float64) <-chan BedEntry {
	s := NewSlider(in, size, step)
	out := make(chan BedEntry, 256)

	go func() {
		for s.Step() {
			entry, e := s.MeanEntry()
			if e != nil {
				panic(e)
			}

			count := 0.0
			for n := s.Items.Front(); n != nil; n = n.Next() {
				if !n.Value.(BedEntry).Other.(GffFields).IsComment {
					count++
				}
			}
			entry.Val = count
			out <- entry
		}
		close(out)
	}()
	return out
}

func SlidingGffBpCovered(in BedOutputScanner, size float64, step float64) <-chan BedEntry {
	s := NewSlider(in, size, step)
	out := make(chan BedEntry, 256)

	go func() {
		for s.Step() {
			entry, e := s.MeanEntry()
			if e != nil {
				panic(e)
			}

			covered := map[int64]struct{}{}
			for n := s.Items.Front(); n != nil; n = n.Next() {
				b := n.Value.(BedEntry)
				start := int64(b.Left)
				end := int64(b.Right)
				for i := start; i < end; i++ {
					covered[i] = struct{}{}
				}
			}
			entry.Val = float64(len(covered))
			out <- entry
		}
		close(out)
	}()
	return out
}

func WriteEntry(w io.Writer, b BedEntry) error {
	_, e := fmt.Fprintf(w, "%v\t%d\t%d\t%v\n", b.Chrom, int(b.Left), int(b.Right), b.Val)
	return e
}

func SlidingGffEntryCountFull(inconn io.Reader, outconn io.Writer, size float64, step float64) error {
	b := NewGffScanner(inconn)
	w := bufio.NewWriter(outconn)
	defer w.Flush()

	slid := SlidingGffEntryCount(b, size, step)

	for entry := range slid {
		e := WriteEntry(w, entry)
		if e != nil { return e }
	}
	return nil
}

func SlidingGffBpCoveredFull(inconn io.Reader, outconn io.Writer, size float64, step float64) error {
	b := NewGffScanner(inconn)
	w := bufio.NewWriter(outconn)
	defer w.Flush()

	slid := SlidingGffBpCovered(b, size, step)

	for entry := range slid {
		e := WriteEntry(w, entry)
		if e != nil { return e }
	}
	return nil
}
