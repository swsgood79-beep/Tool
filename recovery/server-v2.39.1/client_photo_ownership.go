package main

// Remove references proven to be old copies of this linked company source.
// Keep distinct order-only photos and all on-disk files intact.
func (a *App) clientOrderOnlyPhotos(s *State, p Project, c ClientNote) []Photo {
	sourceIDs, aliases, hashes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	prefix := fmtID(p.ID) + ":"
	for _, ph := range c.Photos {
		sourceIDs[ph.ID] = true
		if id := s.MaterializedPhotos[prefix+ph.ID]; id != "" {
			aliases[id] = true
		}
		if b, e := shippingExportPhoto(a.dir, ph); e == nil {
			if data, _, _, e := canonicalJPEG(b); e == nil {
				hashes[digest(data)] = true
			}
		}
	}
	own := []Photo{}
	for _, ph := range p.Photos {
		if sourceIDs[ph.ID] || aliases[ph.ID] {
			continue
		}
		duplicate := false
		if b, e := shippingExportPhoto(a.dir, ph); e == nil {
			if data, _, _, e := canonicalJPEG(b); e == nil {
				duplicate = hashes[digest(data)]
			}
		}
		if !duplicate {
			own = append(own, ph)
		}
	}
	return own
}
