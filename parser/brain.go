package parser

import (
	"regexp"
	"strings"
)

// LogParser hält die Konfiguration für das Parsing.
type LogParser struct {
	RegexPatterns []*regexp.Regexp
}

// NewLogParser initialisiert den Parser mit einer Liste von Regex-Strings.
func NewLogParser(regexStrings []string) (*LogParser, error) {
	var patterns []*regexp.Regexp
	for _, s := range regexStrings {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, re)
	}
	return &LogParser{RegexPatterns: patterns}, nil
}

// Preprocess bereinigt eine Log-Zeile, indem Patterns durch <*> ersetzt werden.
func (lp *LogParser) Preprocess(logLine string) string {
	for _, re := range lp.RegexPatterns {
		logLine = re.ReplaceAllString(logLine, "<*>")
	}
	return logLine
}

// LogToken repräsentiert ein einzelnes Wort mit Metadaten.
// Für Schritt 1 speichern wir das Wort und später dessen Häufigkeit.
type LogToken struct {
	Content   string
	Frequency int
}

// LogEntry repräsentiert eine verarbeitete Log-Zeile.
type LogEntry struct {
	LineID int
	Tokens []LogToken
}

// LogGroup fasst Logs gleicher Länge zusammen und speichert Spalten-Statistiken.
type LogGroup struct {
	Length int
	Logs   []*LogEntry
	// ColumnCounts speichert: SpaltenIndex -> Wort -> Anzahl Vorkommen
	// Dies entspricht dem "Frequency Vector" aus dem Paper/Python-Code.
	ColumnCounts map[int]map[string]int
}

// Vectorize führt Preprocessing, Tokenisierung und Gruppierung durch.
// Rückgabe ist eine Map: Länge -> LogGroup
func (lp *LogParser) Vectorize(rawLogs []string) map[int]*LogGroup {
	groups := make(map[int]*LogGroup)

	for id, line := range rawLogs {
		// 1. Preprocessing
		cleanLine := lp.Preprocess(line)

		// 2. Tokenisierung (Splitten an Whitespaces)
		// strings.Fields behandelt auch mehrere Leerzeichen korrekt im Gegensatz zu Split
		tokenStrings := strings.Fields(cleanLine)
		length := len(tokenStrings)

		// Gruppe initialisieren, falls nicht vorhanden
		if _, exists := groups[length]; !exists {
			groups[length] = &LogGroup{
				Length:       length,
				Logs:         []*LogEntry{},
				ColumnCounts: make(map[int]map[string]int),
			}
		}
		group := groups[length]

		// 3. Log Entry erstellen
		tokens := make([]LogToken, length)
		for i, t := range tokenStrings {
			tokens[i] = LogToken{Content: t}
		}

		entry := &LogEntry{
			LineID: id,
			Tokens: tokens,
		}
		group.Logs = append(group.Logs, entry)

		// 4. Häufigkeiten zählen (Vektorisierung)
		for i, t := range tokenStrings {
			if group.ColumnCounts[i] == nil {
				group.ColumnCounts[i] = make(map[string]int)
			}
			group.ColumnCounts[i][t]++
		}
	}

	// Optional: Die ermittelten Häufigkeiten direkt in die LogTokens zurückschreiben
	// Das erleichtert Schritt 2 (LCP Suche).
	for _, group := range groups {
		for _, log := range group.Logs {
			for i := range log.Tokens {
				tokenContent := log.Tokens[i].Content
				log.Tokens[i].Frequency = group.ColumnCounts[i][tokenContent]
			}
		}
	}

	return groups
}
