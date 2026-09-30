package ingest

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const header = "route,dt_15min,ediny,troyka,koshelek,bank_cards,lgotnye,vedomstvennye,tat,other,total\n"

func TestHourSumNotAverageOrDoubleCount(t *testing.T) {
	a, _ := New("start")
	var csv strings.Builder
	csv.WriteString(header)
	for _, minute := range []int{0, 15, 30, 45} {
		fmt.Fprintf(&csv, "1 трамвай,2025-01-01 04:%02d:00,1,2,0,0,0,0,0,0,3\n", minute)
	}
	if err := a.Add(strings.NewReader(csv.String())); err != nil {
		t.Fatal(err)
	}
	hours := a.Hours()
	if len(hours) != 1 || hours[0].Boardings != 12 || hours[0].ObservedQuarters != 4 {
		t.Fatalf("incorrect sum: %v", hours)
	}
	var output bytes.Buffer
	if err := Write(&output, hours); err != nil {
		t.Fatal(err)
	}
	if output.String() != "route;date;hour;boardings\n1;2025-01-01;4;12\n" {
		t.Fatal(output.String())
	}
	if err := a.Add(strings.NewReader(csv.String())); err == nil {
		t.Fatal("cross-file duplicate accepted")
	}
}

func TestEndTimestampAndPartialCoverage(t *testing.T) {
	a, _ := New("end")
	if err := a.Add(strings.NewReader(header + "7,2025-01-01 00:00:00,0,0,4,0,0,0,0,0,4\n")); err != nil {
		t.Fatal(err)
	}
	hours := a.Hours()
	if len(hours) != 1 || hours[0].Time.Format("2006-01-02 15") != "2024-12-31 23" || hours[0].ObservedQuarters != 1 {
		t.Fatalf("incorrect midnight boundary: %v", hours)
	}
}

func TestInvalidRows(t *testing.T) {
	for _, row := range []string{
		"1,2025-01-01 00:00:00,1,0,0,0,0,0,0,0,2\n",
		"1,2025-01-01 00:00:00,NaN,0,0,0,0,0,0,0,0\n",
		"1,2025-01-01 00:01:00,0,0,0,0,0,0,0,0,0\n",
		"1,2025-01-01 00:00:00,-1,0,0,0,0,0,0,0,-1\n",
	} {
		a, _ := New("start")
		if err := a.Add(strings.NewReader(header + row)); err == nil {
			t.Fatalf("invalid row accepted: %s", row)
		}
	}
}
