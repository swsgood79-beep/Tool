package main

import (
	"testing"
	"time"
)

func TestMachiningThreeStagesPrerequisitesAndLegacy(t *testing.T) {
	a := fixture(t)
	_, s := machiningWorker(t, a, "three-stage")
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
	if act("milling1", "start", 1) == nil || act("milling2", "start", 1) == nil || act("milling", "start", 1) == nil {
		t.Fatal("prerequisite bypass")
	}
	if e := act("cutting", "start", 10); e != nil {
		t.Fatal(e)
	}
	if e := act("cutting", "complete", 0); e != nil {
		t.Fatal(e)
	}
	if act("milling2", "start", 1) == nil {
		t.Fatal("second before first")
	}
	if e := act("milling1", "start", 6); e != nil {
		t.Fatal(e)
	}
	if act("milling2", "start", 1) == nil {
		t.Fatal("unfinished first used as credit")
	}
	if e := act("milling1", "complete", 0); e != nil {
		t.Fatal(e)
	}
	if act("milling2", "start", 7) == nil {
		t.Fatal("over allocated second")
	}
	if e := act("milling2", "start", 6); e != nil {
		t.Fatal(e)
	}
	if e := act("milling2", "complete", 0); e != nil {
		t.Fatal(e)
	}
	if machiningUnreceived(&a.state) != 6 {
		t.Fatal("receipt count includes wrong stage")
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&restored.state, id)
	if len(j.MillingFirst) != 1 || len(j.Milling) != 1 {
		t.Fatal("stage history lost")
	}
	// A pre-upgrade completed final operation stays completed; no first timestamps
	// or actors are fabricated, and its original receipt and run ID are unchanged.
	now := time.Now()
	old := MachiningJob{Required: 10, Cutting: []MachiningRun{{ID: "cut", Quantity: 10, Completed: &now}}, Milling: []MachiningRun{{ID: "old-final", Quantity: 10, Started: now, Completed: &now, ReceiptAt: &now}}}
	machiningUpgrade(&old)
	if machiningFirstDone(&old) != 10 || machiningDone(old.Milling) != 10 || len(old.MillingFirst) != 0 || old.Milling[0].ID != "old-final" || old.Milling[0].ReceiptAt != &now {
		t.Fatal("legacy history rewritten")
	}
	if !machiningLegacy(&old, "old-final") || machiningLegacy(&old, "new-final") {
		t.Fatal("legacy credit escaped original IDs")
	}
}

func TestMachiningLegacyActiveCanFinishWithoutInventedStage(t *testing.T) {
	a := fixture(t)
	_, s := machiningWorker(t, a, "legacy-worker")
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	before := time.Now().Add(-24 * time.Hour)
	now := time.Now()
	actor := machiningActor(&a.state, s)
	if e := a.commitAnnouncement(func(st *State) error {
		j := machiningFind(st, id)
		j.Workflow = 0
		j.Cutting = []MachiningRun{{ID: "old-cut", Quantity: 10, Started: before, StartedBy: actor, Completed: &now}}
		j.Milling = []MachiningRun{{ID: "old-active", Quantity: 10, Started: before, StartedBy: actor}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	if e := a.actMachining(s, id, "milling", "complete", "old-active", j.Revision, 0); e != nil {
		t.Fatal(e)
	}
	j = machiningFind(&a.state, id)
	if len(j.MillingFirst) != 0 || j.Milling[0].ID != "old-active" || !j.Milling[0].Started.Equal(before) || j.Milling[0].Completed == nil || !machiningLegacy(j, "old-active") {
		t.Fatal("legacy active operation was restarted or rewritten")
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	j = machiningFind(&restored.state, id)
	if j.Workflow != 2 || machiningFirstDone(j) != 10 || len(j.LegacyMilling) != 1 || len(j.MillingFirst) != 0 {
		t.Fatal("legacy migration did not persist")
	}
}
