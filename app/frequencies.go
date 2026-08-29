package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed assets/frequencies.json
var frequencyDatabaseJSON []byte

func loadFrequencyDatabase() []FrequencyEntry {
	var entries []FrequencyEntry
	if err := json.Unmarshal(frequencyDatabaseJSON, &entries); err != nil {
		return []FrequencyEntry{}
	}
	return entries
}
