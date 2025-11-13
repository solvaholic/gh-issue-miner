package output

import (
	"encoding/json"
	"io"
)

// WriteFetchJSON writes fetch results as JSON: { repository: <repo>, issues: [...] }
func WriteFetchJSON(w io.Writer, repo string, issues interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{"repository": repo, "issues": issues})
}

// WritePulseJSON writes pulse metrics as JSON: { repository: <repo>, metrics: {...} }
func WritePulseJSON(w io.Writer, repo string, metrics interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{"repository": repo, "metrics": metrics})
}

// WriteGraphJSON writes arbitrary graph data (typically map[string][]edge) as JSON
func WriteGraphJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
