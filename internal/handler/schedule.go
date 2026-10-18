package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/schedule"
)

const (
	defaultScheduleDays = 30 // from/to 未指定のときの期間
	maxScheduleDays     = 92 // 一度に返す最大日数(約3ヶ月)
)

// loadRules は地区の存在確認とルール取得をまとめて行う。失敗時はレスポンス済みで nil を返す。
func (s *Server) loadRules(c *gin.Context) (*repository.Area, []repository.CollectionRule) {
	id, ok := areaIDParam(c)
	if !ok {
		return nil, nil
	}
	area := s.loadArea(c, id)
	if area == nil {
		return nil, nil
	}
	rules, err := repository.ListRulesByArea(c.Request.Context(), s.DB, id)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "db_error", err.Error())
		return nil, nil
	}
	return area, rules
}

// GET /v1/areas/:id/schedule?from=2026-10-01&to=2026-10-31
func (s *Server) getSchedule(c *gin.Context) {
	area, rules := s.loadRules(c)
	if area == nil {
		return
	}
	today := schedule.Today()
	from, err := parseDateOr(c.Query("from"), today)
	if err != nil {
		errorJSON(c, http.StatusBadRequest, "bad_from", "from は YYYY-MM-DD で指定してください")
		return
	}
	to, err := parseDateOr(c.Query("to"), from.AddDate(0, 0, defaultScheduleDays-1))
	if err != nil {
		errorJSON(c, http.StatusBadRequest, "bad_to", "to は YYYY-MM-DD で指定してください")
		return
	}
	if to.Before(from) {
		errorJSON(c, http.StatusBadRequest, "bad_range", "to は from 以降の日付にしてください")
		return
	}
	if to.Sub(from) > time.Duration(maxScheduleDays-1)*24*time.Hour {
		errorJSON(c, http.StatusBadRequest, "range_too_long", fmt.Sprintf("期間は最大 %d 日までです", maxScheduleDays))
		return
	}

	days := schedule.Expand(rules, from, to)
	c.JSON(http.StatusOK, gin.H{
		"area": area,
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
		"days": days,
	})
}

// GET /v1/areas/:id/today
func (s *Server) getToday(c *gin.Context) {
	area, rules := s.loadRules(c)
	if area == nil {
		return
	}
	today := schedule.Today()
	tomorrow := today.AddDate(0, 0, 1)
	todayItems := schedule.ItemsOn(rules, today)
	tomorrowItems := schedule.ItemsOn(rules, tomorrow)

	c.JSON(http.StatusOK, gin.H{
		"area":     area,
		"date":     today.Format("2006-01-02"),
		"weekday":  int(today.Weekday()),
		"today":    todayItems,
		"tomorrow": tomorrowItems,
		"message":  todayMessage(todayItems, tomorrowItems),
	})
}

// todayMessage は「今日は燃やすごみの日です。明日は資源の日です。」のような一言を作る。
func todayMessage(today, tomorrow []schedule.Item) string {
	var sb strings.Builder
	if len(today) == 0 {
		sb.WriteString("今日は収集がありません。")
	} else {
		sb.WriteString("今日は" + joinNames(today) + "の日です。")
	}
	if len(tomorrow) == 0 {
		sb.WriteString("明日は収集がありません。")
	} else {
		sb.WriteString("明日は" + joinNames(tomorrow) + "の日です。前の晩には出さないでください。")
	}
	return sb.String()
}

func joinNames(items []schedule.Item) string {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.CategoryName
	}
	return strings.Join(names, "と")
}

// parseDateOr は "2026-10-01" を JST の日付にする。空なら fallback。
func parseDateOr(s string, fallback time.Time) (time.Time, error) {
	if s == "" {
		return fallback, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, schedule.JST)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
