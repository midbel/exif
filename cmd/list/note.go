package main

import (
	"fmt"
	"math/bits"

	"github.com/midbel/exif/nef"
)

var notes = map[uint16]Value{
	0x1:  makeValue("MakerNoteVersion", makerNoteVersion),
	0x2:  makeValue("ISO", nil),
	0x4:  makeValue("Quality", nil),
	0x5:  makeValue("WhiteBalance", nil),
	0x7:  makeValue("FocusMode", nil),
	0x8:  makeValue("FlashSetting", nil),
	0x9:  makeValue("FlashType", nil),
	0xb:  makeValue("WhiteBalanceFineTune", nil),
	0xc:  makeValue("WB_RBLevels", nil),
	0xd:  makeValue("ProgramShift", nil),
	0xe:  makeValue("ExposureDifference", nil),
	0x11: makeValue("PreviewIFD", nil),
	0x13: makeValue("ISOSetting", nil),
	0x17: makeValue("ExternalFlashExposureComp", nil),
	0x18: makeValue("FlashExposureBracketValue", nil),
	0x19: makeValue("ExposureBracketValue", nil),
	0x1b: makeValue("CropHiSpeed", nil),
	0x1c: makeValue("ExposureTuning", nil),
	0x1d: makeValue("SerialNumber", nil),
	0x1e: makeValue("ColorSpace", nil),
	0x1f: makeValue("VRInfo", nil),
	0x22: makeValue("ActiveD-Lighting", nil),
	0x23: makeValue("PictureControlData", nil),
	0x24: makeValue("WorldTime", nil),
	0x25: makeValue("ISOInfo", nil),
	0x2a: makeValue("VignetteControl", nil),
	0x2b: makeValue("DistortInfo", nil),
	0x2c: makeValue("UnknownInfo", nil),
	0x31: makeValue("<unknown>", nil),
	0x32: makeValue("UnknownInfo2", nil),
	0x83: makeValue("LensType", lensType),
	0x84: makeValue("Lens", nil),
	0x87: makeValue("FlashMode", flashMode),
	0x89: makeValue("ShootingMode", nil),
	0x8a: makeValue("AutoBracketRelease", nil),
	0x8b: makeValue("LensFStops", nil),
	0x8c: makeValue("ContrastCurve", nil),
	0x91: makeValue("ShotInfo", nil),
	0x93: makeValue("NEFCompression", nikonCompression),
	0x95: makeValue("NoiseReduction", nil),
	0x96: makeValue("NEFLinearizationTable", nil),
	0x97: makeValue("ColorBalance", nil),
	0x98: makeValue("LensData", nil),
	0x99: makeValue("RawImageCenter", nil),
	0x9e: makeValue("RetouchHistory", nil),
	0xa3: makeValue("<unknown>", nil),
	0xa4: makeValue("<unknown>", nil),
	0xa7: makeValue("ShutterCount", nil),
	0xa8: makeValue("FlashInfo", nil),
	0xb0: makeValue("MultiExposure", nil),
	0xb1: makeValue("HighISONoiseReduction", nil),
	0xb6: makeValue("PowerUpTime", nil),
	0xb7: makeValue("AFInfo2", nil),
	0xb8: makeValue("FileInfo", nil),
	0xb9: makeValue("AFTune", nil),
	0xba: makeValue("<unknown>", nil),
	0xbb: makeValue("RetouchInfo", nil),
	0xbc: makeValue("<unknown>", nil),
	0xbf: makeValue("<unknown>", nil),
}

func makerNoteVersion(t nef.Tag) interface{} {
	return fmt.Sprintf("0x%08x", t.Raw)
}

func lensType(t nef.Tag) interface{} {
	switch bits.OnesCount32(t.Uint()) {
	case 0:
		return "MF"
	case 1:
		return "D"
	case 2:
		return "G"
	case 3:
		return "VR"
	case 4:
		return "1"
	case 5:
		return "FT-1"
	case 6:
		return "E"
	case 7:
		return "AF-P"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
	return ""
}

func nikonCompression(t nef.Tag) interface{} {
	switch t.Uint() {
	case 1:
		return "lossy (type 1)"
	case 2:
		return "uncompressed"
	case 3:
		return "lossless"
	case 4:
		return "lossy (type 2)"
	case 5:
		return "striped packed 12 bits"
	case 6:
		return "uncompressed (reduced to 12 bit)"
	case 7:
		return "unpacked 12 bits"
	case 8:
		return "small"
	case 9:
		return "packed 12 bits"
	case 10:
		return "packed 14 bits"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}

func flashMode(t nef.Tag) interface{} {
	switch t.Uint() {
	case 0:
		return "Did Not Fire"
	case 1:
		return "Fired, Manual"
	case 3:
		return "Not Ready"
	case 7:
		return "Fired, External"
	case 8:
		return "Fired, Commander Mode"
	case 9:
		return "Fired, TTL Mode"
	default:
		return fmt.Sprintf("other (%d)", t.Uint())
	}
}
