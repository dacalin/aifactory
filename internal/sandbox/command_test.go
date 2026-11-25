package sandbox

import (
	"slices"
	"strings"
	"testing"
)

func TestSandboxLaunchersAllowEachExplicitDirectory(t *testing.T) {
	dirs := []string{"/project/rules", "/project/reports with spaces"}
	mac := macSandboxArgs(dirs)
	wantParameters := []string{"-D", "WRITE_DIR_0=" + dirs[0], "-D", "WRITE_DIR_1=" + dirs[1], "-p"}
	if len(mac) != 6 || !slices.Equal(mac[:5], wantParameters) {
		t.Fatalf("unexpected sandbox profile parameters: %v", mac)
	}
	for _, fragment := range []string{"(deny file-write*)", `(subpath (param "WRITE_DIR_0"))`, `(subpath (param "WRITE_DIR_1"))`} {
		if !strings.Contains(mac[5], fragment) {
			t.Fatalf("profile missing %q: %s", fragment, mac[5])
		}
	}
	linux := linuxSandboxArgs(dirs)
	wantBinds := []string{"--bind", dirs[0], dirs[0], "--bind", dirs[1], dirs[1], "--"}
	if !slices.Equal(linux[len(linux)-len(wantBinds):], wantBinds) {
		t.Fatalf("unexpected writable bind mounts: %v", linux)
	}
}
