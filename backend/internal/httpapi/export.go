package httpapi

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"
	d "tramflow/internal/domain"
)

func number(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'g', -1, 64)
}

func safeCell(v string) string {
	if v != "" && strings.ContainsAny(v[:1], "=+-@\t\r") {
		return "'" + v
	}
	return v
}

func CSV(result d.Response) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xef, 0xbb, 0xbf})
	cw := csv.NewWriter(&buf)
	cw.Comma = ';'
	cw.UseCRLF = true
	if err := cw.Write([]string{
		"calculation_id",
		"forecast_snapshot_id",
		"route_id",
		"from",
		"to",
		"baseline_boardings",
		"evaluated_boardings",
		"baseline_vehicle_hours",
		"evaluated_vehicle_hours",
		"baseline_load_index",
		"evaluated_load_index",
		"required_vehicle_count",
		"fleet_source",
		"fleet_is_proxy",
		"weather_mode",
	}); err != nil {
		return nil, err
	}
	for _, f := range result.Frames {
		for _, r := range f.Routes {
			row := []string{
				result.ID,
				result.Provenance.SnapshotID,
				safeCell(r.RouteID),
				f.Window.From.Format(time.RFC3339),
				f.Window.To.Format(time.RFC3339),
				strconv.FormatFloat(r.Baseline.Boardings, 'g', -1, 64),
				strconv.FormatFloat(r.Evaluated.Boardings, 'g', -1, 64),
				number(r.Baseline.VehicleHours.Value),
				number(r.Evaluated.VehicleHours.Value),
				number(r.Baseline.Index.Value),
				number(r.Evaluated.Index.Value),
				number(r.Evaluated.Required.Value),
				r.Evaluated.Source,
				strconv.FormatBool(r.Evaluated.Proxy),
				r.Weather.Mode,
			}
			if err := cw.Write(row); err != nil {
				return nil, err
			}
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return nil, err
	}
	if buf.Len() > 4194304 {
		return nil, d.Fail(413, "EXPORT_TOO_LARGE", "Экспорт превышает 4 MiB.")
	}
	return buf.Bytes(), nil
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	var q d.ExportRequest
	if !s.body(w, r, "ExportRequest", &q) {
		return
	}
	result, err := s.Service.Calculate(r.Context(), q.Calculation)
	if err != nil {
		s.fail(w, err)
		return
	}
	if result.ID != q.Expected {
		s.problem(w, 409, "CALCULATION_MISMATCH", "Расчёт не совпадает с подтверждённым идентификатором.")
		return
	}
	b, err := CSV(result)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("X-Calculation-ID", result.ID)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="tramflow-`+result.ID+`.csv"`)
	_, _ = w.Write(b)
}
