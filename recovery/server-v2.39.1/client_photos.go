package main

// selectedClientPhotos is the live source subset actually attached to this order.
// Keep its limit calculation identical to the worker view and legacy projection.
func selectedClientPhotos(p Project, c ClientNote) []Photo {
	selected := make(map[string]bool, len(p.ClientPhotoIDs))
	for _, id := range p.ClientPhotoIDs {
		selected[id] = true
	}
	seen := make(map[string]bool)
	out := []Photo{}
	for _, ph := range c.Photos {
		if seen[ph.ID] || p.ClientPhotosSelected && !selected[ph.ID] {
			continue
		}
		seen[ph.ID] = true
		out = append(out, ph)
	}
	return out
}
