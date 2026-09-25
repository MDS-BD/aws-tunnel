package cloud

import "testing"

func TestSupportsMux(t *testing.T) {
	cases := map[string]bool{
		"3.3.1611.0": true,
		"3.0.196.0":  true,
		"3.0.195.9":  false,
		"2.3.1319.0": false,
		"":           false,
		"garbage":    false,
	}
	for agentVersion, expected := range cases {
		if got := (Instance{AgentVersion: agentVersion}).SupportsMux(); got != expected {
			t.Errorf("SupportsMux(%q) = %v, expected %v", agentVersion, got, expected)
		}
	}
}
