package main

import "errors"

// Match the current inspection cycle, not merely an order number or old archive.
func shippingSourceComplete(s *State, record ShippingRecord) bool {
	p := s.project(record.Project.ID)
	if record.DeliveryID != "" {
		return p != nil && !p.Hidden && p.Setting.Completed != nil && splitRecordMatches(s, record)
	}
	return p != nil && !p.Hidden && p.Setting.Completed != nil && p.Shipping.Completed != nil && p.Shipping.EventID == record.Project.Shipping.EventID
}
func shippingRecordForProject(s *State, p Project) *ShippingRecord {
	for i := range s.ShippingRecords {
		r := &s.ShippingRecords[i]
		if r.Project.ID == p.ID && r.Project.Shipping.EventID == p.Shipping.EventID {
			return r
		}
	}
	return nil
}
func shippingRecordRecoverable(s *State, p Project) bool {
	return !p.Hidden && p.Setting.Completed != nil && p.Shipping.Completed != nil && p.Shipping.EventID > 0 && shippingRecordForProject(s, p) == nil
}

// If the live order still needs shipping photos, deleting archived photos must
// restore that obligation. Only source-less/old-cycle archives are fully removed.
func (a *App) deleteShippingArchive(id string) (bool, error) {
	record := shippingRecord(&a.state, id)
	if record == nil {
		return false, errors.New("보관 자료가 없습니다.")
	}
	reset := shippingSourceComplete(&a.state, *record)
	if reset && record.ArchivedAt == nil {
		return false, errors.New("검사가 완료된 수주입니다. 출하사진 필요 목록에서 사진을 등록해 주세요.")
	}
	removed := append([]Photo{}, record.Photos...)
	if !reset {
		removed = append(removed, record.Project.Photos...)
		removed = append(removed, attachmentParts(*record)...)
	}
	err := a.commitAnnouncement(func(s *State) error {
		for i := range s.ShippingRecords {
			if s.ShippingRecords[i].ID != id {
				continue
			}
			if reset {
				r := &s.ShippingRecords[i]
				r.ID = token()
				r.ArchivedAt = nil
				r.Photos = nil
				r.Memo = ""
				r.PhotoUploadKeys = nil
			} else {
				s.ShippingRecords = append(s.ShippingRecords[:i], s.ShippingRecords[i+1:]...)
			}
			return nil
		}
		return errors.New("보관 자료가 없습니다.")
	})
	if err == nil {
		a.pruneCommunityPhotos(removed)
	}
	return reset, err
}

// Explicit repair for records already removed by older releases. This does not
// automatically turn historical completed orders into new photo obligations.
func (a *App) recoverShippingRecord(projectID int64) error {
	p := a.state.project(projectID)
	if p == nil || p.Hidden || p.Setting.Completed == nil || p.Shipping.Completed == nil || p.Shipping.EventID <= 0 {
		return errors.New("세팅·출하 검사가 완료된 현재 수주만 복구할 수 있습니다.")
	}
	if shippingRecordForProject(&a.state, *p) != nil {
		return nil
	}
	var copied []Photo
	err := a.commitAnnouncement(func(s *State) error {
		var err error
		copied, err = a.captureShippingRecord(s, *s.project(projectID))
		return err
	})
	if err != nil {
		a.pruneCommunityPhotos(copied)
	}
	return err
}
