package engine

import (
	"fmt"
	"strings"
	d "tramflow/internal/domain"
)

func Summarize(response d.Response, focus d.Focus) d.Summary {
	s := d.Summary{
		ID:     response.ID,
		Status: "unavailable",
		Focus:  focus,
		Window: response.Descriptor.Selection.Window,
		Reason: "FOCUS_NOT_SUPPORTED",
	}
	if focus.Kind == "route_stop" {
		return s
	}
	selected := []d.RouteReading{}
	for _, r := range response.Totals {
		if focus.Kind == "overall" || r.RouteID == focus.Route {
			selected = append(selected, r)
		}
	}
	if len(selected) == 0 {
		return s
	}
	s.Status = "ready"
	s.Reason = ""
	s.Facts = []d.Fact{}
	total := 0.0
	maxIndex := -1.0
	for _, r := range selected {
		total += r.Evaluated.Boardings
		if r.Evaluated.Index.Value != nil && *r.Evaluated.Index.Value > maxIndex {
			maxIndex = *r.Evaluated.Index.Value
		}
	}
	s.Facts = append(s.Facts, d.Fact{
		Key:  "boardings",
		Text: fmt.Sprintf("Ожидаемые посадки за выбранный период: %.2f.", total),
	})
	if maxIndex >= 0 {
		s.Facts = append(s.Facts, d.Fact{
			Key:  "load",
			Text: fmt.Sprintf("Максимальный маршрутный индекс периода: %.2f; это не заполненность салона.", maxIndex),
		})
	}
	delta := 0.0
	for _, r := range selected {
		delta += r.Delta.Boardings
	}
	if response.Descriptor.Kind == "scenario" {
		s.Facts = append(s.Facts, d.Fact{
			Key:  "scenario_delta",
			Text: fmt.Sprintf("Изменение посадок относительно исходного прогноза: %+.2f.", delta),
		})
	}
	texts := []string{}
	for _, f := range s.Facts {
		texts = append(texts, f.Text)
	}
	s.Text = strings.Join(texts, " ")
	return s
}
