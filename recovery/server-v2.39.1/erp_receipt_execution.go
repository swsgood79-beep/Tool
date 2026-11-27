package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Native execution is deliberately an interface until the ERP transaction
// implementation has been verified. No HTTP route constructs a live driver.
type ERPReceiptDriver interface {
	ExecuteVerifiedTransaction(context.Context, ERPReceiptPlan) (ERPReceiptOutcome, error)
}
type ERPReceiptOutcome struct{ Status, Document, Message string }
type ERPReceiptExecution struct {
	ID, Actor string
	At        time.Time
	Plan      ERPReceiptPlan
	Outcome   ERPReceiptOutcome
}

func receiptExecutionFile(dir, id string) string {
	return filepath.Join(dir, "erp-receipt-"+digest([]byte(id))+".dat")
}
func saveReceiptExecution(path string, record ERPReceiptExecution) error {
	raw := jsonRaw(record)
	sealed, e := protect(raw)
	if e != nil {
		return e
	}
	check, e := unprotect(sealed)
	if e != nil || string(check) != string(raw) {
		return fmt.Errorf("입고 이력 보호 검증 실패")
	}
	return atomicFile(path, sealed)
}
func executeERPReceipt(ctx context.Context, dir, id, actor string, plan ERPReceiptPlan, driver ERPReceiptDriver) (ERPReceiptOutcome, error) {
	if id == "" || actor == "" || driver == nil {
		return ERPReceiptOutcome{}, fmt.Errorf("검증된 저장 연결과 실행자 정보가 필요합니다")
	}
	checked, e := buildERPReceiptPlan(plan.Lines)
	if e != nil {
		return ERPReceiptOutcome{}, e
	}
	if checked.Unit != plan.Unit || checked.CustomerCode != plan.CustomerCode {
		return ERPReceiptOutcome{}, fmt.Errorf("미리보기 내용이 일치하지 않습니다")
	}
	// Serializes all Alpha receipt attempts, including concurrent app instances.
	lock := filepath.Join(dir, "erp-receipt-execution.lock")
	f, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return ERPReceiptOutcome{Status: "uncertain"}, fmt.Errorf("입고 처리 중이거나 이전 실행의 결과 확인이 필요합니다")
	}
	f.Close()
	releaseLock := true
	defer func() {
		if releaseLock {
			os.Remove(lock)
		}
	}()
	path := receiptExecutionFile(dir, id)
	if sealed, e := os.ReadFile(path); e == nil {
		raw, e := unprotect(sealed)
		var old ERPReceiptExecution
		if e != nil || json.Unmarshal(raw, &old) != nil {
			return ERPReceiptOutcome{}, fmt.Errorf("기존 입고 이력을 읽지 못했습니다")
		}
		if old.Actor != actor || digest(jsonRaw(old.Plan)) != digest(jsonRaw(plan)) {
			return ERPReceiptOutcome{}, fmt.Errorf("동일 요청 ID의 내용이 다릅니다")
		}
		return old.Outcome, nil
	} else if !os.IsNotExist(e) {
		return ERPReceiptOutcome{}, e
	}
	record := ERPReceiptExecution{ID: id, Actor: actor, At: time.Now(), Plan: plan, Outcome: ERPReceiptOutcome{Status: "uncertain", Message: "전송 전 기록. 중단 후 자동 재전송 금지, ERP 대조 필요"}}
	if e = saveReceiptExecution(path, record); e != nil {
		return ERPReceiptOutcome{}, e
	}
	releaseLock = false // Crash or panic after this point must retain the global recovery lock.
	outcome, callErr := driver.ExecuteVerifiedTransaction(ctx, plan)
	switch outcome.Status {
	case "committed":
		if outcome.Document == "" {
			outcome = record.Outcome
		}
	case "rolled_back":
	case "uncertain":
	default:
		outcome = record.Outcome
	}
	if callErr != nil && outcome.Status != "rolled_back" {
		outcome = record.Outcome
	}
	record.Outcome = outcome
	if e = saveReceiptExecution(path, record); e != nil {
		return ERPReceiptOutcome{Status: "uncertain", Message: "ERP 결과 기록 저장 실패. ERP 대조 필요"}, e
	}
	if outcome.Status == "committed" || outcome.Status == "rolled_back" {
		releaseLock = true
	}
	return outcome, callErr
}
