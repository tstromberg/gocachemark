package output

import (
	"encoding/json"
	"os"
	"time"
)

// WriteJSON writes benchmark results to a JSON file.
func WriteJSON(filename string, results Results, commandLine string) error {
	results.Timestamp = time.Now().Format(time.RFC3339)
	results.MachineInfo.CommandLine = commandLine

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
