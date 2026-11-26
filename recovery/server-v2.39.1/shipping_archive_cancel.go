package main

import "sort"

// A pending photo obligation belongs to the completed inspection cycle that
// created it. A still-present order whose cycle was reopened/cancelled no longer
// needs those photos. Missing source orders are intentionally retained: removing
// an order from the connected sheet is not an inspection cancellation.
func pruneCancelledShippingRecords(s *State) []Photo {
	removed := []Photo{}
	records := make([]ShippingRecord, 0, len(s.ShippingRecords))
	for _, r := range s.ShippingRecords {
		p := s.project(r.Project.ID)
		cancelled := r.ArchivedAt == nil && p != nil && (p.Setting.Completed == nil || p.Shipping.Completed == nil || p.Shipping.EventID != r.Project.Shipping.EventID)
		if r.DeliveryID != "" && p != nil {
			cancelled = r.ArchivedAt == nil && !splitRecordMatches(s, r)
		}
		if cancelled {
			removed = append(removed, r.Project.Photos...)
			removed = append(removed, r.Photos...)
			removed = append(removed, attachmentParts(r)...)
		} else {
			records = append(records, r)
		}
	}
	s.ShippingRecords = records
	return removed
}

// Compare the complete pending set so a filtered/paged list also refreshes when
// cancellation changes its total or the current page becomes empty.
func shippingPendingSignature(s *State) string {
	ids := []string{}
	for _, r := range s.ShippingRecords {
		if r.ArchivedAt == nil {
			ids = append(ids, r.ID)
		}
	}
	sort.Strings(ids)
	return digest(jsonRaw(ids))
}
