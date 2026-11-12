package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type OriginalManifest struct {
	Version                                   int
	BoxID, Created                            string
	Host                                      json.RawMessage
	Connection, Phone, Designers, ClientNotes json.RawMessage
	Workbook                                  bool
}

func originalBackupState(raw []byte, key RecoveryKey, entries map[string][]byte) (PortableManifest, map[string][]byte, error) {
	var old OriginalManifest
	var out PortableManifest
	if e := json.Unmarshal(raw, &old); e != nil {
		return out, nil, e
	}
	if (old.Version != 1 && old.Version != 2) || old.BoxID != key.BoxID {
		return out, nil, errors.New("원본 백업 버전·복구키가 일치하지 않습니다")
	}
	dir, e := os.MkdirTemp("", "alpha-legacy-verify-*")
	if e != nil {
		return out, nil, e
	}
	defer os.RemoveAll(dir)
	for name, b := range map[string]json.RawMessage{"projects-v2.dat": old.Host, "connection-v1.dat": old.Connection, "phone-push-v1.dat": old.Phone, "designers-v1.dat": old.Designers, "client-notes-v1.dat": old.ClientNotes} {
		if len(b) == 0 || string(b) == "null" {
			continue
		}
		if !json.Valid(b) {
			return out, nil, errors.New("원본 백업 JSON 오류")
		}
		encrypted, e := protect(b)
		if e != nil {
			return out, nil, e
		}
		if e = atomicFile(filepath.Join(dir, name), encrypted); e != nil {
			return out, nil, e
		}
	}
	state, e := importLegacy(dir)
	if e != nil {
		return out, nil, e
	}
	state.LegacySourceHash = ""
	state.LegacyHashes = nil
	state.Integration.Backup.Key = key
	normalized := map[string][]byte{}
	for name, b := range entries {
		n := name
		if strings.HasPrefix(n, "photos/") {
			n = strings.TrimSuffix(n, ".jpg")
		}
		if _, ok := normalized[n]; ok {
			return out, nil, errors.New("원본 백업 사진 경로 중복")
		}
		normalized[n] = b
	}
	check := map[string]string{}
	for _, p := range state.Projects {
		for _, ph := range p.Photos {
			if e := validateArchivePhoto(ph, normalized); e != nil {
				return out, nil, e
			}
		}
	}
	for _, c := range state.ClientNotes {
		for _, ph := range c.Photos {
			if e := validateArchivePhoto(ph, normalized); e != nil {
				return out, nil, e
			}
		}
	}
	for name, b := range normalized {
		if name != "manifest.json" {
			check[name] = digest(b)
		}
	}
	out = PortableManifest{Version: 1, Format: "wanryotalk-reconstructed-v2", BoxID: key.BoxID, Created: old.Created, State: state, Checksums: check}
	return out, normalized, nil
}
func validateArchivePhoto(ph Photo, entries map[string][]byte) error {
	data, ok := entries["photos/"+ph.ID]
	if !ok || len(data) > 8<<20 || len(data) == 0 || http.DetectContentType(data) != "image/jpeg" || (ph.Size > 0 && ph.Size != len(data)) {
		return errors.New("원본 사진 누락·형식·크기를 확인하세요")
	}
	return nil
}
