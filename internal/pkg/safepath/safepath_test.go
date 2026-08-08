package safepath

import "testing"

func TestResolve_Jail(t *testing.T) {
  jail := "/home/admin"
  tests := []struct{ in, want string; ok bool }{
    {"/etc/passwd", "/home/admin/etc/passwd", true},
    {"../etc/passwd", "", false},
    {"/home/admin/public_html", "/home/admin/public_html", true},
    {"public_html/test", "/home/admin/public_html/test", true},
    {"", "/home/admin", true},
  }
  for _, tc := range tests {
    got, err := Resolve(jail, tc.in)
    if tc.ok && err != nil { t.Errorf("Resolve %q failed: %v", tc.in, err) }
    if !tc.ok && err == nil { t.Errorf("Resolve %q should fail", tc.in) }
    if tc.ok && got != tc.want { t.Errorf("Resolve %q = %q want %q", tc.in, got, tc.want) }
  }
}
func TestIsWithin(t *testing.T) {
  if !isWithin("/home/admin/a", "/home/admin") { t.Error("within failed") }
  if isWithin("/home/admin2/a", "/home/admin") { t.Error("should not be within") }
}
