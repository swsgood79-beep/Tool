package main

import "strings"

type ERPDBCustomer struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

// Some ERP installations share customer masters between units. Prefer a unique
// name for the exact unit, then a unique name for the same code across units.
// Conflicting names remain unresolved. Never substitute a different code.
func resolveOutstandingCustomers(rows []OrderOutstandingRow, customers []ERPDBCustomer) []OrderOutstandingRow {
	byCode := map[string][]ERPDBCustomer{}
	for _, customer := range customers {
		code := strings.TrimSpace(customer.Code)
		customer.Name = strings.TrimSpace(customer.Name)
		customer.Unit = strings.TrimSpace(customer.Unit)
		if code != "" && customer.Name != "" {
			byCode[code] = append(byCode[code], customer)
		}
	}
	result := make([]OrderOutstandingRow, len(rows))
	for i, row := range rows {
		code := strings.TrimSpace(row.CustomerCode)
		if code != "" {
			names := map[string]bool{}
			unitNames := map[string]bool{}
			for _, customer := range byCode[code] {
				names[customer.Name] = true
				if strings.TrimSpace(row.Unit) != "" && customer.Unit == strings.TrimSpace(row.Unit) {
					unitNames[customer.Name] = true
				}
			}
			if len(unitNames) > 0 {
				names = unitNames
			}
			row.Customer = ""
			if len(names) == 1 {
				for name := range names {
					row.Customer = name
				}
			}
		}
		if row.Unit != "" && row.PurchaseDate != "" && row.Purchase != "" && row.Sequence > 0 && row.ItemCode != "" {
			row.ReceiptID = digest(jsonRaw(receiptRowKey(row)))
		}
		// Keep internal matching keys private; the owner preview resolves opaque IDs against a fresh query.
		row.CustomerCode = ""
		row.Unit = ""
		result[i] = row
	}
	return result
}
