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
		row.Purchase = strings.TrimSpace(row.Purchase)
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
		// Internal matching keys are not needed by the popup response.
		row.CustomerCode = ""
		row.Unit = ""
		result[i] = row
	}
	return result
}
