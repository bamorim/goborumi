package borumi

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func queryJSON(dbPath string, query string) ([]map[string]any, error) {
	cmd := exec.Command("sqlite3", "-readonly", "-json", dbPath, query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite query failed: %v (%s)", err, string(output))
	}

	var rows []map[string]any
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, fmt.Errorf("could not parse sqlite json output: %w", err)
	}
	return rows, nil
}

func execWrite(dbPath string, query string) error {
	cmd := exec.Command("sqlite3", dbPath, query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite write failed: %v (%s)", err, string(output))
	}
	return nil
}

func sqlQuote(value string) string {
	return "'" + replaceSingleQuotes(value) + "'"
}

func replaceSingleQuotes(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		out = append(out, r)
		if r == '\'' {
			out = append(out, '\'')
		}
	}
	return string(out)
}
