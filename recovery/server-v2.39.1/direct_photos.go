package main

import (
	"bytes"
	"errors"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
)

func canonicalJPEG(data []byte) ([]byte, int, int, error) {
	cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
	if e != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 32_000_000 {
		return nil, 0, 0, errors.New("원본 호환 사진 형식·해상도 오류")
	}
	if format == "jpeg" && len(data) <= 8<<20 {
		return data, cfg.Width, cfg.Height, nil
	}
	img, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return nil, 0, 0, e
	}
	w, h := cfg.Width, cfg.Height
	if w > 2048 || h > 2048 {
		if w >= h {
			h = h * 2048 / w
			w = 2048
		} else {
			w = w * 2048 / h
			h = 2048
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), draw.Over, nil)
	var b bytes.Buffer
	if e = jpeg.Encode(&b, out, &jpeg.Options{Quality: 88}); e != nil {
		return nil, 0, 0, e
	}
	if b.Len() > 8<<20 {
		return nil, 0, 0, errors.New("JPEG 변환 후 용량 초과")
	}
	return b.Bytes(), w, h, nil
}
func (a *App) normalizePhotos(s *State) error {
	for i := range s.ClientNotes {
		for j := range s.ClientNotes[i].Photos {
			if e := a.normalizePhoto(&s.ClientNotes[i].Photos[j]); e != nil {
				return e
			}
		}
	}
	for i := range s.Projects {
		for j := range s.Projects[i].Photos {
			if e := a.normalizePhoto(&s.Projects[i].Photos[j]); e != nil {
				return e
			}
		}
	}
	return nil
}
func (a *App) normalizePhoto(p *Photo) error {
	if !validPhotoID.MatchString(p.ID) {
		return errors.New("원본 호환 사진 ID 오류")
	}
	var data []byte
	var e error
	if p.Legacy {
		data, e = a.legacyPhoto(*p)
	} else {
		data, e = os.ReadFile(filepath.Join(a.dir, "photos", p.ID))
		if e == nil {
			data, e = unprotect(data)
		}
	}
	if e != nil {
		return e
	}
	data, w, h, e := canonicalJPEG(data)
	if e != nil {
		return e
	}
	if e = a.createCanonicalPhoto(p.ID, data); e != nil {
		return e
	}
	p.Legacy = true
	p.MIME = "image/jpeg"
	p.Size = len(data)
	p.Width = w
	p.Height = h
	return nil
}
func (a *App) createCanonicalPhoto(id string, data []byte) error {
	dir := filepath.Join(a.dir, "project-photos")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	path := filepath.Join(dir, id+".dat")
	if old, e := os.ReadFile(path); e == nil {
		plain, e := unprotect(old)
		if e != nil {
			return e
		}
		if !bytes.Equal(plain, data) {
			return errors.New("기존 사진 식별자의 내용이 달라 덮어쓰지 않았습니다")
		}
		return nil
	} else if !os.IsNotExist(e) {
		return e
	}
	protected, e := protect(data)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	success := false
	defer func() {
		f.Close()
		if !success {
			os.Remove(path)
		}
	}()
	if _, e = f.Write(protected); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	success = true
	return nil
}
func (a *App) materializeSourcePhoto(s *State, project int64, source Photo) (Photo, error) {
	key := fmtID(project) + ":" + source.ID
	id := s.MaterializedPhotos[key]
	if id == "" {
		id = token()
		s.MaterializedPhotos[key] = id
	}
	data, e := a.legacyPhoto(source)
	if e != nil {
		return Photo{}, e
	}
	if e = a.createCanonicalPhoto(id, data); e != nil {
		return Photo{}, e
	}
	p := source
	p.ID = id
	return p, nil
}
