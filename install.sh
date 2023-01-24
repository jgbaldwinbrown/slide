#!/bin/bash
set -e

(cd cmd && (
	ls *.go | while read i ; do
		go build $i
	done
))

(cd scripts && (
	ls *.go | while read i ; do
		go build $i
	done
))

cp ./scripts/fst_sliding_window ~/mybin
cp ./scripts/cov_sliding_window ~/mybin
cp ./scripts/bedspan ~/mybin
cp ./cmd/lib_fst_sliding_window ~/mybin
