package main

import (
	"fmt"
	"strings"

	"github.com/midbel/exif/nef"
)

var exif = map[uint16]Value{
	0x829a: makeValue("ExposureTime", nil),
	0x829d: makeValue("FNumber", nil),
	0x8822: makeValue("ExposureProgram", nil),
	0x8827: makeValue("ISO", nil),
	0x8830: makeValue("SensitivityType", nil),
	0x9003: makeValue("DateTimeOriginal", nil),
	0x9004: makeValue("CreateDate", nil),
	0x9204: makeValue("ExposureCompensation", nil),
	0x9205: makeValue("MaxApertureValue", nil),
	0x9207: makeValue("MeteringMode", nil),
	0x9208: makeValue("LightSource", nil),
	0x9209: makeValue("Flash", nil),
	0x920a: makeValue("FocalLength", nil),
	0x927c: makeValue("MakerNote", makerNote),
	0x9286: makeValue("UserComment", userComment),
	0x9290: makeValue("SubSecTime", nil),
	0x9291: makeValue("SubSecTimeOriginal", nil),
	0x9292: makeValue("SubSecTimeDigitized", nil),
	0xa217: makeValue("SensingMethod", nil),
	0xa300: makeValue("FileSource", nil),
	0xa301: makeValue("SceneType", nil),
	0xa302: makeValue("CFAPattern", nil),
	0xa401: makeValue("CustomRendered", nil),
	0xa402: makeValue("ExposureMode", nil),
	0xa403: makeValue("WhiteBalance", nil),
	0xa404: makeValue("DigitalZoomRatio", nil),
	0xa405: makeValue("FocalLengthIn35mmFormat", nil),
	0xa406: makeValue("SceneCaptureType", nil),
	0xa407: makeValue("GainControl", nil),
	0xa408: makeValue("Contrast", nil),
	0xa409: makeValue("Saturation", nil),
	0xa40a: makeValue("Sharpness", nil),
	0xa40c: makeValue("SubjectDistanceRange", nil),
}

func userComment(t nef.Tag) interface{} {
	str := t.String()
	return strings.TrimLeft(str, "ASCII\x00\x00")
}

func makerNote(t nef.Tag) interface{} {
	maker := strings.TrimRight(string(t.Raw[:6]), "\x00")
	return fmt.Sprintf("%s 0x%04x", maker, t.Raw[6:8])
}
