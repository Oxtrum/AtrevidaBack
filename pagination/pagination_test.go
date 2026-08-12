package pagination

import (
	"errors"
	"testing"
)

type testPosition struct {
	Name string `json:"n"`
	ID   int    `json:"i"`
}

func TestParseLegacy(t *testing.T) {
	request, err := Parse("", "")
	if err != nil || request.Enabled {
		t.Fatalf("Parse legacy = %+v, %v", request, err)
	}
}

func TestParseValidatesLimits(t *testing.T) {
	for _, raw := range []string{"0", "101", "abc"} {
		if _, err := Parse(raw, ""); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Parse(%q) error = %v", raw, err)
		}
	}
	request, err := Parse("", "cursor")
	if err != nil || request.Limit != DefaultLimit || !request.Enabled {
		t.Fatalf("Parse cursor = %+v, %v", request, err)
	}
}

func TestBuildEmptyAndExactPage(t *testing.T) {
	request, _ := Parse("2", "")
	empty, emptyMeta, err := Build([]testPosition{}, request, "test", "filter", func(item testPosition) any { return item })
	if err != nil || len(empty) != 0 || emptyMeta == nil || emptyMeta.HasMore || emptyMeta.NextCursor != nil {
		t.Fatalf("empty Build = %v, %+v, %v", empty, emptyMeta, err)
	}
	exact, exactMeta, err := Build([]testPosition{{"a", 1}, {"b", 2}}, request, "test", "filter", func(item testPosition) any { return item })
	if err != nil || len(exact) != 2 || exactMeta == nil || exactMeta.HasMore || exactMeta.NextCursor != nil {
		t.Fatalf("exact Build = %v, %+v, %v", exact, exactMeta, err)
	}
}

func TestBuildAndDecode(t *testing.T) {
	request, _ := Parse("2", "")
	items, meta, err := Build([]testPosition{{"a", 1}, {"b", 2}, {"c", 3}}, request, "test", map[string]string{"q": "x"}, func(item testPosition) any { return item })
	if err != nil || len(items) != 2 || meta == nil || !meta.HasMore || meta.NextCursor == nil {
		t.Fatalf("Build = %v, %+v, %v", items, meta, err)
	}
	var decoded testPosition
	if err := Decode(*meta.NextCursor, "test", map[string]string{"q": "x"}, &decoded); err != nil || decoded.ID != 2 {
		t.Fatalf("Decode = %+v, %v", decoded, err)
	}
}

func TestAddTotalCalculatesPages(t *testing.T) {
	meta := &Metadata{Limit: 50}
	AddTotal(meta, 2501)
	if meta.TotalRecords == nil || *meta.TotalRecords != 2501 {
		t.Fatalf("TotalRecords = %v, want 2501", meta.TotalRecords)
	}
	if meta.TotalPages == nil || *meta.TotalPages != 51 {
		t.Fatalf("TotalPages = %v, want 51", meta.TotalPages)
	}
}

func TestDecodeRejectsOtherFiltersAndTampering(t *testing.T) {
	request, _ := Parse("1", "")
	_, meta, _ := Build([]testPosition{{"a", 1}, {"b", 2}}, request, "test", "filter-a", func(item testPosition) any { return item })
	var decoded testPosition
	if err := Decode(*meta.NextCursor, "test", "filter-b", &decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Decode filter error = %v", err)
	}
	tampered := *meta.NextCursor
	if tampered[len(tampered)-1] == 'A' {
		tampered = tampered[:len(tampered)-1] + "B"
	} else {
		tampered = tampered[:len(tampered)-1] + "A"
	}
	if err := Decode(tampered, "test", "filter-a", &decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Decode tampered error = %v", err)
	}
}
