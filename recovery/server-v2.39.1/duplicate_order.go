package main

import (
	"fmt"
	"strings"
)

type duplicateOrderError struct{ Project Project }

func (e *duplicateOrderError) Error() string {
	return fmt.Sprintf("이미 등록된 수주번호입니다: %s · %s", e.Project.Number, e.Project.Name)
}
func duplicateOrderStatus(p Project) string {
	if p.Hidden {
		return "숨김"
	}
	if p.Shipping.Completed != nil || p.LegacyCompleted {
		return "검사 완료 · 날짜가 지나 일반 목록에서 제외될 수 있습니다"
	}
	if p.Setting.Pending || p.Shipping.Pending {
		return "검사 신청 · 관리자 검사 대기"
	}
	if p.Setting.Completed != nil {
		return "출하검사 대기"
	}
	return "세팅검사 대기"
}
func findOrderByNumber(s *State, number string) *Project {
	for i := range s.Projects {
		if strings.EqualFold(s.Projects[i].Number, strings.TrimSpace(number)) {
			return &s.Projects[i]
		}
	}
	return nil
}
