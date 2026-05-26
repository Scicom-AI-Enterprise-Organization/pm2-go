package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

func TestReadDotEnv(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, ".env")
	body := `# comment
FOO=bar
BAZ="quoted value"
QUUX='single quoted'
EMPTY=
NO_EQUAL_LINE
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readDotEnv(p)
	if got["FOO"] != "bar" {
		t.Errorf("FOO=%q", got["FOO"])
	}
	if got["BAZ"] != "quoted value" {
		t.Errorf("BAZ=%q", got["BAZ"])
	}
	if got["QUUX"] != "single quoted" {
		t.Errorf("QUUX=%q", got["QUUX"])
	}
	if _, ok := got["NO_EQUAL_LINE"]; ok {
		t.Error("malformed line should be skipped")
	}
}

func TestReadDotEnvMissingFile(t *testing.T) {
	got := readDotEnv("/no/such/path.env")
	if len(got) != 0 {
		t.Error("missing file should return empty map")
	}
}

func TestBuildEnvIncludesInstanceVars(t *testing.T) {
	spec := &process.Spec{
		Name: "x",
		Env:  map[string]string{"INLINE": "yes"},
	}
	env := buildEnv(spec, 2)
	joined := strings.Join(env, "\n")
	required := []string{
		"INLINE=yes",
		"PM2_GO_INSTANCE=2",
		"PM2_GO_APP_NAME=x",
		"NODE_APP_INSTANCE=2",
		"PM2_INSTANCE_ID=2",
	}
	for _, want := range required {
		if !strings.Contains(joined, want) {
			t.Errorf("env missing %q", want)
		}
	}
}

func TestBuildEnvFileBeforeInline(t *testing.T) {
	d := t.TempDir()
	envFile := filepath.Join(d, ".env")
	_ = os.WriteFile(envFile, []byte("KEY=fromfile\n"), 0o644)
	spec := &process.Spec{
		Name:     "x",
		EnvFiles: []string{envFile},
		Env:      map[string]string{"KEY": "inline"},
	}
	env := buildEnv(spec, 0)
	// Walk through env in order; later entries win when consumed by exec.
	// We expect KEY=inline to appear AFTER KEY=fromfile.
	idxFile := -1
	idxInline := -1
	for i, e := range env {
		if e == "KEY=fromfile" {
			idxFile = i
		}
		if e == "KEY=inline" {
			idxInline = i
		}
	}
	if idxFile < 0 || idxInline < 0 {
		t.Fatalf("missing entries: file=%d inline=%d env=%v", idxFile, idxInline, env)
	}
	if idxInline < idxFile {
		t.Errorf("inline env should win (appear later); got file=%d inline=%d", idxFile, idxInline)
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 42: "42", -7: "-7", 1000: "1000"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d)=%q want %q", in, got, want)
		}
	}
}
