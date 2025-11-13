package cmd

import "testing"

func TestPulseAndGraphHaveSortFlags(t *testing.T) {
	if pulseCmd.Flags().Lookup("sort") == nil {
		t.Fatal("pulse: missing --sort flag")
	}
	if pulseCmd.Flags().Lookup("direction") == nil {
		t.Fatal("pulse: missing --direction flag")
	}
	if pulseCmd.Flags().Lookup("order") == nil {
		t.Fatal("pulse: missing --order alias flag")
	}

	if graphCmd.Flags().Lookup("sort") == nil {
		t.Fatal("graph: missing --sort flag")
	}
	if graphCmd.Flags().Lookup("direction") == nil {
		t.Fatal("graph: missing --direction flag")
	}
	if graphCmd.Flags().Lookup("order") == nil {
		t.Fatal("graph: missing --order alias flag")
	}
}
