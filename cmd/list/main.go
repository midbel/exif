package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/midbel/exif/nef"
)

type Value struct {
	Name      string
	Transform func(nef.Tag) interface{}
}

func makeValue(str string, fn func(nef.Tag) interface{}) Value {
	if fn == nil {
		fn = noop
	}
	return Value{
		Name:      str,
		Transform: fn,
	}
}

func noop(t nef.Tag) interface{} {
	vs, err := t.Values()
	if err != nil {
		return err
	}
	switch len(vs) {
	case 0:
		return nil
		// return "<empty>"
	case 1:
		return vs[0]
	default:
		return strings.Join(vs, ", ")
	}
}

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
	if err == nil {
		for i := range files {
			if i > 0 {
				fmt.Println("===")
			}
			listTagsFromFile(files[i])
		}
	}
	return err
}

const pat = "%s: %03d) id: %32s (0x%04x), source: %6s, type: %12s, len: %6d, offset: %12d, values: %v"

func listTagsFromFile(f *nef.File) {
	dir := f.Directory()
	printTags(dir, f.TagsFor(nef.Tiff), tiff)
	printTags(dir, f.TagsFor(nef.Exif), exif)
	printTags(dir, f.TagsFor(nef.Note), notes)
	printTags(dir, f.TagsFor(nef.Gps), gps)
	for i := range f.Files {
		fmt.Println("---")
		printTags(f.Files[i].Directory(), f.Files[i].TagsFor(nef.Tiff), tiff)
	}
}

func printTags(dir string, tags []nef.Tag, tagnames map[uint16]Value) {
	for i, t := range tags {
		var (
			str    string
			values interface{}
		)
		v, ok := tagnames[t.Id]
		if !ok {
			str = "<unknown>"
			values = "<undefined>"
		} else {
			str = v.Name
			values = v.Transform(t)
		}
		fmt.Printf(pat, dir, i+1, str, t.Id, t.Origin(), t.Type, t.Count, t.Offset, values)
		fmt.Println()
	}
}
