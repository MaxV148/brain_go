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

// LogEntry repräsentiert eine verarbeitete Log-Zeile.
type LogEntry struct {
	LineID int
	Tokens []LogToken
}

type LogToken struct {
	Content   string
	ColumnIdx int
}

type LogLine struct {
	Tokens []LogToken
}

type LogTable struct {
	Rows []LogLine
}

type FrequencyVector struct {
	Frequency int
	Token     string
	Column    int
}

type WordCombination struct {
	FrequencyVecs []FrequencyVector
	Length        int
}

// colFreqKey identifies a unique (column, token) pair for frequency counting.
type colFreqKey struct {
	column  int
	content string
}

func (lp *LogParser) Vectorize(rawLogs []string) LogTable {
	logLines := make([]LogLine, 0, len(rawLogs))

	for _, line := range rawLogs {
		cleanLine := lp.Preprocess(line)

		splitLine := strings.Fields(cleanLine)
		logLine := make([]LogToken, 0, len(splitLine))
		for idx, token := range splitLine {
			logLine = append(logLine, LogToken{Content: token, ColumnIdx: idx})
		}
		logLines = append(logLines, LogLine{Tokens: logLine})
	}

	return LogTable{Rows: logLines}
}

func (lp *LogParser) GenerateFrequencyVectors(logTable LogTable) []FrequencyVector {
	freqMap := make(map[colFreqKey]int)
	totalTokens := 0

	for _, row := range logTable.Rows {
		totalTokens += len(row.Tokens)
		for _, token := range row.Tokens {
			freqMap[colFreqKey{column: token.ColumnIdx, content: token.Content}]++
		}
	}

	frequencies := make([]FrequencyVector, 0, totalTokens)
	for _, row := range logTable.Rows {
		for _, token := range row.Tokens {
			freq := freqMap[colFreqKey{column: token.ColumnIdx, content: token.Content}]
			frequencies = append(frequencies, FrequencyVector{Frequency: freq, Token: token.Content, Column: token.ColumnIdx})
		}
	}
	return frequencies
}
