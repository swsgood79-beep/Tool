package main

// Old Milling records remain the final operation; no first-stage times or
// actors are invented. Only their IDs receive legacy prerequisite credit.
func machiningUpgrade(j *MachiningJob) {
	if j.Workflow >= 2 {
		return
	}
	j.Workflow = 2
	for _, r := range j.Milling {
		j.LegacyMilling = append(j.LegacyMilling, r.ID)
	}
}
func machiningLegacy(j *MachiningJob, id string) bool {
	if j.Workflow < 2 {
		return true
	}
	for _, v := range j.LegacyMilling {
		if v == id {
			return true
		}
	}
	return false
}
func machiningFirstCredit(j *MachiningJob, completedOnly bool) int {
	n := 0
	for _, r := range j.Milling {
		if machiningLegacy(j, r.ID) && (!completedOnly || r.Completed != nil) {
			n += r.Quantity
		}
	}
	return n
}
func machiningFirstDone(j *MachiningJob) int {
	return machiningDone(j.MillingFirst) + machiningFirstCredit(j, true)
}
