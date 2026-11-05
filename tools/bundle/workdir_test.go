package bundle

import "testing"

func TestRestoredDir(t *testing.T) {
	home := t.TempDir()
	for _, tc := range []struct {
		name, wd, pwd, want string
	}{
		{"bundle from a shell", "/Applications/Name.app/Contents/Resources", home, home},
		{"bundle from Finder", "/Applications/Name.app/Contents/Resources", "", ""},
		{"not a bundle", "/tmp/Resources", home, ""},
		{"already there", home, home, ""},
		{"shell folder gone", "/Applications/Name.app/Contents/Resources", home + "/missing", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := restoredDir(tc.wd, tc.pwd); got != tc.want {
				t.Errorf("restoredDir(%q, %q) = %q, want %q", tc.wd, tc.pwd, got, tc.want)
			}
		})
	}
}
