package main

import (
	"flag"
	"strconv"
	"os"
	"github.com/jgbaldwinbrown/slide/pkg"
)

func main() {
	winsize_strptr := flag.String("w", "", "Window size for sliding window")
	winstep_strptr := flag.String("s", "", "Window step for sliding window")
	flag.Parse()
	winsize, err := strconv.ParseFloat(*winsize_strptr, 64)
	if err != nil { panic(err) }
	winstep, err := strconv.ParseFloat(*winstep_strptr, 64)
	if err != nil { panic(err) }
	slide.SlidingMeans(os.Stdin, os.Stdout, winsize, winstep)
}
