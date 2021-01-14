package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/busoc/nef"
)

func main() {
	flag.Parse()
	for _, a := range flag.Args() {
		if err := readFile(a); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", a, err)
		}
	}
}

func readFile(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	files, err := nef.Decode(r)
	if err != nil {
		return err
	}
	for i := range files {
		printStat(filepath.Clean(file), files[i])
	}
	return nil
}

const (
	mpat = "%s (%s): %3d tiff - %3d exif - %3d note (sub directories: %d)"
	spat = "- %d: %s (%s): %3d tiff"
)

const (
	raw = "raw"
	jpg = "jpg"
)

func printStat(file string, f *nef.File) {
	var (
		tiff = f.TagsFor(nef.Tiff)
		exif = f.TagsFor(nef.Exif)
		note = f.TagsFor(nef.Note)
		typ  = f.ImageType()
	)

	fmt.Println(file)
	fmt.Printf(mpat, f.Directory(), typ, len(tiff), len(exif), len(note), len(f.Files))
	fmt.Println()
	for i := range f.Files {
		var (
			tiff = f.Files[i].TagsFor(nef.Tiff)
			typ  = f.Files[i].ImageType()
		)
		fmt.Printf(spat, i+1, f.Files[i].Directory(), typ, len(tiff))
		fmt.Println()
	}
}
