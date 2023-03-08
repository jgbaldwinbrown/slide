package main

import (
	"os"
	"github.com/jgbaldwinbrown/slide/pkg"
	"flag"
)

func main() {
	sizep := flag.Int("s", 1, "Window size")
	stepp := flag.Int("t", 1, "Window step")
	flag.Parse()

	e := slide.SlidingGffBpCoveredFull(os.Stdin, os.Stdout, float64(*sizep), float64(*stepp))
	if e != nil { panic(e) }
}
