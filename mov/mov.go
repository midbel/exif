package mov

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

var ErrNotFound = errors.New("not found")

var (
	unix  = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	epoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	delta = epoch.Sub(unix)
)

const (
	mdat  = "mdat"
	udat  = "udat"
	mvhd  = "mvhd"
	moov  = "moov"
	ftyp  = "ftyp"
	quick = "qt"
)

type Profile struct {
	Version uint8

	Spare1 [3]byte

	Created           uint32
	Modified          uint32
	TimeScale         uint32
	Duration          uint32
	Rate              uint32
	Volume            uint16
	Spare2            [10]byte
	Matrix            [36]byte
	PreviewTime       uint32
	PreviewDuration   uint32
	PosterTime        uint32
	SelectionTime     uint32
	SelectionDuration uint32
	CurrentTime       uint32
	Next              uint32
}

func (p Profile) Length() time.Duration {
	length := p.Duration / p.TimeScale
	return time.Duration(length) * time.Second
}

func (p Profile) AcqTime() time.Time {
	return time.Unix(int64(p.Created), 0).Add(delta)
}

func (p Profile) ModTime() time.Time {
	return time.Unix(int64(p.Modified), 0).Add(delta)
}

type File struct {
	io.Closer
	atoms map[string]*io.SectionReader
}

func Decode(file string) (*File, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	if err := readMagic(r); err != nil {
		return nil, err
	}
	return readAtoms(r)
}

func (f File) DecodeProfile() (Profile, error) {
	var p Profile
	r, ok := f.atoms[moov]
	if !ok {
		return p, fmt.Errorf("%w: atoms %s", ErrNotFound, moov)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return p, err
	}
	rs, err := findAtom(mvhd, r)
	if err != nil {
		return p, nil
	}
	return p, binary.Read(rs, binary.BigEndian, &p)
}

func findAtom(atom string, r io.ReadSeeker) (io.Reader, error) {
	var (
		buf = make([]byte, 8)
		rs io.Reader
	)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		size := binary.BigEndian.Uint32(buf) - 8
		if string(buf[4:]) == atom {
			buf = make([]byte, int(size))
			if _, err := io.ReadFull(r, buf); err != nil {
				return nil, err
			}
			rs = bytes.NewReader(buf)
			break
		}
		if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	if rs == nil {
		return nil, fmt.Errorf("%w: atom %s", ErrNotFound, atom)
	}
	return rs, nil
}

func readAtoms(r *os.File) (*File, error) {
	var (
		buf   = make([]byte, 8)
		atoms = make(map[string]*io.SectionReader)
	)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		size := binary.BigEndian.Uint32(buf)
		tell, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		atoms[string(buf[4:])] = io.NewSectionReader(r, tell, int64(size)-8)
		if _, err := r.Seek(int64(size)-8, io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	f := File{
		Closer: r,
		atoms:  atoms,
	}
	return &f, nil
}

func readMagic(r io.ReadSeeker) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if string(buf[4:]) != ftyp {
		return fmt.Errorf("expected %s, got %s", ftyp, buf[4:])
	}
	size := binary.BigEndian.Uint32(buf[:4])
	_, err := r.Seek(int64(size)-8, io.SeekCurrent)
	return err
}
