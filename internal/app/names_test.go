package app

import "testing"

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"BLE unlock retry\n":                "BLE unlock retry",
		"\n\"Dark mode toggle.\"":           "Dark mode toggle",
		"Name: **Fix login redirect loop**": "Fix login redirect loop",
		"":                                  "",
		"fake claude: Name this coding task in 2 to 5 words, so it is easy": "",
	} {
		if got := cleanName(in); got != want {
			t.Errorf("cleanName(%q) = %q, want %q", in, got, want)
		}
	}
}
