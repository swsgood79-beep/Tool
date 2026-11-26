package main

import (
	"errors"
	"strings"
	"time"
)

type ShippingDateEdit struct {
	Before, Date, ActorAccountID, ActorUsername, ActorName string
	When                                                   time.Time
}

func shippingDateRevision(r *ShippingRecord) string {
	return digest(jsonRaw([]any{r.ID, r.DateRevision, r.Project.ShippingDueDate, r.Project.ShippingDueDates}))
}
func (a *App) saveShippingDate(id, revision, date string, actor InspectionInspector) error {
	date = strings.TrimSpace(date)
	d, e := time.Parse("2006-01-02", date)
	if e != nil || len(date) != 10 || d.Format("2006-01-02") != date {
		return errors.New("출고일자를 YYYY-MM-DD 형식의 올바른 날짜로 입력해 주세요.")
	}
	return a.commitAnnouncement(func(s *State) error {
		r := shippingRecord(s, id)
		if r == nil {
			return errors.New("출하 기록을 찾을 수 없습니다.")
		}
		if !equal(revision, shippingDateRevision(r)) {
			return errors.New("출고일자가 다른 화면에서 변경되었습니다. 현재 날짜를 확인하고 다시 저장해 주세요.")
		}
		if r.Project.ShippingDueDate == date && !r.Project.SplitShipment() {
			return nil
		}
		r.DateEdits = append(r.DateEdits, ShippingDateEdit{Before: r.Project.ShippingDueLabel(), Date: date, ActorAccountID: actor.AccountID, ActorUsername: actor.Username, ActorName: actor.Name, When: time.Now()})
		r.DateRevision++
		r.Project.ShippingDueDate = date
		r.Project.ShippingDueDates = nil
		return nil
	})
}
