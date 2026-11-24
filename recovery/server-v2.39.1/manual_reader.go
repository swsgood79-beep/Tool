package main

import (
	"fmt"
	"io"
)

// A single verified chunk is cached, so Range/Seek downloads stay bounded.
type manualFileReader struct {
	dir      string
	file     ShippingAttachment
	position int64
	index    int
	data     []byte
}

func (r *manualFileReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.position >= r.file.Size {
		return 0, io.EOF
	}
	i := int(r.position / int64(shippingAttachmentChunk))
	if i != r.index {
		b, e := shippingExportPhoto(r.dir, r.file.Parts[i])
		if e != nil {
			return 0, e
		}
		if digest(b) != r.file.Hashes[i] {
			return 0, fmt.Errorf("첨부 체크섬 불일치")
		}
		r.data = b
		r.index = i
	}
	offset := int(r.position % int64(shippingAttachmentChunk))
	n := copy(p, r.data[offset:])
	r.position += int64(n)
	return n, nil
}
func (r *manualFileReader) Seek(offset int64, whence int) (int64, error) {
	var p int64
	switch whence {
	case io.SeekStart:
		p = offset
	case io.SeekCurrent:
		p = r.position + offset
	case io.SeekEnd:
		p = r.file.Size + offset
	default:
		return r.position, fmt.Errorf("잘못된 위치")
	}
	if p < 0 {
		return r.position, fmt.Errorf("잘못된 위치")
	}
	r.position = p
	return p, nil
}
