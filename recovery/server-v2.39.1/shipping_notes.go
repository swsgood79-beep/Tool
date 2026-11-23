package main

import "strings"

// Only the same order and completed inspection cycle may recover a legacy
// archive's source link. A re-used order number is never a source binding.
func shippingNotesSource(s *State, r ShippingRecord) *Project {
	p := s.project(r.Project.ID)
	if p == nil || p.Number != r.Project.Number || r.Project.Shipping.EventID == 0 || p.Shipping.EventID != r.Project.Shipping.EventID || p.Shipping.Completed == nil || r.Project.Shipping.Completed == nil || !p.Shipping.Completed.Equal(*r.Project.Shipping.Completed) {
		return nil
	}
	return p
}

// Refresh only special instructions and their photos. Completion, people,
// supplied materials, shipping evidence and archive dates remain snapshots.
func shippingCurrentNotes(s *State, r ShippingRecord) ShippingRecord {
	source := r.ClientSourceID
	own := []Photo{}
	if live := shippingNotesSource(s, r); live != nil {
		source = live.ClientID
		if source == "" && r.ClientSourceID == "" {
			return r
		}
		own = append(own, live.Photos...)
		r.Project.AdditionalInstructions = live.AdditionalInstructions
		if source == "" {
			r.Project.SpecialInstructions = live.SpecialInstructions
			r.Project.Photos = own
			r.ClientSourceID = ""
			r.ClientSourcePhotoIDs = nil
			return r
		}
	} else {
		sourcePhotos := map[string]bool{}
		for _, id := range r.ClientSourcePhotoIDs {
			sourcePhotos[id] = true
		}
		for _, ph := range r.Project.Photos {
			if !sourcePhotos[ph.ID] {
				own = append(own, ph)
			}
		}
	}
	if strings.TrimSpace(source) == "" {
		return r
	}
	for _, c := range s.ClientNotes {
		if c.ID == source {
			if shippingNotesSource(s, r) == nil && r.ClientSourceRevision == c.Revision {
				return r
			}
			r.ClientSourceRevision = c.Revision
			r.ClientSourceID = source
			r.Project.SpecialInstructions = c.Text
			photos := c.Photos
			// Preserve deliberate selection until the company source is edited.
			if live := shippingNotesSource(s, r); live != nil {
				photos = selectedClientPhotos(*live, c)
			}
			r.Project.Photos = append(own, photos...)
			r.ClientSourcePhotoIDs = nil
			for _, ph := range photos {
				r.ClientSourcePhotoIDs = append(r.ClientSourcePhotoIDs, ph.ID)
			}
			return r
		}
	}
	// Deleted/unavailable sources must not erase the retained archive copy.
	return r
}

func syncShippingNotes(s *State) {
	for i := range s.ShippingRecords {
		s.ShippingRecords[i] = shippingCurrentNotes(s, s.ShippingRecords[i])
	}
}

// An explicit order edit may also update order-only notes, including clearing
// them; merely viewing an unlinked archive never substitutes empty live fields.
func syncShippingProjectNotes(s *State, id int64) {
	for i := range s.ShippingRecords {
		r := &s.ShippingRecords[i]
		if r.Project.ID != id {
			continue
		}
		if p := shippingNotesSource(s, *r); p != nil && p.ClientID == "" {
			r.Project.SpecialInstructions = p.SpecialInstructions
			r.Project.AdditionalInstructions = p.AdditionalInstructions
			r.Project.Photos = append([]Photo{}, p.Photos...)
			r.ClientSourceID = ""
			r.ClientSourcePhotoIDs = nil
			r.ClientSourceRevision = 0
		}
	}
	syncShippingNotes(s)
}
