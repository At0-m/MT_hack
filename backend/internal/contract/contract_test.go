package contract

import "testing"

func TestContractLoads(t *testing.T) {
	c, err := Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Spec.Paths.Len() != 17 {
		t.Fatal(c.Spec.Paths.Len())
	}
}
func TestStrictJSON(t *testing.T) {
	for _, b := range []string{`{"a":1,"a":2}`, `{"a":[{"b":1,"b":2}]}`, `{} {}`, `{"a":NaN}`, `null trailing`} {
		if UniqueJSON([]byte(b)) == nil {
			t.Fatal(b)
		}
	}
	if err := UniqueJSON([]byte(`{"a":[{},1,"b",null]}`)); err != nil {
		t.Fatal(err)
	}
}
func TestNullAndUnknownAreRejected(t *testing.T) {
	c, err := Load("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []string{`{"selection":null}`, `{"selection":{},"extra":true}`} {
		if c.Validate("ForecastQuery", []byte(b)) == nil {
			t.Fatal(b)
		}
	}
}
