package main

import (
	"errors"
	"net/http"
)

// Additional photos never change the inspection snapshot or original archive date.
func (a *App) addShippingPhotos(id string, r *http.Request) error {
	record := shippingRecord(&a.state, id)
	if record == nil || record.ArchivedAt == nil {
		return errors.New("보관 완료된 수주만 사진을 추가할 수 있습니다.")
	}
	key := r.FormValue("upload_key")
	if !validPhotoID.MatchString(key) {
		return errors.New("사진 추가 화면을 다시 열어 주세요.")
	}
	for _, used := range record.PhotoUploadKeys {
		if equal(used, key) {
			return nil
		}
	}
	if !equal(r.FormValue("photo_revision"), shippingPhotoRevision(record)) {
		return errors.New("다른 화면에서 사진이 추가되었습니다. 새로고침 후 다시 선택해 주세요.")
	}
	count := 0
	if r.MultipartForm != nil {
		count = len(r.MultipartForm.File["photos"])
	}
	if count < 1 || len(record.Photos)+count > 20 {
		return errors.New("기존 사진을 포함해 1장 이상, 최대 20장까지 저장할 수 있습니다.")
	}
	photos, err := a.uploadReducedPhotos(r, 20-len(record.Photos))
	if err != nil {
		return err
	}
	err = a.commitAnnouncement(func(s *State) error {
		record := shippingRecord(s, id)
		if record == nil || record.ArchivedAt == nil {
			return errors.New("보관 자료가 변경되었습니다.")
		}
		if !equal(r.FormValue("photo_revision"), shippingPhotoRevision(record)) {
			return errors.New("사진이 변경되었습니다. 새로고침해 주세요.")
		}
		record.Photos = append(record.Photos, photos...)
		record.PhotoUploadKeys = append(record.PhotoUploadKeys, key)
		return nil
	})
	if err != nil {
		a.pruneCommunityPhotos(photos)
	}
	return err
}
