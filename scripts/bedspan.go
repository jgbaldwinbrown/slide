package main

import (
	"io/ioutil"
	"os/exec"
	"errors"
	"bufio"
	"sort"
	"fmt"
	"github.com/jgbaldwinbrown/fasttsv"
	"io"
	"strconv"
	"os"
	"flag"
)

type span struct {
	chrom string
	start int
	end int
}

type flag_args struct {
	high_threshold float64
	low_threshold float64
	multi_file bool
	suffix string
	intersect string
	left_extend int
	right_extend int
}

type hit_entry struct {
	line []string
	chrom string
	pos int
	s_val float64
}

type hit_list []hit_entry

func (l hit_list) Len() int {return len(l)}
func (l hit_list) Swap(i, j int) {l[i], l[j] = l[j], l[i]}

type ByChromAndPos struct {hit_list}

func (s ByChromAndPos) Less(i, j int) bool {
	a := s.hit_list[i]
	b := s.hit_list[j]
	if a.chrom < b.chrom {
		return true
	} else if a.chrom > b.chrom {
		return false
	} else {
		return a.pos < b.pos
	}
}

func get_flags() flag_args {
	var out flag_args
	high_str := flag.String("H", "", "High threshold for elements.")
	low_str := flag.String("L", "", "Low threshold for elements.")
	flag.BoolVar(&out.multi_file, "m", false, "Operate on multiple (listed) files.")
	flag.StringVar(&out.intersect, "i", "", "Path to put intersection output.")
	flag.StringVar(&out.suffix, "s", "bedspan_out", "Suffix for output files.")
	flag.IntVar(&out.left_extend, "l", 0, "Extend bed entries left by this amount.")
	flag.IntVar(&out.right_extend, "r", 0, "Extend bed entries right by this amount.")
	flag.Parse()
	var err error
	out.high_threshold, err = strconv.ParseFloat(*high_str, 64)
	if err != nil {panic(err)}
	out.low_threshold, err = strconv.ParseFloat(*low_str, 64)
	if err != nil {panic(err)}
	return out
}

func get_hits(r io.Reader, high_threshold float64, low_threshold float64) (hit_list, error) {
	var out hit_list
	var len_err error
	scanner := fasttsv.NewScanner(r)
	scanner.Scan()
	for scanner.Scan() {
		if len(scanner.Line()) < 3 {
			len_err = errors.New("Error: Line does not have enough field")
			return out, len_err
		}
		s_val, err := strconv.ParseFloat(scanner.Line()[2], 64)
		if err != nil {
			continue
		}
		pos, err := strconv.Atoi(scanner.Line()[1])
		if s_val >= high_threshold || s_val <= low_threshold {
			hit_line := make([]string, len(scanner.Line()))
			copy(hit_line, scanner.Line())
			chrom := scanner.Line()[0]
			if err != nil {continue}
			out = append(out, hit_entry{hit_line, chrom, pos, s_val})
		}
	}
	return out, len_err
}

func get_spans_from_sorted_hits(h hit_list) []span {
	var out []span
	prev_chrom := ""
	var s span
	for _, hit := range h {
		if hit.chrom != prev_chrom {
			if prev_chrom != "" {
				out = append(out, s)
			}
			s = span{chrom: hit.chrom, start: hit.pos - 1, end: hit.pos}
		} else {
			s.end = hit.pos
		}
		prev_chrom = hit.chrom
	}
	out = append(out, s)
	return out
}

func extend_spans(s []span, left int, right int) []span {
	var out []span
	for _, aspan := range s {
		if aspan.start - left >= 0 {
			aspan.start -= left
		} else {
			aspan.start = 0
		}
		aspan.end += right
		out = append(out, aspan)
	}
	return out
}

func print_spans(spans []span, w io.Writer) {
	for _, s := range spans {
		fmt.Fprintf(w, "%s\t%d\t%d\n", s.chrom, s.start, s.end)
	}
}

func get_span(r io.Reader, w io.Writer, high_threshold float64, low_threshold float64, left_extend int, right_extend int) error {
	hits, err := get_hits(r, high_threshold, low_threshold)
	if err != nil {
		return err
	}
	sort.Sort(ByChromAndPos{hits})
	spans := get_spans_from_sorted_hits(hits)
	long_spans := extend_spans(spans, left_extend, right_extend)
	print_spans(long_spans, w)
	return err
}

func get_spans_multi_file(r io.Reader, high_threshold float64, low_threshold float64, suffix string, left_extend int, right_extend int) []string {
	var out []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		file, err := os.Open(scanner.Text())
		if err != nil {panic(err)}
		defer file.Close()
		outfile, err := os.Create(scanner.Text() + "_" + suffix)
		if err != nil {panic(err)}
		defer outfile.Close()
		err = get_span(file, outfile, high_threshold, low_threshold, left_extend, right_extend)
		if err != nil {
			err = os.Remove(scanner.Text() + "_" + suffix)
			if err != nil {
				panic(err)
			}
		} else {
			out = append(out, scanner.Text() + "_" + suffix)
		}
	}
	return out
}

func intersect_all_paths(paths []string, w io.Writer) {
	input_temp, err := ioutil.TempFile("./", "input_temp")
	if err != nil { panic(err) }

	output_temp, err := ioutil.TempFile("./", "output_temp")
	if err != nil { panic(err) }

	defer os.Remove(input_temp.Name())
	defer os.Remove(output_temp.Name())
	in_name := input_temp.Name()
	out_name := output_temp.Name()
	if len(paths) < 2 { return }

	cur_conn, err := os.Open(paths[0])
	if err != nil { panic(err) }

	defer cur_conn.Close()
	_, err = io.Copy(input_temp, cur_conn)
	if err != nil { panic(err) }

	input_temp.Close()
	output_temp.Close()
	for _, path := range paths[1:] {
		output_conn, err := os.Create(out_name)
		if err != nil { panic(err) }

		command := []string{"bedtools", "intersect", "-a", in_name, "-b", path}
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout = output_conn
		cmd.Run()
		output_conn.Close()

		input_conn, err := os.Create(in_name)
		if err != nil {panic(err)}
		output_conn, err = os.Open(out_name)
		io.Copy(input_conn, output_conn)
		input_conn.Close()
		output_conn.Close()
	}
	output_conn, err := os.Open(out_name)
	if err != nil {panic(err)}
	io.Copy(w, output_conn)
	output_conn.Close()
}

func main() {
	flags := get_flags()
	if ! flags.multi_file {
		_ = get_span(os.Stdin, os.Stdout, flags.high_threshold, flags.low_threshold, flags.left_extend, flags.right_extend)
	} else {
		output_paths := get_spans_multi_file(os.Stdin, flags.high_threshold, flags.low_threshold, flags.suffix, flags.left_extend, flags.right_extend)
		if flags.intersect != "" {
			intersect_conn, err := os.Create(flags.intersect)
			if err != nil {
				panic(err)
			}
			defer intersect_conn.Close()
			intersect_all_paths(output_paths, intersect_conn)
		}
	}
}
