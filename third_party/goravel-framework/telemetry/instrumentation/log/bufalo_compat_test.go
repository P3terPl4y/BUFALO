package log

import (
	"bytes"
	"go.opentelemetry.io/otel/attribute"
	"math"
	"testing"
)

func TestBufaloCompatibilityPreservesStructuredValues(t *testing.T) {
	data := []byte{0, 1, 255}
	if !bytes.Equal(toValue(data).AsByteSlice(), data) {
		t.Fatal("byte value changed")
	}
	if toValue("message").AsString() != "message" {
		t.Fatal("string changed")
	}
	if toValue(uint64(math.MaxUint64)).AsString() != "18446744073709551615" {
		t.Fatal("unsigned overflow")
	}
	nested := toValue(map[string]any{"items": []int{1, 2}, "active": true}).AsMap()
	values := map[attribute.Key]attribute.Value{}
	for _, kv := range nested {
		values[kv.Key] = kv.Value
	}
	if len(values["items"].AsSlice()) != 2 || !values["active"].AsBool() {
		t.Fatal("structured data changed")
	}
	original := attribute.StringValue("preserved")
	if toValue(original).AsString() != "preserved" {
		t.Fatal("attribute conversion changed")
	}
	if toValue(nil).Type() != attribute.INVALID {
		t.Fatal("nil changed")
	}
}
