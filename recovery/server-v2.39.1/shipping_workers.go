package main

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Internal correction history; the visible inspection/archive dates are intact.
type ShippingWorkerEdit struct {
	BeforeSetting, BeforeShipping, Setting, Shipping string
	ActorAccountID, ActorUsername, ActorName         string
	When                                             time.Time
}

func shippingWorkersRevision(r *ShippingRecord) string {
	return digest(jsonRaw([]any{r.ID, r.WorkerRevision, r.Project.Setting.Worker, r.Project.Shipping.Worker, r.ArchivedAt}))
}
func (a *App) saveShippingWorkers(id, revision, setting, shipping string, actor InspectionInspector) error {
	setting, shipping = strings.TrimSpace(setting), strings.TrimSpace(shipping)
	for _, name := range []string{setting, shipping} {
		if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 40 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return errors.New("작업자 이름은 줄바꿈 없이 각각 40자 이내로 입력해 주세요.")
		}
	}
	return a.commitAnnouncement(func(s *State) error {
		r := shippingRecord(s, id)
		if r == nil || r.ArchivedAt == nil {
			return errors.New("보관 완료된 출하 기록에서만 이름을 수정할 수 있습니다.")
		}
		if !equal(revision, shippingWorkersRevision(r)) {
			return errors.New("다른 화면에서 작업자 정보가 변경되었습니다. 현재 이름을 확인하고 다시 저장해 주세요.")
		}
		if r.Project.Setting.Worker == setting && r.Project.Shipping.Worker == shipping {
			return nil
		}
		r.WorkerEdits = append(r.WorkerEdits, ShippingWorkerEdit{BeforeSetting: r.Project.Setting.Worker, BeforeShipping: r.Project.Shipping.Worker, Setting: setting, Shipping: shipping, ActorAccountID: actor.AccountID, ActorUsername: actor.Username, ActorName: actor.Name, When: time.Now()})
		r.Project.Setting.Worker, r.Project.Shipping.Worker = setting, shipping
		r.WorkerRevision++
		return nil
	})
}
