package main

import (
	"context"
	"fmt"
	"time"
)

type erpDBPreviewKey struct{}
type erpDBPreviewRange struct{ From, To string }
type ERPDBPurchase struct {
	Robot           string `json:"robot"`
	RobotMatches    int    `json:"robot_matches"`
	Unit            string `json:"unit"`
	PurchaseDate    string `json:"purchase_date"`
	Purchase        string `json:"purchase"`
	Sequence        int    `json:"sequence"`
	Order           string `json:"order"`
	Project         string `json:"project"`
	Item            string `json:"item"`
	PartsType       string `json:"parts_type"`
	LineSize        string `json:"line_size"`
	BOM             string `json:"bom"`
	Due             string `json:"due"`
	Quantity        int    `json:"quantity"`
	Received        int    `json:"received"`
	WarehouseStatus string `json:"warehouse_status"`
	ItemName        string `json:"item_name"`
	MasterSize      string `json:"master_size"`
	PartNo          string `json:"part_no"`
}

func erpDBPreviewDates(from, to string) (erpDBPreviewRange, error) {
	a, e := time.Parse("2006-01-02", from)
	if e != nil {
		return erpDBPreviewRange{}, fmt.Errorf("입고예정일 시작일을 입력해 주세요")
	}
	b, e := time.Parse("2006-01-02", to)
	if e != nil || b.Before(a) || b.Sub(a) > 370*24*time.Hour {
		return erpDBPreviewRange{}, fmt.Errorf("종료일은 시작일 이후 370일 이내로 지정해 주세요")
	}
	return erpDBPreviewRange{from, to}, nil
}
func erpDBPreviewOptions(ctx context.Context) (erpDBPreviewRange, bool) {
	v, ok := ctx.Value(erpDBPreviewKey{}).(erpDBPreviewRange)
	return v, ok
}
