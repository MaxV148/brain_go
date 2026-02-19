package main

import (
	"brain_go/parser"
	"log"
)

func main() {

	rawLogs := []string{
		"blk_101 info: Block 101 received from 10.0.0.1",
		"blk_102 info: Block 102 received from 10.0.0.2",
		"blk_103 warn: Connection refused",
	}

	regexPatterns := []string{
		`blk_\d+`,            // Block ID
		`\d+\.\d+\.\d+\.\d+`, // IP Address
		`\d+`,                // Einzelne Zahlen
	}

	lp, err := parser.NewLogParser(regexPatterns)
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}

	tbl := lp.Vectorize(rawLogs)
	lp.GenerateFrequencyVectors(tbl)

}
