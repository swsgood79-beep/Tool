package main

import "strings"

// The user's customer convention applies to every shipment surface, not only
// Excel: the prefix before the whitespace-delimited axis count is the customer.
// A template link is optional and must never be a condition for displaying it.
func shippingCustomerName(s *State, record ShippingRecord) string {
	if name := shippingExportClientName(record); name != "" {
		return name
	}
	source := record.ClientSourceID
	if source == "" {
		if live := shippingNotesSource(s, record); live != nil {
			source = live.ClientID
		}
	}
	if source != "" {
		for _, c := range s.ClientNotes {
			if c.ID == source {
				return strings.TrimSpace(c.Name)
			}
		}
	}
	return ""
}
