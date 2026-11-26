package main

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const shippingPageSize = 20

type ShippingMonth struct {
	Key, Title string
	Count      int
	Records    []ShippingRecord
}

func shippingDate(record ShippingRecord) time.Time {
	if record.ArchivedAt != nil {
		return *record.ArchivedAt
	}
	return record.InspectedAt
}
func shippingMonth(record ShippingRecord) string {
	return shippingDate(record).In(kst).Format("2006-01")
}
func shippingPhotoRevision(record *ShippingRecord) string { return digest(jsonRaw(record.Photos)) }
func shippingListView(s *State, base string, q url.Values, now time.Time) (map[string]any, error) {
	pending := q.Get("pending") == "1"
	query := strings.TrimSpace(q.Get("q"))
	period := q.Get("period")
	if period == "" {
		period = "all"
	}
	if pending {
		period = "all"
	}
	sortBy := q.Get("sort")
	if sortBy != "due" && sortBy != "number" {
		sortBy = "recent"
	}
	first := time.Date(now.In(kst).Year(), now.In(kst).Month(), 1, 0, 0, 0, 0, kst)
	start, end := "", ""
	switch period {
	case "all":
	case "month":
		start, end = day(first), day(first.AddDate(0, 1, -1))
	case "last":
		start, end = day(first.AddDate(0, -1, 0)), day(first.AddDate(0, 0, -1))
	case "custom":
		start, end = q.Get("start"), q.Get("end")
		for _, value := range []string{start, end} {
			if _, err := time.ParseInLocation("2006-01-02", value, kst); err != nil {
				return nil, errors.New("시작일과 종료일을 선택해 주세요.")
			}
		}
		if start > end {
			return nil, errors.New("종료일은 시작일 이후로 선택해 주세요.")
		}
	default:
		return nil, errors.New("기간 선택을 확인해 주세요.")
	}
	matches := []ShippingRecord{}
	pendingCount, archiveCount := 0, 0
	for _, r := range s.ShippingRecords {
		if r.ArchivedAt == nil {
			pendingCount++
		}
	}
	for _, r := range splitListRecords(s, false) {
		if r.ArchivedAt != nil {
			archiveCount++
		}
	}
	for _, record := range splitListRecords(s, pending) {
		record.ClientName = shippingCustomerName(s, record)
		if (record.ArchivedAt == nil) != pending {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(record.Project.Number+" "+record.Project.Name+" "+record.ClientName), strings.ToLower(query)) {
			continue
		}
		date := day(shippingDate(record))
		if start != "" && (date < start || date > end) {
			continue
		}
		matches = append(matches, record)
	}
	// Keep month groups together; sort choices arrange records within each month.
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		ma, mb := shippingMonth(a), shippingMonth(b)
		if ma != mb {
			return ma > mb
		}
		if sortBy == "due" && a.Project.ShippingDueDate != b.Project.ShippingDueDate {
			if a.Project.ShippingDueDate == "" {
				return false
			}
			if b.Project.ShippingDueDate == "" {
				return true
			}
			return a.Project.ShippingDueDate < b.Project.ShippingDueDate
		}
		if sortBy == "number" && a.Project.Number != b.Project.Number {
			return a.Project.Number < b.Project.Number
		}
		da, db := shippingDate(a), shippingDate(b)
		if !da.Equal(db) {
			return da.After(db)
		}
		return a.ID < b.ID
	})
	total := len(matches)
	pages := (total + shippingPageSize - 1) / shippingPageSize
	if pages == 0 {
		pages = 1
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	counts := map[string]int{}
	for _, record := range matches {
		counts[shippingMonth(record)]++
	}
	lo := (page - 1) * shippingPageSize
	hi := lo + shippingPageSize
	if hi > total {
		hi = total
	}
	records := matches[lo:hi]
	groups := []ShippingMonth{}
	for _, record := range records {
		key := shippingMonth(record)
		if len(groups) == 0 || groups[len(groups)-1].Key != key {
			t := shippingDate(record).In(kst)
			groups = append(groups, ShippingMonth{Key: key, Title: fmt.Sprintf("%d년 %d월", t.Year(), t.Month()), Count: counts[key]})
		}
		groups[len(groups)-1].Records = append(groups[len(groups)-1].Records, record)
	}
	values := url.Values{"q": {query}, "period": {period}, "sort": {sortBy}}
	if pending {
		values.Set("pending", "1")
	}
	if period == "custom" {
		values.Set("start", start)
		values.Set("end", end)
	}
	pageURL := func(n int) string {
		v := url.Values{}
		for k, items := range values {
			v[k] = append([]string{}, items...)
		}
		v.Set("page", strconv.Itoa(n))
		return base + "?" + v.Encode()
	}
	periodURLs := map[string]string{}
	for _, p := range []string{"all", "month", "last"} {
		v := url.Values{"q": {query}, "sort": {sortBy}, "period": {p}}
		periodURLs[p] = base + "?" + v.Encode()
	}
	rangeLabel := "전체 기간"
	if start != "" {
		rangeLabel = strings.ReplaceAll(start, "-", ".") + " ~ " + strings.ReplaceAll(end, "-", ".")
	}
	startInput, endInput := start, end
	if startInput == "" {
		startInput, endInput = day(first), day(now)
	}
	return map[string]any{"ShippingPending": pending, "Pending": pending, "Query": query, "Period": period, "Sort": sortBy, "PeriodURLs": periodURLs, "RangeLabel": rangeLabel, "Start": startInput, "End": endInput, "Records": records, "Months": groups, "PendingCount": pendingCount, "ArchivedCount": archiveCount, "Total": total, "Page": page, "Pages": pages, "HasPrevious": page > 1, "HasNext": page < pages, "PreviousURL": pageURL(page - 1), "NextURL": pageURL(page + 1)}, nil
}
