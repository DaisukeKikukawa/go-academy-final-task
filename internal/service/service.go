package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

type Item struct {
	Category string `json:"category"`
	Name     string `json:"name"`
}

type Today struct {
	Area     repository.Area `json:"area"`
	Date     string          `json:"date"`
	Today    []Item          `json:"today"`
	Tomorrow []Item          `json:"tomorrow"`
	Message  string          `json:"message"`
}

type Day struct {
	Date    string `json:"date"`
	Weekday int    `json:"weekday"`
	Items   []Item `json:"items"`
}

type Schedule struct {
	Area repository.Area `json:"area"`
	From string          `json:"from"`
	To   string          `json:"to"`
	Days []Day           `json:"days"`
}

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Municipalities() ([]repository.Municipality, error) {
	return s.repo.ListMunicipalities()
}

func (s *Service) SearchAreas(code, q string) ([]repository.Area, error) {
	return s.repo.SearchAreas(code, q)
}

func (s *Service) Area(id int) (repository.Area, error) {
	return s.repo.FindArea(id)
}

func (s *Service) Today(id int, now time.Time) (Today, error) {
	a, err := s.repo.FindArea(id)
	if err != nil {
		return Today{}, err
	}
	rules, err := s.repo.ListRules(id)
	if err != nil {
		return Today{}, err
	}
	today := StartOfDay(now)
	t, m := itemsOn(rules, today), itemsOn(rules, today.AddDate(0, 0, 1))
	return Today{
		Area:     a,
		Date:     today.Format("2006-01-02"),
		Today:    t,
		Tomorrow: m,
		Message:  fmt.Sprintf("今日は%s。明日は%s。", phrase(t), phrase(m)),
	}, nil
}

func (s *Service) Schedule(id int, from, to time.Time) (Schedule, error) {
	a, err := s.repo.FindArea(id)
	if err != nil {
		return Schedule{}, err
	}
	rules, err := s.repo.ListRules(id)
	if err != nil {
		return Schedule{}, err
	}
	days := []Day{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		days = append(days, Day{Date: d.Format("2006-01-02"), Weekday: int(d.Weekday()), Items: itemsOn(rules, d)})
	}
	return Schedule{Area: a, From: from.Format("2006-01-02"), To: to.Format("2006-01-02"), Days: days}, nil
}

func StartOfDay(t time.Time) time.Time {
	t = t.In(JST)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, JST)
}

func itemsOn(rules []repository.Rule, d time.Time) []Item {
	seen := map[string]bool{}
	items := []Item{}
	for _, r := range rules {
		if r.Matches(d) && !seen[r.Category] {
			seen[r.Category] = true
			items = append(items, Item{r.Category, r.Name})
		}
	}
	return items
}

func phrase(items []Item) string {
	if len(items) == 0 {
		return "収集がありません"
	}
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.Name
	}
	return strings.Join(names, "と") + "の日です"
}
