package main

import (
	"os"
	"math"
	"github.com/jgbaldwinbrown/slide/pkg"
	"bufio"
	"fmt"
	"flag"
)

func AbsFilter(s slide.BedOutputScanner) *slide.BedEntryScanner {
	outc := make(chan slide.BedEntry, 256)

	go func() {
		for s.Scan() {
			entry := s.Entry()
			entry.Val = math.Abs(entry.Val)
			outc <- entry
		}
		close(outc)
	}()

	return slide.NewBedEntryScanner(outc)
}

func Log10(s slide.BedOutputScanner) *slide.BedEntryScanner {
	outc := make(chan slide.BedEntry, 256)

	go func() {
		for s.Scan() {
			entry := s.Entry()
			entry.Other = math.Log10(entry.Val)
			outc <- entry
		}
		close(outc)
	}()

	return slide.NewBedEntryScanner(outc)
}

func PrintWins(s slide.BedOutputScanner) error {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for s.Scan() {
		entry := s.Entry()
		_, e := fmt.Printf("%v\t%d\t%d\t%v\t%v\n", entry.Chrom, int(entry.Left), int(entry.Right), entry.Val, entry.Other.(float64))
		if e != nil {
			return fmt.Errorf("PrintWins: %w", e)
		}
	}
	return nil
}

func main() {
	sizep := flag.Int("s", 1, "Window size")
	stepp := flag.Int("t", 1, "Window step")
	flag.Parse()

	bedscan := slide.NewBedReaderScanner(os.Stdin)
	abs := AbsFilter(bedscan)
	wins := slide.NewBedEntryScanner(slide.SlidingEntryMeans(abs, float64(*sizep), float64(*stepp)))
	logged := Log10(wins)
	err := PrintWins(logged)
	if err != nil {
		panic(err)
	}
}
