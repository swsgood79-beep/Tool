package main

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

func TestMachiningQuantityOnCompletionBatches(t *testing.T) {
	a := fixture(t)
	_, s := machiningWorker(t, a, "actual-qty")
	_, other := machiningWorker(t, a, "other-qty")
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	act := func(phase, action string, q int) error {
		j := machiningFind(&a.state, id)
		rs := j.Cutting
		if phase == "milling1" {
			rs = j.MillingFirst
		} else if phase == "milling2" {
			rs = j.Milling
		}
		run := ""
		if r := machiningActive(rs); r != nil {
			run = r.ID
		}
		return a.actMachining(s, id, phase, action, run, j.Revision, q)
	}
	if e := act("cutting", "start", 0); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	r := j.Cutting[0]
	if r.Quantity != 0 || !r.QuantityOnComplete || r.Completed != nil {
		t.Fatal("quantity recorded before completion")
	}
	yesterday := time.Now().Add(-24 * time.Hour)
	a.state.MachiningJobs[0].Cutting[0].Started = yesterday
	if act("cutting", "complete", 0) == nil || act("cutting", "complete", 33) == nil || act("cutting", "complete", -1) == nil {
		t.Fatal("invalid completion accepted")
	}
	j = machiningFind(&a.state, id)
	if e := a.actMachining(other, id, "cutting", "complete", r.ID, j.Revision, 12); e == nil {
		t.Fatal("other actor completed work")
	}
	// Persisted active run can resume the next day with no prefilled quantity.
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if !machiningFind(&restored.state, id).Cutting[0].QuantityOnComplete {
		t.Fatal("quantity mode lost on restart")
	}
	before := time.Now()
	if e := act("cutting", "complete", 12); e != nil {
		t.Fatal(e)
	}
	j = machiningFind(&a.state, id)
	if machiningDone(j.Cutting) != 12 || j.Cutting[0].Completed.Before(before) || !j.Cutting[0].Started.Equal(yesterday) {
		t.Fatal("actual quantity or time incorrect")
	}
	if e := act("milling1", "start", 0); e != nil {
		t.Fatal(e)
	}
	if act("milling1", "complete", 13) == nil {
		t.Fatal("first stage exceeded completed cutting")
	}
	if e := act("milling1", "complete", 7); e != nil {
		t.Fatal(e)
	}
	if e := act("milling2", "start", 0); e != nil {
		t.Fatal(e)
	}
	if act("milling2", "complete", 8) == nil {
		t.Fatal("second exceeded completed first")
	}
	if e := act("milling2", "complete", 7); e != nil {
		t.Fatal(e)
	}
	if machiningUnreceived(&a.state) != 7 {
		t.Fatal("receipt queue uses planned count")
	}
	if e := act("cutting", "start", 0); e != nil {
		t.Fatal(e)
	}
	if e := act("cutting", "complete", 20); e != nil {
		t.Fatal(e)
	}
	if act("cutting", "start", 0) == nil {
		t.Fatal("started fully completed cutting")
	}
	j = machiningFind(&a.state, id)
	if machiningDone(j.Cutting) != 32 || len(j.Cutting) != 2 || j.Cutting[0].Quantity != 12 || j.Cutting[1].Quantity != 20 {
		t.Fatal("partial batches overwritten")
	}
}

func TestMachiningPreUpdateRunPartialCompletion(t *testing.T) {
	a := fixture(t)
	_, s := machiningWorker(t, a, "previous-run")
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	if e := a.actMachining(s, id, "cutting", "start", "", 1, 10); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	old := j.Cutting[0]
	if e := a.actMachining(s, id, "cutting", "complete", old.ID, j.Revision, 4); e != nil {
		t.Fatal(e)
	}
	j = machiningFind(&a.state, id)
	if j.Cutting[0].ID != old.ID || !j.Cutting[0].Started.Equal(old.Started) || j.Cutting[0].Quantity != 4 || machiningDone(j.Cutting) != 4 {
		t.Fatal("legacy start rewritten")
	}
	if e := a.actMachining(s, id, "cutting", "start", "", j.Revision, 0); e != nil {
		t.Fatal(e)
	}
}

func TestMachiningOldCachedClientCannotGuessCompletion(t *testing.T) {
	a := fixture(t)
	cookie, s := machiningWorker(t, a, "cached-completion")
	a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{})
	id := a.state.MachiningJobs[0].ID
	if e := a.actMachining(s, id, "cutting", "start", "", 1, 0); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	f := url.Values{"csrf": {s.CSRF}, "id": {id}, "phase": {"cutting"}, "action": {"complete"}, "run": {j.Cutting[0].ID}, "revision": {fmt.Sprint(j.Revision)}, "quantity": {"1"}}
	if w := call(a, "POST", "/machining/action", f, cookie); w.Code != 409 {
		t.Fatal("old hidden form guessed completion", w.Code)
	}
	if machiningDone(machiningFind(&a.state, id).Cutting) != 0 {
		t.Fatal("cached UI changed quantity")
	}
	f.Set("quantity_entry", "actual")
	f.Set("quantity", "12")
	if w := call(a, "POST", "/machining/action", f, cookie); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if machiningDone(machiningFind(&a.state, id).Cutting) != 12 {
		t.Fatal("explicit quantity not recorded")
	}
}
