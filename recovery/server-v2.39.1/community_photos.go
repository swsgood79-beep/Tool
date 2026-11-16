package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Downsize and strip image metadata before storing worker uploads. Existing
// project and notice upload behavior is intentionally unchanged.
func (a *App) uploadCommunityPhotos(r *http.Request) ([]Photo, error) {
	out := []Photo{}
	complete := false
	defer func() {
		if !complete {
			for _, p := range out {
				_ = os.Remove(filepath.Join(a.dir, "photos", p.ID))
			}
		}
	}()
	if r.MultipartForm == nil {
		return out, nil
	}
	files := r.MultipartForm.File["photos"]
	if len(files) > 5 {
		return nil, errors.New("사진은 최대 5장입니다.")
	}
	for _, file := range files {
		if file.Size > 8<<20 {
			return nil, errors.New("사진은 한 장당 8MB 이하입니다.")
		}
		f, err := file.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(f, (8<<20)+1))
		f.Close()
		if err != nil || len(b) > 8<<20 {
			return nil, errors.New("사진을 읽을 수 없습니다.")
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
			return nil, errors.New("JPG·PNG·GIF 사진을 사용해 주세요. 너무 큰 사진은 등록할 수 없습니다.")
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		orientation := communityOrientation(b)
		orientedW, orientedH := cfg.Width, cfg.Height
		if orientation >= 5 {
			orientedW, orientedH = orientedH, orientedW
		}
		w, h := orientedW, orientedH
		if w > 1600 || h > 1600 {
			if w >= h {
				h = h * 1600 / w
				w = 1600
			} else {
				w = w * 1600 / h
				h = 1600
			}
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		bounds := im.Bounds()
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				sx, sy := communitySourcePoint(x*orientedW/w, y*orientedH/h, cfg.Width, cfg.Height, orientation)
				c := color.NRGBAModel.Convert(im.At(bounds.Min.X+sx, bounds.Min.Y+sy)).(color.NRGBA)
				a := uint32(c.A)
				dst.SetRGBA(x, y, color.RGBA{uint8((uint32(c.R)*a + 255*(255-a)) / 255), uint8((uint32(c.G)*a + 255*(255-a)) / 255), uint8((uint32(c.B)*a + 255*(255-a)) / 255), 255})
			}
		}
		var encoded bytes.Buffer
		if err = jpeg.Encode(&encoded, dst, &jpeg.Options{Quality: 82}); err != nil {
			return nil, err
		}
		protected, err := protect(encoded.Bytes())
		if err != nil {
			return nil, err
		}
		id := token()
		if err = os.MkdirAll(filepath.Join(a.dir, "photos"), 0700); err != nil {
			return nil, err
		}
		if err = atomicFile(filepath.Join(a.dir, "photos", id), protected); err != nil {
			return nil, err
		}
		out = append(out, Photo{ID: id, Name: filepath.Base(file.Filename), MIME: "image/jpeg", Width: w, Height: h, Size: encoded.Len()})
	}
	complete = true
	return out, nil
}

// Read only EXIF orientation so phone photos stay upright after metadata removal.
func communityOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xff || b[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos+4 <= len(b); {
		if b[pos] != 0xff {
			return 1
		}
		marker := b[pos+1]
		pos += 2
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		if marker == 0xff || marker == 0x00 {
			continue
		}
		length := int(binary.BigEndian.Uint16(b[pos : pos+2]))
		if length < 2 || pos+length > len(b) {
			return 1
		}
		part := b[pos+2 : pos+length]
		pos += length
		if marker != 0xe1 || len(part) < 14 || string(part[:6]) != "Exif\x00\x00" {
			continue
		}
		t := part[6:]
		var order binary.ByteOrder
		if string(t[:2]) == "II" {
			order = binary.LittleEndian
		} else if string(t[:2]) == "MM" {
			order = binary.BigEndian
		} else {
			return 1
		}
		if order.Uint16(t[2:4]) != 42 {
			return 1
		}
		offset := uint64(order.Uint32(t[4:8]))
		if offset+2 > uint64(len(t)) {
			return 1
		}
		count := uint64(order.Uint16(t[offset : offset+2]))
		for i := uint64(0); i < count; i++ {
			entry := offset + 2 + i*12
			if entry+12 > uint64(len(t)) {
				return 1
			}
			p := t[entry : entry+12]
			if order.Uint16(p[:2]) == 0x112 && order.Uint16(p[2:4]) == 3 && order.Uint32(p[4:8]) == 1 {
				v := int(order.Uint16(p[8:10]))
				if v >= 1 && v <= 8 {
					return v
				}
				return 1
			}
		}
	}
	return 1
}
func communitySourcePoint(x, y, w, h, orientation int) (int, int) {
	switch orientation {
	case 2:
		return w - 1 - x, y
	case 3:
		return w - 1 - x, h - 1 - y
	case 4:
		return x, h - 1 - y
	case 5:
		return y, x
	case 6:
		return y, h - 1 - x
	case 7:
		return w - 1 - y, h - 1 - x
	case 8:
		return w - 1 - y, x
	}
	return x, y
}
