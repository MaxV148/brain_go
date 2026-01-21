package parser

import (
	"testing"
)

func TestVectorize(t *testing.T) {
	// Beispiel-Logs
	rawLogs := []string{
		"blk_101 info: Block 101 received from 10.0.0.1", // 7 Tokens
		"blk_102 info: Block 102 received from 10.0.0.2", // 7 Tokens
		"blk_103 warn: Connection refused",               // 4 Tokens
	}

	// Regex Definitionen
	regexPatterns := []string{
		`blk_\d+`,            // Block ID
		`\d+\.\d+\.\d+\.\d+`, // IP Address
		`\d+`,                // Einzelne Zahlen
	}

	lp, err := NewLogParser(regexPatterns)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	groups := lp.Vectorize(rawLogs)

	// Test 1: Gruppierung prüfen
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// --- PRÜFUNG GRUPPE LÄNGE 7 ---
	group7, exists := groups[7] // Korrigiert von 6 auf 7
	if !exists {
		t.Fatalf("Group with length 7 missing. Available groups: %v", getKeys(groups))
	}

	if len(group7.Logs) != 2 {
		t.Errorf("Expected 2 logs in group 7, got %d", len(group7.Logs))
	}

	// --- PRÜFUNG GRUPPE LÄNGE 4 ---
	_, exists4 := groups[4] // Korrigiert von 3 auf 4
	if !exists4 {
		t.Fatalf("Group with length 4 missing")
	}

	// Test 2: Preprocessing & Tokenisierung
	// Das erste Token sollte "<*>" sein (wegen blk_101)
	firstToken := group7.Logs[0].Tokens[0].Content
	if firstToken != "<*>" {
		t.Errorf("Preprocessing failed. Expected '<*>', got '%s'", firstToken)
	}

	// Test 3: Häufigkeitszählung (Frequency Vector)
	// Spalte 0 ("<*>") sollte Häufigkeit 2 haben (kommt in beiden Logs vor)
	freqCol0 := group7.ColumnCounts[0]["<*>"]
	if freqCol0 != 2 {
		t.Errorf("Frequency count error for col 0. Expected 2, got %d", freqCol0)
	}

	// Spalte 2 ("Block") sollte Häufigkeit 2 haben
	// Index 2 entspricht dem Wort "Block" (0:<*>, 1:info:, 2:Block)
	freqCol2 := group7.ColumnCounts[2]["Block"]
	if freqCol2 != 2 {
		t.Errorf("Frequency count error for col 2. Expected 2, got %d", freqCol2)
	}

	// Prüfe, ob die Frequenz korrekt im Token gespeichert wurde
	if group7.Logs[0].Tokens[2].Frequency != 2 {
		t.Errorf("Token frequency not updated in struct. Expected 2, got %d", group7.Logs[0].Tokens[2].Frequency)
	}
}
func TestGroupByLCP(t *testing.T) {
	// Szenario: 4 Logs.
	// Wir wählen die Daten so, dass "User" und "login" dieselbe Frequenz haben (2),
	// damit sie ein gemeinsames LCP bilden.
	rawLogs := []string{
		"User 100 login",      // Freqs: User(2), 100(1), login(2) -> LCP: {User, login}
		"User 101 login",      // Freqs: User(2), 101(1), login(2) -> LCP: {User, login}
		"System failure disk", // Freqs: Alle 1 -> LCP: {System, failure, disk}
		"Other 102 logout",    // ÄNDERUNG: "Other" statt "User", damit "User" Freq 2 bleibt (gleich wie "login")
	}

	lp, _ := NewLogParser([]string{`\d+`}) // Zahlen maskieren
	groups := lp.Vectorize(rawLogs)

	// Wir testen die Gruppe mit Länge 3
	group3, exists := groups[3]
	if !exists {
		t.Fatalf("Group 3 missing")
	}

	// Threshold 0.5 (Standard)
	initialGroups := group3.GroupByLCP(0.5)

	// Erwartung 1: "User <*> login" (Log 1 und 2)
	// Da "User" und "login" jetzt beide Freq 2 haben, landen sie im selben Tupel.
	sig1 := "User <*> login"
	if g, exists := initialGroups[sig1]; !exists {
		t.Errorf("Expected group '%s' not found. Found keys: %v", sig1, getInitialGroupKeys(initialGroups))
	} else {
		if len(g.Logs) != 2 {
			t.Errorf("Group '%s' should have 2 logs, has %d", sig1, len(g.Logs))
		}
	}

	// Erwartung 2: "System failure disk" (Log 3)
	sig2 := "System failure disk"
	if _, exists := initialGroups[sig2]; !exists {
		t.Errorf("Expected group '%s' not found", sig2)
	}
}

// Hilfsfunktion für Fehlermeldungen im Test
func getInitialGroupKeys(m map[string]*InitialGroup) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Hilfsfunktion für Fehlermeldungen
func getKeys(m map[int]*LogGroup) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
