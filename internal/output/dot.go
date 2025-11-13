package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// WriteGraphDOT accepts an arbitrary serializable graph (map[string][]edge-like)
// It marshals into generic structures then emits a DOT digraph to w.
func WriteGraphDOT(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var m map[string][]map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	fmt.Fprintln(w, "digraph G {")
	for src, edges := range m {
		for _, e := range edges {
			destI, ok := e["dest"]
			if !ok {
				continue
			}
			dest, _ := destI.(string)
			var parts []string
			if s, ok := e["source"].(string); ok && s != "" {
				parts = append(parts, "source="+s)
			}
			if a, ok := e["actor"].(string); ok && a != "" {
				parts = append(parts, "actor="+a)
			}
			if act, ok := e["action"].(string); ok && act != "" {
				parts = append(parts, "action="+act)
			}
			if ts, ok := e["timestamp"].(string); ok && ts != "" {
				parts = append(parts, "at="+ts)
			}
			label := escapeLabel(strings.Join(parts, ", "))
			fmt.Fprintf(w, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", src, dest, label)
		}
	}
	fmt.Fprintln(w, "}")
	return nil
}
