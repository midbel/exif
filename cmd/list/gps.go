package main

import (
	"strings"

	"github.com/midbel/exif/nef"
)

var gps = map[uint16]Value{
	0x0: makeValue("GPSVersionId", gpsVersionId),
}

func gpsVersionId(t nef.Tag) interface{} {
	vs, err := t.Values()
	if err != nil {
		return err
	}
	return strings.Join(vs, ".")
}
