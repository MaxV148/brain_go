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

// ... (bisheriger Code bleibt unverändert) ...

// InitialGroup repräsentiert eine Gruppe von Logs, die dasselbe LCP (Longest Common Pattern) haben.
// Das ist das Ergebnis von Schritt 2.
type InitialGroup struct {
	Signature     string      // Eindeutige Signatur des Patterns (z.B. "Info <*> Service")
	Logs          []*LogEntry // Die Logs in dieser Gruppe
	RootFrequency int         // Die Häufigkeit der Wörter, die dieses Pattern bilden
}

// GroupByLCP führt Schritt 2 des Brain-Algorithmus aus.
// Es unterteilt eine LogGroup (Logs gleicher Länge) in initiale Cluster basierend auf dem häufigsten Wortmuster.
// thresholdPercent (üblicherweise 0.5) filtert Wortkombinationen, die zu selten sind.
func (lg *LogGroup) GroupByLCP(thresholdPercent float64) map[string]*InitialGroup {
	initialGroups := make(map[string]*InitialGroup)

	for _, log := range lg.Logs {
		// 1. Gruppiere Token-Indizes nach ihrer Frequenz
		// Map: Frequenz -> Liste von Positionen im Log, die diese Frequenz haben
		freqToIndices := make(map[int][]int)
		maxFreqInLog := 0

		for i, token := range log.Tokens {
			f := token.Frequency
			freqToIndices[f] = append(freqToIndices[f], i)
			if f > maxFreqInLog {
				maxFreqInLog = f
			}
		}

		// 2. Wähle die "Beste" Wortkombination (Root)
		// Kriterien gemäß Brain Paper/Code:
		// a) Frequenz muss über dem Schwellenwert liegen (relativ zum Max im Log).
		// b) Es muss die längste Kombination sein (meiste Wörter).
		// c) Bei gleicher Länge gewinnt die höhere Frequenz.

		bestFreq := -1
		maxWordCount := -1
		// Threshold berechnen: Wörter müssen oft genug vorkommen
		threshold := int(float64(maxFreqInLog) * thresholdPercent)

		for f, indices := range freqToIndices {
			// Filter: Frequenz zu niedrig?
			if f < threshold {
				continue
			}

			count := len(indices)

			// Ist diese Kombination länger als die bisher beste?
			if count > maxWordCount {
				maxWordCount = count
				bestFreq = f
			} else if count == maxWordCount {
				// Bei gleicher Länge: Nimm die mit der höheren Frequenz (stabiler)
				if f > bestFreq {
					bestFreq = f
				}
			}
		}

		// Fallback: Falls keine Kombination den Threshold erreicht (sehr selten),
		// nimm einfach die längste verfügbare, unabhängig vom Threshold.
		if bestFreq == -1 {
			for f, indices := range freqToIndices {
				count := len(indices)
				if count > maxWordCount {
					maxWordCount = count
					bestFreq = f
				} else if count == maxWordCount && f > bestFreq {
					bestFreq = f
				}
			}
		}

		// 3. Generiere die Signatur (LCP) für diesen Log-Eintrag
		// Wir erstellen einen String, der das Template repräsentiert.
		// Konstante (Teil des Roots) -> Wort selbst
		// Variable (Nicht Teil des Roots) -> "<*>"

		// Set der Indizes, die zum Root gehören, für schnellen Zugriff
		rootIndices := make(map[int]bool)
		for _, idx := range freqToIndices[bestFreq] {
			rootIndices[idx] = true
		}

		var signatureBuilder strings.Builder
		for i, token := range log.Tokens {
			if i > 0 {
				signatureBuilder.WriteString(" ")
			}
			if rootIndices[i] {
				signatureBuilder.WriteString(token.Content)
			} else {
				signatureBuilder.WriteString("<*>")
			}
		}
		signature := signatureBuilder.String()

		// 4. Log in die entsprechende InitialGroup einsortieren
		if _, exists := initialGroups[signature]; !exists {
			initialGroups[signature] = &InitialGroup{
				Signature:     signature,
				Logs:          []*LogEntry{},
				RootFrequency: bestFreq,
			}
		}
		initialGroups[signature].Logs = append(initialGroups[signature].Logs, log)
	}

	return initialGroups
}
