package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n\nENVTEST_A=plain\nexport ENVTEST_B=\"quoted value\"\nENVTEST_C='single'\nENVTEST_SET=from-file\nnot a pair\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENVTEST_SET", "from-env")
	for _, k := range []string{"ENVTEST_A", "ENVTEST_B", "ENVTEST_C"} {
		os.Unsetenv(k)
		t.Cleanup(func() { os.Unsetenv(k) })
	}
	if err := Load(p); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"ENVTEST_A": "plain", "ENVTEST_B": "quoted value", "ENVTEST_C": "single", "ENVTEST_SET": "from-env"}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "absent")); err != nil {
		t.Fatalf("missing file should be ignored, got %v", err)
	}
}
