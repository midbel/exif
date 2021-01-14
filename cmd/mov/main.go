package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/exif/mov"
)

func main() {
	flag.Parse()

	file, err := mov.Decode(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	p, err := file.DecodeProfile()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println(p.AcqTime())
	fmt.Println(p.ModTime())
	fmt.Println(p.Length())
}
