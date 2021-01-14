package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/midbel/exif/nef"
)

func main() {
	dir := flag.String("d", "", "directory")
	flag.Parse()
	for _, a := range flag.Args() {
		if err := extract(a, *dir); err != nil {
			fmt.Fprintf(os.Stdout, "%s: %s\n", a, err)
		}
	}
}

func extract(file, dir string) error {
	dir, err := mkdir(dir, file)
	if err != nil {
		return err
	}
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
		if err := extractImages(files[i], dir); err != nil {
			return err
		}
	}
	return nil
}

func writeImage(f *nef.File) ([]byte, error) {
	img, err := f.Image()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), err
}

func writeBytes(f *nef.File) ([]byte, error) {
	return f.Bytes()
}

func extractImages(f *nef.File, dir string) error {
	var (
		buf []byte
		err error
		ext string
	)
	if f.IsSupported() {
		buf, err = writeImage(f)
		ext = ExtJPG
	} else {
		buf, err = writeBytes(f)
		ext = ExtDAT
	}
	if err != nil {
		return err
	}
	file := filepath.Join(dir, f.Filename()) + ext
	if err := ioutil.WriteFile(file, buf, 0644); err != nil {
		return err
	}
	fmt.Printf("extracted %s (%d KB) from %s\n", file, len(buf)>>10, f.Directory())
	for i := range f.Files {
		if err := extractImages(f.Files[i], dir); err != nil {
			return err
		}
	}
	return nil
}

func mkdir(dir, file string) (string, error) {
	var (
		ext  = filepath.Ext(file)
		base = filepath.Base(file)
	)
	dir = filepath.Join(dir, strings.TrimRight(base, ext))
	return dir, os.MkdirAll(dir, 0755)
}
