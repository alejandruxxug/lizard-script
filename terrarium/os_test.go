package terrarium

import (
	"testing"
	"time"
)

func TestOS(t *testing.T) {
	t.Setenv("LIZARD_TEST_VALUE", "scales")
	Args = []string{"one", "two"}
	defer func() { Args = nil }()
	check(t, []testCase{
		{"args", `os.args()`, `["one", "two"]`, false},
		{"env", `os.env("LIZARD_TEST_VALUE")`, "scales", false},
		{"env not set", `os.env("LIZARD_TEST_NOT_SET_ANYWHERE")`, "null", false},
		{"platform", `os.platform()`, Platform(), false},
		{"locale is two letters", `len(os.locale())`, "2", false},
		{"config dir", `type(os.config_dir())`, "STRING", false},

		{"env a number", `os.env(1)`, "want a STRING name, got INTEGER", true},
		{"exit a string", `os.exit("1")`, "want an INTEGER code, got STRING", true},
		{"exit out of range", `os.exit(300)`, "the code must be from 0 to 255, got 300", true},
		{"args takes nothing", `os.args(1)`, "want 0, got 1", true},
	})
}

func TestOSExit(t *testing.T) {
	code := -1
	saved := Exit
	Exit = func(c int) { code = c }
	defer func() { Exit = saved }()
	run(t, `os.exit(3)`)
	if code != 3 {
		t.Fatalf("os.exit(3) exited with %d", code)
	}
	run(t, `os.exit()`)
	if code != 0 {
		t.Fatalf("os.exit() exited with %d", code)
	}
}

func TestTwoLetters(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"es_CO.UTF-8", "es"}, {"EN-us", "en"}, {"de", "de"},
		{"eng", ""}, {"C", ""}, {"C.UTF-8", ""}, {"", ""}, {"1a", ""},
	} {
		if got := twoLetters(tc.in); got != tc.want {
			t.Errorf("twoLetters(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTimeAndClip(t *testing.T) {
	check(t, []testCase{
		{"ms is a float", `type(time.ms())`, "FLOAT", false},
		{"ms goes forward", `let a = time.ms(); time.sleep(2); time.ms() - a >= 2`, "true", false},
		{"now", `keys(time.now())`, `["year", "month", "day", "hour", "minute", "second", "weekday", "epoch", "utc_offset"]`, false},
		{"now month", `let n = time.now(); n.month >= 1 && n.month <= 12 && n.year >= 2024`, "true", false},
		{"epoch is the clock", `let n = time.now(); type(n.epoch) == "FLOAT" && n.epoch > 1700000000`, "true", false},
		{"weekday", `let n = time.now(); n.weekday >= 1 && n.weekday <= 7`, "true", false},
		{"utc", `let u = time.utc(); [u.utc_offset, keys(u) == keys(time.now())]`, "[0, true]", false},
		{"utc and now agree", `let n = time.now(); let u = time.utc(); math.abs(n.epoch - u.epoch) < 5`, "true", false},
		{"sleep a float", `time.sleep(0.5)`, "null", false},

		{"sleep a string", `time.sleep("1")`, "want a number of milliseconds, got STRING", true},
		{"sleep negative", `time.sleep(-1)`, "the milliseconds must be from 0 to 3600000, got -1", true},
		{"now takes nothing", `time.now(1)`, "want 0, got 1", true},
		{"utc takes nothing", `time.utc(1)`, "want 0, got 1", true},
		{"clip.set a number", `clip.set(1)`, "want a STRING text, got INTEGER", true},
		{"clip.get takes nothing", `clip.get(1)`, "want 0, got 1", true},
	})
}

// TestDateHash checks the fields of time.now and time.utc for known times.
func TestDateHash(t *testing.T) {
	east := time.FixedZone("east", 5*3600+1800)
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"the epoch", time.Unix(0, 0).UTC(),
			`{"year": 1970, "month": 1, "day": 1, "hour": 0, "minute": 0, "second": 0, "weekday": 4, "epoch": 0.0, "utc_offset": 0}`},
		{"a Sunday ahead of UTC", time.Date(2026, 11, 1, 9, 30, 15, 250_000_000, east),
			`{"year": 2026, "month": 11, "day": 1, "hour": 9, "minute": 30, "second": 15, "weekday": 7, "epoch": 1793505615.25, "utc_offset": 19800}`},
	}
	for _, tt := range tests {
		if got := dateHash(tt.at).Inspect(); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}
