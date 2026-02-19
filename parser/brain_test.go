package parser

import (
	"reflect"
	"testing"
)

func TestVectorize(t *testing.T) {
	regexPatterns := []string{
		`blk_\d+`,
		`\d+\.\d+\.\d+\.\d+`,
		`\d+`,
	}

	lp, err := NewLogParser(regexPatterns)
	if err != nil {
		t.Fatalf("NewLogParser failed: %v", err)
	}

	rawLogs := []string{
		"blk_101 info: Block 101 received from 10.0.0.1",
		"blk_102 info: Block 102 received from 10.0.0.2",
		"blk_103 warn: Connection refused",
	}

	got := lp.Vectorize(rawLogs)

	want := LogTable{
		Rows: []LogLine{
			{Tokens: []LogToken{{"<*>", 0}, {"info:", 1}, {"Block", 2}, {"<*>", 3}, {"received", 4}, {"from", 5}, {"<*>", 6}}},
			{Tokens: []LogToken{{"<*>", 0}, {"info:", 1}, {"Block", 2}, {"<*>", 3}, {"received", 4}, {"from", 5}, {"<*>", 6}}},
			{Tokens: []LogToken{{"<*>", 0}, {"warn:", 1}, {"Connection", 2}, {"refused", 3}}},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Vectorize mismatch")
		for i, row := range got.Rows {
			t.Errorf("  row %d got:  %v", i, row.Tokens)
			if i < len(want.Rows) {
				t.Errorf("  row %d want: %v", i, want.Rows[i].Tokens)
			}
		}
	}
}

func TestGenerateFrequencyVectors(t *testing.T) {
	regexPatterns := []string{
		`blk_\d+`,
		`\d+\.\d+\.\d+\.\d+`,
		`\d+`,
	}

	lp, err := NewLogParser(regexPatterns)
	if err != nil {
		t.Fatalf("NewLogParser failed: %v", err)
	}

	rawLogs := []string{
		"blk_101 info: Block 101 received from 10.0.0.1",
		"blk_102 info: Block 102 received from 10.0.0.2",
		"blk_103 warn: Connection refused",
	}

	logTable := lp.Vectorize(rawLogs)
	got := lp.GenerateFrequencyVectors(logTable)

	want := []FrequencyVector{
		{3, "<*>", 0},
		{2, "info:", 1},
		{2, "Block", 2},
		{2, "<*>", 3},
		{2, "received", 4},
		{2, "from", 5},
		{2, "<*>", 6},

		{3, "<*>", 0},
		{2, "info:", 1},
		{2, "Block", 2},
		{2, "<*>", 3},
		{2, "received", 4},
		{2, "from", 5},
		{2, "<*>", 6},

		{3, "<*>", 0},
		{1, "warn:", 1},
		{1, "Connection", 2},
		{1, "refused", 3},
	}

	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got (%d, %s, %d), want (%d, %s, %d)",
				i,
				got[i].Frequency, got[i].Token, got[i].Column,
				want[i].Frequency, want[i].Token, want[i].Column,
			)
		}
	}
}
