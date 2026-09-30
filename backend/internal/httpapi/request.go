package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/engine"
)

func (s *Server) body(w http.ResponseWriter, r *http.Request, name string, out any) bool {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/json" {
		s.problem(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Используйте application/json.")
		return false
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			s.problem(w, 413, "BODY_TOO_LARGE", "Лимит запроса 64 KiB.")
		} else {
			s.problem(w, 400, "INVALID_JSON", "Не удалось прочитать запрос.")
		}
		return false
	}
	if err = s.Contract.Validate(name, b); err != nil {
		if s.customHourlyRequest(name, b) {
			s.problem(w, 422, "CUSTOM_HOURLY_UNSUPPORTED", "Произвольный период поддерживает только суточную агрегацию.")
			return false
		}
		s.problem(w, 400, "INVALID_SCHEMA", "JSON не соответствует контракту: проверьте поля, null, типы и диапазоны.")
		return false
	}
	if err = json.Unmarshal(b, out); err != nil {
		s.problem(w, 400, "INVALID_JSON", "Некорректный JSON.")
		return false
	}
	return true
}

func param(r *http.Request, key string) (string, error) {
	v := r.URL.Query()[key]
	if len(v) != 1 || !engine.IDPattern.MatchString(v[0]) {
		return "", d.Fail(400, "INVALID_QUERY", "Нужен корректный параметр "+key+".")
	}
	return v[0], nil
}

// Distinguish an otherwise valid custom/hour request from malformed JSON.
// The contract requires 422 for this unsupported business combination.
func (s *Server) customHourlyRequest(schema string, body []byte) bool {
	if contract.UniqueJSON(body) != nil {
		return false
	}
	var request map[string]any
	if json.Unmarshal(body, &request) != nil {
		return false
	}
	container := request
	if schema == "ExportRequest" || schema == "SummaryQuery" {
		calculation, ok := request["calculation"].(map[string]any)
		if !ok {
			return false
		}
		container = calculation
	}
	selection, ok := container["selection"].(map[string]any)
	if !ok || selection["view_mode"] != "custom" || selection["resolution"] != "hour" {
		return false
	}
	selection["resolution"] = "day"
	corrected, err := json.Marshal(request)
	return err == nil && s.Contract.Validate(schema, corrected) == nil
}
