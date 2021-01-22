package nef

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrImage  = errors.New("unknown image")
	ErrExist  = errors.New("not found")
	ErrFormat = errors.New("unknown format")
)

var (
	little  = []byte{0x49, 0x49} // II
	big     = []byte{0x4d, 0x4d} // MM
	magicbe = []byte{0x00, 0x2a}
	magicle = []byte{0x2a, 0x00}
)

const (
	Tiff = 0x0
	Exif = 0x8769
	Nef  = 0x14a
	Note = 0x927c
	Gps  = 0x8825
)

const (
	Xmp     = 0x2bc
	Comment = 0x9286

	ImageWidth        = 0x100
	ImageLength       = 0x101
	JpegFromRawStart  = 0x0201
	JpegFromRawLength = 0x0202
	StripOffsets      = 0x111
	RowsPerStrip      = 0x116
	StripByteCounts   = 0x117
	Photometric       = 0x106
	BitsPerSample     = 0x102
)

const (
	ImgBlack uint32 = iota
	ImgWhite
	ImgRGB
	ImgPalette
	ImgMask
	ImgCMYK
	ImgYCbCr
)

type Format uint16

const (
	Byte      Format = 0x1
	String           = 0x2
	Short            = 0x3
	Long             = 0x4
	Rational         = 0x5
	SByte            = 0x6
	Undef            = 0x7
	SShort           = 0x8
	SLong            = 0x9
	SRational        = 0xa
	Float            = 0xb
	Double           = 0xc
)

var formats = map[Format]string{
	Byte:      "byte",
	String:    "ascii",
	Short:     "short",
	Long:      "long",
	Rational:  "rational",
	SByte:     "sbyte",
	Undef:     "undefined",
	SShort:    "sshort",
	SLong:     "slong",
	SRational: "srational",
	Float:     "float",
	Double:    "double",
}

func (f Format) Size() int {
	switch f {
	case Byte, String, SByte, Undef:
		return 1
	case Short, SShort:
		return 2
	case Long, SLong, Float:
		return 4
	case Rational, SRational, Double:
		return 8
	default:
		return 0
	}
}

func (f Format) String() string {
	str, ok := formats[f]
	if !ok {
		str = "unknown"
	}
	return str
}

type Tag struct {
	Id     uint16
	Type   Format
	Count  uint32
	Offset uint32

	Raw []byte
	// Tags   []Tag
	family int
	order  binary.ByteOrder
}

func (t Tag) Size() int {
	return t.Type.Size() * int(t.Count)
}

func (t Tag) Bytes() []byte {
	return append([]byte{}, t.Raw...)
}

func (t Tag) IsPtr() bool {
	switch t.Id {
	case Tiff, Exif, Nef, Note, Gps:
		return true
	default:
		return false
	}
}

func (t Tag) Uint() uint32 {
	switch t.Type {
	case Byte:
		return uint32(t.Raw[0])
	case Short:
		return uint32(t.order.Uint16(t.Raw))
	case Long:
		return t.order.Uint32(t.Raw)
	default:
		return 0
	}
}

func (t Tag) Int() int32 {
	switch t.Type {
	case SByte:
	case SShort:
	case SLong:
	default:
	}
	return 0
}

func (t Tag) Float() float64 {
	switch t.Type {
	case Float:
	case Double:
	case Rational:
	case SRational:
	default:
	}
	return 0
}

func (t Tag) String() string {
	if t.Id == Xmp || t.Id == Comment {
		b := bytes.TrimSpace(t.Raw)
		return string(b)
	}
	switch t.Type {
	case String:
		b := bytes.TrimRight(t.Raw, "\x00")
		return string(b)
	default:
		return ""
	}
}

func (t Tag) Time() time.Time {
	when, err := time.Parse("2006:01:02 15:04:05", t.String())
	if err == nil {
		when = when.UTC()
	}
	return when
}

func (t Tag) Values() ([]string, error) {
	if t.Id == Xmp || t.Id == Comment {
		str := strings.TrimSpace(string(t.Raw))
		return []string{str}, nil
	}
	var str []string
	switch t.Type {
	default:
		return nil, fmt.Errorf("%04x: %w", t.Type, ErrFormat)
	case String:
		b := bytes.TrimRight(t.Raw, "\x00")
		str = append(str, string(bytes.TrimSpace(b)))
	case Long:
		str = decodeLong(t)
	case SLong:
		str = decodeSignedLong(t)
	case Short:
		str = decodeShort(t)
	case SShort:
		str = decodeSignedShort(t)
	case Byte:
		str = decodeByte(t)
	case SByte:
	case Undef:
		// if !t.IsPtr() {
		// 	str = decodeUndefined(t)
		// }
	case Rational:
		str = decodeRational(t)
	case SRational:
		str = decodeSignedRational(t)
	case Float:
		str = decodeFloat(t)
	case Double:
		str = decodeDouble(t)
	}
	return str, nil
}

func (t Tag) Origin() string {
	switch t.family {
	case Tiff, Nef:
		return "tiff"
	case Exif:
		return "exif"
	case Note:
		return "note"
	case Gps:
		return "gps"
	default:
		return "unknown"
	}
}

type File struct {
	reader *bytes.Reader
	order  binary.ByteOrder

	tiff  []Tag
	exif  []Tag
	notes []Tag
	gps   []Tag

	Index []int
	Files []*File
}

func (f File) Tags() []Tag {
	tags := make([]Tag, 0, len(f.tiff)+len(f.exif)+len(f.notes))
	tags = append(tags, f.tiff...)
	tags = append(tags, f.exif...)
	tags = append(tags, f.notes...)
	tags = append(tags, f.gps...)
	return tags
}

func (f File) TagsFor(family uint16) []Tag {
	var tags []Tag
	switch family {
	case Exif:
		tags = f.exif
	case Note:
		tags = f.notes
	case Tiff, Nef:
		tags = f.tiff
	case Gps:
		tags = f.gps
	default:
		return tags
	}
	return append([]Tag{}, tags...)
}

func (f File) GetTag(id uint16, origin int) (Tag, error) {
	var (
		t    Tag
		tags []Tag
	)
	switch origin {
	default:
		return t, ErrExist
	case Tiff, Nef:
		tags = f.tiff
	case Exif:
		tags = f.exif
	case Note:
		tags = f.notes
	}
	x := sort.Search(len(tags), func(i int) bool { return tags[i].Id >= id })
	if x >= len(tags) || tags[x].Id != id {
		return t, ErrExist
	}
	return tags[x], nil
}

func (f File) IsMainDir() bool {
	return len(f.Index) == 1
}

func (f File) IsSubDir() bool {
	return len(f.Index) > 1
}

func (f File) Filename() string {
	prefix := "M"
	if f.IsSubDir() {
		prefix = "S"
	}
	str := make([]string, len(f.Index))
	for i := range f.Index {
		str[i] = fmt.Sprintf("%02d", f.Index[i])
	}
	return fmt.Sprintf("%s-%s", prefix, strings.Join(str, "-"))
}

func (f File) Directory() string {
	switch len(f.Index) {
	case 1:
		return fmt.Sprintf("M-IFD#%d", f.Index[0])
	case 2:
		return fmt.Sprintf("S-IFD#%d", f.Index[1])
	default:
		return "???"
	}
}

func (f File) Image() (image.Image, error) {
	switch {
	case f.IsJpeg():
		return f.decodeJpeg()
	case f.IsRaw():
		return f.decodeRaw()
	default:
		return nil, ErrImage
	}
}

func (f File) IsSupported() bool {
	typ, err := f.get(Photometric)
	if err != nil {
		if errors.Is(err, ErrExist) {
			return true
		}
		return false
	}
	switch typ.Uint() {
	case ImgBlack, ImgWhite, ImgRGB, ImgPalette, ImgMask, ImgCMYK, ImgYCbCr:
		return true
	default:
		return false
	}
}

func (f File) ImageType() string {
	typ, err := f.get(Photometric)
	if errors.Is(err, ErrExist) {
		return "jpeg"
	}
	switch typ := typ.Uint(); typ {
	case ImgBlack, ImgWhite:
		return "raw/gray"
	case ImgRGB:
		return "raw/rgb"
	case ImgPalette:
		return "raw/palette"
	case ImgMask:
		return "raw/mask"
	case ImgCMYK:
		return "raw/cmyk"
	case ImgYCbCr:
		return "raw/ycbcr"
	default:
		return "unsupported"
	}
}

func (f File) decodeRaw() (image.Image, error) {
	var (
		imgtype, _ = f.get(Photometric)
		width, _   = f.get(ImageWidth)
		height, _  = f.get(ImageLength)
		rect       = image.Rect(0, 0, int(width.Offset), int(height.Offset))
		img        image.Image
	)
	buf, err := f.Bytes()
	if err != nil {
		return nil, err
	}
	switch typ := imgtype.Uint(); typ {
	default:
		return nil, fmt.Errorf("%d: %w", typ, ErrFormat)
	case ImgBlack, ImgWhite:
		img = grayImage(rect, buf, typ == ImgWhite)
	case ImgRGB:
		img = rgbImage(rect, buf)
	case ImgCMYK:
		img = image.NewCMYK(rect)
	}
	return img, nil
}

func (f File) decodeJpeg() (image.Image, error) {
	raw, err := f.Bytes()
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func (f File) Bytes() ([]byte, error) {
	switch {
	case f.IsJpeg():
		return f.processJpeg()
	case f.IsRaw():
		return f.processRaw()
	default:
		return nil, ErrImage
	}
}

func (f File) Has(id uint16) bool {
	x := sort.Search(len(f.tiff), func(i int) bool {
		return f.tiff[i].Id >= id
	})
	return x < len(f.tiff) && f.tiff[x].Id == id
}

func (f File) IsJpeg() bool {
	return f.Has(JpegFromRawStart) && f.Has(JpegFromRawLength)
}

func (f File) IsRaw() bool {
	return f.Has(StripOffsets) && f.Has(RowsPerStrip) && f.Has(StripByteCounts)
}

func (f File) processJpeg() ([]byte, error) {
	var (
		start, _  = f.get(JpegFromRawStart)
		length, _ = f.get(JpegFromRawLength)
		img       = make([]byte, int(length.Offset))
		rs        = io.NewSectionReader(f.reader, int64(start.Offset), int64(length.Offset))
	)
	if _, err := io.ReadFull(rs, img); err != nil {
		return nil, err
	}
	return img, nil
}

func (f File) processRaw() ([]byte, error) {
	var (
		strip, _  = f.get(RowsPerStrip)
		offset, _ = f.get(StripOffsets)
		count, _  = f.get(StripByteCounts)
		length, _ = f.get(ImageLength)
		block     uint32
		img       []byte
	)
	block = length.Offset / strip.Offset
	if mod := length.Offset % strip.Offset; mod != 0 {
		block++
	}
	rsp := bytes.NewReader(offset.Raw)
	rsc := bytes.NewReader(count.Raw)
	for i := 0; i < int(block); i++ {
		var (
			pos uint32
			num uint32
		)
		if err := binary.Read(rsp, offset.order, &pos); err != nil {
			return nil, err
		}
		if err := binary.Read(rsc, count.order, &num); err != nil {
			return nil, err
		}
		r := io.NewSectionReader(f.reader, int64(pos), int64(num))
		tmp, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		img = append(img, tmp...)
	}
	return img, nil
}

func (f File) get(id uint16) (Tag, error) {
	x := sort.Search(len(f.tiff), func(i int) bool {
		return f.tiff[i].Id >= id
	})
	var t Tag
	if x >= len(f.tiff) || f.tiff[x].Id != id {
		return t, fmt.Errorf("%04x: %w", id, ErrExist)
	}
	return f.tiff[x], nil
}

func DecodeFile(file string) ([]*File, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return Decode(r)
}

func Decode(r io.Reader) ([]*File, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var (
		rs     = bytes.NewReader(buf)
		offset uint32
	)
	order, err := readOrder(rs)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(rs, order, &offset); err != nil {
		return nil, err
	}
	var files []*File
	for i := 0; offset != 0; i++ {
		f, err := readDirectory(rs, order, offset, i)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
		if err := binary.Read(rs, order, &offset); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return files, nil
}

func readDirectory(r *bytes.Reader, order binary.ByteOrder, at uint32, index int) (*File, error) {
	tags, err := readTags(r, order, at, 0, Tiff)
	if err != nil {
		return nil, err
	}
	offset, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	defer func(offset int64) { r.Seek(offset, io.SeekStart) }(int64(offset))
	f := File{
		reader: r,
		order:  order,
		tiff:   tags,
		Index:  []int{index},
	}
	if f.exif, err = exifTags(r, order, tags); err != nil {
		return nil, err
	}
	if f.gps, err = gpsTags(r, order, tags); err != nil {
		return nil, err
	}
	if f.notes, err = notesTags(r, f.exif); err != nil {
		return nil, err
	}
	data, err := subTags(r, order, tags)
	if err != nil {
		return nil, err
	}
	for i, ts := range data {
		c := File{
			reader: r,
			order:  order,
			tiff:   ts,
			Index:  []int{index, i},
		}
		c.exif = append(c.exif, f.exif...)
		c.notes = append(c.notes, f.notes...)
		f.Files = append(f.Files, &c)
	}
	return &f, nil
}

func exifTags(r *bytes.Reader, order binary.ByteOrder, tags []Tag) ([]Tag, error) {
	x := sort.Search(len(tags), func(i int) bool {
		return tags[i].Id >= Exif
	})
	if x >= len(tags) || tags[x].Id != Exif {
		return nil, nil
	}
	return readTags(r, order, tags[x].Offset, 0, Exif)
}

func gpsTags(r *bytes.Reader, order binary.ByteOrder, tags []Tag) ([]Tag, error) {
	x := sort.Search(len(tags), func(i int) bool {
		return tags[i].Id >= Gps
	})
	if x >= len(tags) || tags[x].Id != Gps {
		return nil, nil
	}
	return readTags(r, order, tags[x].Offset, 0, Gps)
}

const (
	notePreview   uint16 = 0x11
	noteShotInfo         = 0x91
	noteLensData         = 0x98
	noteFlashInfo        = 0xa8
)

func notesTags(r *bytes.Reader, tags []Tag) ([]Tag, error) {
	x := sort.Search(len(tags), func(i int) bool {
		return tags[i].Id >= Note
	})
	if x >= len(tags) || tags[x].Id != Note {
		return nil, nil
	}
	if _, err := r.Seek(int64(tags[x].Offset), io.SeekStart); err != nil {
		return nil, err
	}
	preamble := make([]byte, 10)
	if _, err := io.ReadFull(r, preamble); err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(preamble, []byte("Nikon\x00")) {
		return nil, nil
	}
	order, err := readOrder(r)
	if err != nil {
		return nil, err
	}
	var offset uint32
	if err = binary.Read(r, order, &offset); err != nil {
		return nil, err
	}
	offset += tags[x].Offset + 10
	notes, err := readTags(r, order, offset, tags[x].Offset+10, Note)
	if err != nil {
		return nil, err
	}
	// for _, which := range []uint16{notePreview, noteShotInfo, noteLensData, noteFlashInfo} {
	// 	if others, err := findTags(r, notes, which); err == nil {
	// 		// notes = append(notes, others...)
	// 	}
	// }
	return notes, err
}

func subTags(r *bytes.Reader, order binary.ByteOrder, tags []Tag) ([][]Tag, error) {
	x := sort.Search(len(tags), func(i int) bool {
		return tags[i].Id >= Nef
	})
	if x >= len(tags) || tags[x].Id != Nef {
		return nil, nil
	}
	if _, err := r.Seek(int64(tags[x].Offset), io.SeekStart); err != nil {
		return nil, err
	}
	pos := make([]uint32, int(tags[x].Count))
	for i := 0; i < int(tags[x].Count); i++ {
		binary.Read(r, order, &pos[i])
	}
	data := make([][]Tag, len(pos))
	for i := range pos {
		ts, err := readTags(r, order, pos[i], 0, Tiff)
		if err != nil {
			return nil, err
		}
		data[i] = ts
	}
	return data, nil
}

func findTags(r *bytes.Reader, tags []Tag, which uint16) ([]Tag, error) {
	x := sort.Search(len(tags), func(i int) bool {
		return tags[i].Id >= which
	})
	if x >= len(tags) || tags[x].Id != which {
		return nil, nil
	}
	tell, _ := r.Seek(0, io.SeekCurrent)
	defer func(where int64) {
		r.Seek(where, io.SeekStart)
	}(tell)
	t := tags[x]
	return readTags(r, t.order, t.Offset, 0, t.family)
}

func readTags(r *bytes.Reader, order binary.ByteOrder, at, delta uint32, family int) ([]Tag, error) {
	if _, err := r.Seek(int64(at), io.SeekStart); err != nil {
		return nil, err
	}
	var count uint16
	if err := binary.Read(r, order, &count); err != nil {
		return nil, err
	}
	var tags []Tag
	for i := 0; i < int(count); i++ {
		g := struct {
			Id     uint16
			Type   Format
			Count  uint32
			Offset uint32
		}{}
		if err := binary.Read(r, order, &g); err != nil {
			return nil, err
		}
		if n := len(tags); n > 0 && g.Id <= tags[n-1].Id {
			return nil, fmt.Errorf("tags not sorted properly")
		}
		if g.Count == (1<<32)-1 {
			return nil, fmt.Errorf("invalid count")
		}
		tag := Tag{
			Id:     g.Id,
			Type:   g.Type,
			Count:  g.Count,
			Offset: g.Offset + delta,
			family: family,
			order:  order,
		}
		if z := tag.Size(); z > 4 {
			sr := io.NewSectionReader(r, int64(tag.Offset), int64(z))
			tag.Raw = make([]byte, z)
			if _, err := io.ReadFull(sr, tag.Raw); err != nil {
				return nil, err
			}
		} else {
			tag.Raw = make([]byte, 4)
			order.PutUint32(tag.Raw, tag.Offset)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func readOrder(rs io.ReadSeeker) (binary.ByteOrder, error) {
	var (
		intro = make([]byte, 4)
		magic []byte
		order binary.ByteOrder
	)
	if _, err := io.ReadFull(rs, intro); err != nil {
		return nil, err
	}
	switch {
	case bytes.Equal(intro[:2], little):
		order = binary.LittleEndian
		magic = magicle
	case bytes.Equal(intro[:2], big):
		order = binary.BigEndian
		magic = magicbe
	default:
		return nil, fmt.Errorf("invalid byte order %04x", intro[:2])
	}
	if !bytes.Equal(intro[2:], magic) {
		return nil, fmt.Errorf("invalid magic number %04x", intro[2:])
	}
	return order, nil
}

func decodeShort(tag Tag) []string {
	str := make([]string, int(tag.Count))
	for i := 0; i < len(str); i++ {
		x := tag.order.Uint16(tag.Raw[i*2:])
		str[i] = strconv.FormatUint(uint64(x), 10)
	}
	return str
}

func decodeSignedShort(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var s int16
		binary.Read(rs, tag.order, &s)
		str[i] = strconv.FormatInt(int64(s), 10)
	}
	return str
}

func decodeLong(tag Tag) []string {
	str := make([]string, int(tag.Count))
	for i := 0; i < len(str); i++ {
		x := tag.order.Uint32(tag.Raw[i*4:])
		str[i] = strconv.FormatUint(uint64(x), 10)
	}
	return str
}

func decodeSignedLong(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var s int32
		binary.Read(rs, tag.order, &s)
		str[i] = strconv.FormatInt(int64(s), 10)
	}
	return str
}

func decodeByte(tag Tag) []string {
	str := make([]string, int(tag.Count))
	for i := 0; i < len(str); i++ {
		str[i] = strconv.FormatUint(uint64(tag.Raw[i]), 10)
	}
	return str
}

func decodeRational(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var n, d uint32
		binary.Read(rs, tag.order, &n)
		binary.Read(rs, tag.order, &d)

		str[i] = fmt.Sprintf("%d/%d", n, d)
	}
	return str
}

func decodeFloat(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var f float32
		binary.Read(rs, tag.order, &f)

		str[i] = strconv.FormatFloat(float64(f), 'f', -1, 64)
	}
	return str
}

func decodeDouble(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var f float64
		binary.Read(rs, tag.order, &f)

		str[i] = strconv.FormatFloat(f, 'f', -1, 64)
	}
	return str
}

func decodeSignedRational(tag Tag) []string {
	var (
		str = make([]string, int(tag.Count))
		rs  = bytes.NewReader(tag.Raw)
	)
	for i := 0; i < len(str); i++ {
		var n, d int32
		binary.Read(rs, tag.order, &n)
		binary.Read(rs, tag.order, &d)

		str[i] = fmt.Sprintf("%d/%d", n, d)
	}
	return str
}

func decodeUndefined(t Tag) []string {
	str := hex.EncodeToString(t.Raw)
	return []string{str}
}

func grayImage(rect image.Rectangle, buf []byte, inverted bool) image.Image {
	var (
		img  = image.NewGray(rect)
		rs   = bytes.NewReader(buf)
		gray color.Gray
	)
	for j := 0; j < rect.Dy(); j++ {
		for i := 0; i < rect.Dx(); i++ {
			gray.Y, _ = rs.ReadByte()
			if inverted {
				gray.Y = 255 - gray.Y
			}
			img.Set(i, j, gray)
		}
	}
	return img
}

func rgbImage(rect image.Rectangle, buf []byte) image.Image {
	var (
		img = image.NewRGBA(rect)
		rs  = bytes.NewReader(buf)
		rgb color.RGBA
	)
	for j := 0; j < rect.Dy(); j++ {
		for i := 0; i < rect.Dx(); i++ {
			rgb.R, _ = rs.ReadByte()
			rgb.G, _ = rs.ReadByte()
			rgb.B, _ = rs.ReadByte()
			rgb.A = 255

			img.Set(i, j, rgb)
		}
	}
	return img
}
