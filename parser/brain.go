package parser

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// LogParser hält die Konfiguration für das Parsing.
type LogParser struct {
	RegexPatterns []*regexp.Regexp
	Weight        float64
}

// NewLogParser initialisiert den Parser mit einer Liste von Regex-Strings.
func NewLogParser(regexStrings []string, weight float64) (*LogParser, error) {
	var patterns []*regexp.Regexp
	for _, s := range regexStrings {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, re)
	}
	return &LogParser{RegexPatterns: patterns, Weight: weight}, nil
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
	Frequency     int
	LogId         int
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

func (lp *LogParser) GenerateFrequencyVectors(logTable LogTable) map[int][]FrequencyVector {
	freqMap := make(map[colFreqKey]int)
	totalTokens := 0

	for _, row := range logTable.Rows {
		totalTokens += len(row.Tokens)
		for _, token := range row.Tokens {
			freqMap[colFreqKey{column: token.ColumnIdx, content: token.Content}]++
		}
	}

	frequencies := make(map[int][]FrequencyVector, len(logTable.Rows))
	for logNr, row := range logTable.Rows {
		for _, token := range row.Tokens {
			freq := freqMap[colFreqKey{column: token.ColumnIdx, content: token.Content}]
			frequencies[logNr] = append(frequencies[logNr], FrequencyVector{Frequency: freq, Token: token.Content, Column: token.ColumnIdx})
		}
	}
	return frequencies
}

func (lp *LogParser) FindWordCombinations(freqVectors map[int][]FrequencyVector) []WordCombination {
	wordCombinationSet := make(map[int]WordCombination, len(freqVectors))
	for logId, freqVecs := range freqVectors {
		maxFreq := 0
		threshold := 0
		for _, freqVec := range freqVecs {
			val, ok := wordCombinationSet[freqVec.Frequency]
			if val.Frequency > maxFreq {
				maxFreq = val.Frequency
				threshold = int(math.Round(lp.Weight * float64(maxFreq)))
			}
			if !ok {
				wordCombinationSet[freqVec.Frequency] = WordCombination{FrequencyVecs: []FrequencyVector{freqVec}, Frequency: freqVec.Frequency, LogId: logId}
			} else {
				val.FrequencyVecs = append(val.FrequencyVecs, freqVec)
				wordCombinationSet[freqVec.Frequency] = val
			}
		}
		for _, wc := range wordCombinationSet {
			if wc.Frequency < threshold {
				delete(wordCombinationSet, wc.Frequency)
			}
		}
		for _, wc := range wordCombinationSet {
			fmt.Printf("%d: %v\n", wc.Frequency, wc.FrequencyVecs)
		}
		break

	}

	return nil
}
