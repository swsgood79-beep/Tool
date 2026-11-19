package main

import "errors"

// Only archive photos are eligible; snapshot/special-instruction photos are not.
// Call under App.mu, with authenticated admin and CSRF checked by the route.
func (a *App) deleteShippingPhoto(id, photoID, revision string) (string, error) {
	record := shippingRecord(&a.state, id)
	if record == nil || record.ArchivedAt == nil {
		return "", errors.New("보관된 출하사진을 찾을 수 없습니다.")
	}
	if !equal(revision, shippingPhotoRevision(record)) {
		return "", errors.New("사진 목록이 변경되었습니다. 새로고침 후 다시 선택해 주세요.")
	}
	index := -1
	for i, p := range record.Photos {
		if p.ID == photoID {
			index = i
			break
		}
	}
	if index < 0 {
		return "", errors.New("삭제할 출하사진을 찾을 수 없습니다.")
	}
	if len(record.Photos) == 1 {
		project := record.Project
		reset, err := a.deleteShippingArchive(id)
		if err != nil {
			return "", err
		}
		if reset {
			if pending := shippingRecordForProject(&a.state, project); pending != nil {
				return pending.ID, nil
			}
		}
		return "", nil
	}
	removed := record.Photos[index]
	err := a.commitAnnouncement(func(s *State) error {
		current := shippingRecord(s, id)
		if current == nil || current.ArchivedAt == nil || !equal(revision, shippingPhotoRevision(current)) {
			return errors.New("사진 목록이 변경되었습니다. 새로고침해 주세요.")
		}
		current.Photos = append(current.Photos[:index], current.Photos[index+1:]...)
		return nil
	})
	if err != nil {
		return "", err
	}
	a.pruneCommunityPhotos([]Photo{removed})
	return id, nil
}
