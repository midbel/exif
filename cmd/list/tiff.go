package main

import (
	"fmt"

	"github.com/midbel/exif/nef"
)

var tiff = map[uint16]Value{
	0xfe:   makeValue("NewSubfileType", subfileType),
	0x100:  makeValue("ImageWidth", imagePixels),
	0x101:  makeValue("ImageLength", imagePixels),
	0x102:  makeValue("BitsPerSample", nil),
	0x103:  makeValue("Compression", tiffCompression),
	0x106:  makeValue("PhotometricInterpretation", photometricInterpretation),
	0x10f:  makeValue("Make", nil),
	0x110:  makeValue("Model", nil),
	0x111:  makeValue("StripOffsets", nil),
	0x112:  makeValue("Orientation", imageOrientation),
	0x115:  makeValue("SamplesPerPixel", nil),
	0x116:  makeValue("RowsPerStrip", nil),
	0x117:  makeValue("StripByteCount", nil),
	0x11a:  makeValue("XResolution", nil),
	0x11b:  makeValue("YResolution", nil),
	0x11c:  makeValue("PlanarConfiguration", planarConfiguration),
	0x128:  makeValue("ResolutionUnit", resolutionUnit),
	0x131:  makeValue("Software", nil),
	0x132:  makeValue("DateTime", nil),
	0x13b:  makeValue("Artist", nil),
	0x14a:  makeValue("SubIFDS", nil),
	0x201:  makeValue("JpegFromRawStart", nil),
	0x202:  makeValue("JpegFromRawLength", nil),
	0x213:  makeValue("YCbCrPositioning", ycbcrPositioning),
	0x214:  makeValue("ReferenceBlackWhite", nil),
	0x2bc:  makeValue("XMP", nil),
	0x828d: makeValue("CFARepeatPatternDim", nil),
	0x828e: makeValue("CFAPattern", nil),
	0x8298: makeValue("Copyright", nil),
	0x8769: makeValue("ExifIFD", nil),
	0x8825: makeValue("GPSIFD", nil),
	0x9003: makeValue("DateTimeOriginal", nil),
	0x9216: makeValue("EPStandardID", nil),
	0x9217: makeValue("SensingMethod", nil),
}

func subfileType(t nef.Tag) interface{} {
	switch t.Uint() {
	case 0:
		return "full resolution image"
	case 1:
		return "reduced resolution image"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func tiffCompression(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "uncompressed"
	case 6:
		return "jpeg"
	case 34713:
		return "nikon nef compressed"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func photometricInterpretation(t nef.Tag) interface{} {
	switch t.Uint() {
	case 0:
		return "white"
	case 1:
		return "black"
	case 2:
		return "rgb"
	case 3:
		return "palette"
	case 4:
		return "mask"
	case 5:
		return "cmyk"
	case 6:
		return "ycbcr"
	case 32803:
		return "color array"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func imagePixels(t nef.Tag) interface{} {
	return fmt.Sprintf("%dpx", t.Uint())
}

func imageOrientation(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "horizontal"
	case 2:
		return "mirror horizontal"
	case 3:
		return "rotate 180°"
	case 4:
		return "mirror vertical"
	case 5:
		return "mirror horizontal + rotate 270° CW"
	case 6:
		return "rotate 90°"
	case 7:
		return "mirror horizontal + rotate 90° CW"
	case 8:
		return "rotate 270° CW"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func planarConfiguration(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "chunky"
	case 2:
		return "planar"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func resolutionUnit(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "none"
	case 2:
		return "inch"
	case 3:
		return "cm"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func ycbcrPositioning(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "centered"
	case 2:
		return "co-sited"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}
