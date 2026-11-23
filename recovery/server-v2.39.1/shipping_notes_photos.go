package main

import "strings"

// Keep the saved evidence intact. Legacy client photos can also remain as
// materialized order copies with different IDs; archive views and new exports
// should present each identical image only once, preferring the current source.
func (a *App) shippingCurrentNotes(s *State, record ShippingRecord) ShippingRecord {
	r := shippingCurrentNotes(s, record)
	r.ClientName = shippingCustomerName(s, r)
	source := map[string]bool{}
	for _, id := range r.ClientSourcePhotoIDs {
		source[id] = true
	}
	// Materialized IDs prove that a photo belongs to the linked source rather
	// than being an order-only attachment. Do not hide aliases after unlinking.
	aliases := map[string]bool{}
	if r.ClientSourceID != "" {
		linked := false
		for _, c := range s.ClientNotes {
			if c.ID == r.ClientSourceID {
				linked = true
				break
			}
		}
		if linked {
			prefix := fmtID(r.Project.ID) + ":"
			for key, id := range s.MaterializedPhotos {
				if strings.HasPrefix(key, prefix) {
					aliases[id] = true
				}
			}
		}
	}
	hashes := map[string]string{}
	fingerprint := func(p Photo) string {
		if value, ok := hashes[p.ID]; ok {
			return value
		}
		value := ""
		if raw, e := shippingExportPhoto(a.dir, p); e == nil {
			// Canonical conversion also matches old PNG originals with their JPEG
			// materialized copies. Names, dimensions and file sizes are not identity.
			if data, _, _, e := canonicalJPEG(raw); e == nil {
				value = digest(data)
			}
		}
		hashes[p.ID] = value
		return value
	}
	preferred := map[string]bool{}
	for _, p := range r.Project.Photos {
		if source[p.ID] {
			if h := fingerprint(p); h != "" {
				preferred[h] = true
			}
		}
	}
	seenID, seenHash := map[string]bool{}, map[string]bool{}
	photos := []Photo{}
	for _, p := range r.Project.Photos {
		if seenID[p.ID] || aliases[p.ID] && !source[p.ID] {
			continue
		}
		h := fingerprint(p)
		if h != "" && (seenHash[h] || !source[p.ID] && preferred[h]) {
			continue
		}
		seenID[p.ID] = true
		if h != "" {
			seenHash[h] = true
		}
		photos = append(photos, p)
	}
	r.Project.Photos = photos
	return r
}
