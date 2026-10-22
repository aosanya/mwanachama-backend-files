package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrTimeAndPlaceRequired = errors.New("this photo must carry the time and place it was taken")
	ErrUnreadablePhoto      = errors.New("the photo could not be read")
)

const (
	typeJPEG = "image/jpeg"
	typePNG  = "image/png"
	typeWebP = "image/webp"
)

func checkPhoto(content []byte, requireTimeAndPlace bool) (string, []byte, error) {
	kind := http.DetectContentType(content)
	if requireTimeAndPlace {
		if kind != typeJPEG {
			return kind, nil, ErrTimeAndPlaceRequired
		}
		taken, placed, err := jpegCapture(content)
		if err != nil {
			return kind, nil, err
		}
		if !taken || !placed {
			return kind, nil, ErrTimeAndPlaceRequired
		}
		return kind, content, nil
	}
	var out []byte
	var err error
	switch kind {
	case typeJPEG:
		out, err = stripJPEG(content)
	case typePNG:
		out, err = stripPNG(content)
	case typeWebP:
		out, err = stripWebP(content)
	default:
		return kind, content, nil
	}
	if err != nil {
		return kind, nil, err
	}
	return kind, out, nil
}

func unreadable(what string) error {
	return fmt.Errorf("%w: %s", ErrUnreadablePhoto, what)
}

type jpegSegment struct {
	marker  byte
	payload []byte
	raw     []byte
}

func jpegSegments(b []byte) ([]jpegSegment, []byte, error) {
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xD8 {
		return nil, nil, unreadable("not a jpeg")
	}
	var segs []jpegSegment
	i := 2
	for {
		if i+4 > len(b) || b[i] != 0xFF {
			return nil, nil, unreadable("broken jpeg segment")
		}
		marker := b[i+1]
		if marker == 0xFF {
			i++
			continue
		}
		if marker == 0xD9 {
			return segs, b[i:], nil
		}
		n := int(binary.BigEndian.Uint16(b[i+2 : i+4]))
		if n < 2 || i+2+n > len(b) {
			return nil, nil, unreadable("jpeg segment overruns the file")
		}
		if marker == 0xDA {
			return segs, b[i:], nil
		}
		segs = append(segs, jpegSegment{marker: marker, payload: b[i+4 : i+2+n], raw: b[i : i+2+n]})
		i += 2 + n
	}
}

func stripJPEG(b []byte) ([]byte, error) {
	segs, rest, err := jpegSegments(b)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.Write(b[:2])
	for _, s := range segs {
		if s.marker == 0xE1 {
			continue
		}
		out.Write(s.raw)
	}
	out.Write(rest)
	return out.Bytes(), nil
}

var exifHeader = []byte("Exif\x00\x00")

const (
	tagExifIFD          = 0x8769
	tagGPSIFD           = 0x8825
	tagDateTimeOriginal = 0x9003
	tagGPSLatitude      = 0x0002
	tagGPSLongitude     = 0x0004
)

func jpegCapture(b []byte) (bool, bool, error) {
	segs, _, err := jpegSegments(b)
	if err != nil {
		return false, false, err
	}
	for _, s := range segs {
		if s.marker == 0xE1 && bytes.HasPrefix(s.payload, exifHeader) {
			return tiffCapture(s.payload[len(exifHeader):])
		}
	}
	return false, false, nil
}

type tiffEntry struct {
	tag   uint16
	kind  uint16
	count uint32
	value uint32
}

func tiffCapture(t []byte) (bool, bool, error) {
	if len(t) < 8 {
		return false, false, unreadable("short exif")
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return false, false, unreadable("exif byte order")
	}
	ifd0, err := readIFD(t, order, order.Uint32(t[4:8]))
	if err != nil {
		return false, false, err
	}
	taken, placed := false, false
	for _, e := range ifd0 {
		switch e.tag {
		case tagExifIFD:
			sub, err := readIFD(t, order, e.value)
			if err != nil {
				return false, false, err
			}
			for _, s := range sub {
				if s.tag == tagDateTimeOriginal && s.count > 1 {
					taken = true
				}
			}
		case tagGPSIFD:
			sub, err := readIFD(t, order, e.value)
			if err != nil {
				return false, false, err
			}
			lat, lon := false, false
			for _, s := range sub {
				if s.tag == tagGPSLatitude && s.count == 3 {
					lat = true
				}
				if s.tag == tagGPSLongitude && s.count == 3 {
					lon = true
				}
			}
			placed = lat && lon
		}
	}
	return taken, placed, nil
}

func readIFD(t []byte, order binary.ByteOrder, at uint32) ([]tiffEntry, error) {
	off := int(at)
	if off < 8 || off+2 > len(t) {
		return nil, unreadable("exif directory out of range")
	}
	n := int(order.Uint16(t[off : off+2]))
	if off+2+12*n > len(t) {
		return nil, unreadable("exif directory overruns")
	}
	out := make([]tiffEntry, 0, n)
	for k := 0; k < n; k++ {
		p := off + 2 + 12*k
		out = append(out, tiffEntry{
			tag:   order.Uint16(t[p : p+2]),
			kind:  order.Uint16(t[p+2 : p+4]),
			count: order.Uint32(t[p+4 : p+8]),
			value: order.Uint32(t[p+8 : p+12]),
		})
	}
	return out, nil
}

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func stripPNG(b []byte) ([]byte, error) {
	if !bytes.HasPrefix(b, pngSignature) {
		return nil, unreadable("not a png")
	}
	var out bytes.Buffer
	out.Write(pngSignature)
	i := len(pngSignature)
	for i < len(b) {
		if i+12 > len(b) {
			return nil, unreadable("broken png chunk")
		}
		n := int(binary.BigEndian.Uint32(b[i : i+4]))
		end := i + 12 + n
		if n < 0 || end > len(b) {
			return nil, unreadable("png chunk overruns the file")
		}
		if string(b[i+4:i+8]) != "eXIf" {
			out.Write(b[i:end])
		}
		i = end
	}
	return out.Bytes(), nil
}

const webpExifFlag = 0x08

func stripWebP(b []byte) ([]byte, error) {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return nil, unreadable("not a webp")
	}
	var body bytes.Buffer
	i := 12
	for i < len(b) {
		if i+8 > len(b) {
			return nil, unreadable("broken webp chunk")
		}
		n := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		end := i + 8 + n + n%2
		if n < 0 || end > len(b) {
			return nil, unreadable("webp chunk overruns the file")
		}
		chunk := b[i:end]
		switch string(b[i : i+4]) {
		case "EXIF":
		case "VP8X":
			kept := append([]byte(nil), chunk...)
			if len(kept) > 8 {
				kept[8] &^= webpExifFlag
			}
			body.Write(kept)
		default:
			body.Write(chunk)
		}
		i = end
	}
	out := make([]byte, 12, 12+body.Len())
	copy(out, "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(4+body.Len()))
	copy(out[8:], "WEBP")
	return append(out, body.Bytes()...), nil
}
