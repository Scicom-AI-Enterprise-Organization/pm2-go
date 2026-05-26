package cli

import (
	"reflect"
	"testing"
)

func TestReorderArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		bool []string
		want []string
	}{
		{
			name: "flag-before-positional unchanged",
			in:   []string{"--name", "foo", "script.sh"},
			want: []string{"--name", "foo", "script.sh"},
		},
		{
			name: "flag-after-positional moves to front",
			in:   []string{"script.sh", "--name", "foo"},
			want: []string{"--name", "foo", "script.sh"},
		},
		{
			name: "inline = value stays together",
			in:   []string{"script.sh", "--name=foo"},
			want: []string{"--name=foo", "script.sh"},
		},
		{
			name: "boolean flag does not eat next token",
			in:   []string{"--shell", "the command line"},
			bool: []string{"shell"},
			want: []string{"--shell", "the command line"},
		},
		{
			name: "non-boolean flag does eat next token",
			in:   []string{"--name", "foo", "script.sh"},
			want: []string{"--name", "foo", "script.sh"},
		},
		{
			name: "double-dash terminator preserves rest as positional",
			in:   []string{"--name", "foo", "--", "--name", "bar"},
			want: []string{"--name", "foo", "--", "--name", "bar"},
		},
		{
			name: "multiple flags reordered",
			in:   []string{"script.sh", "extra", "--name", "foo", "--shell"},
			bool: []string{"shell"},
			want: []string{"--name", "foo", "--shell", "script.sh", "extra"},
		},
		{
			name: "empty input",
			in:   []string{},
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reorderArgs(c.in, c.bool...)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %#v\nwant %#v", got, c.want)
			}
		})
	}
}

func TestShellSplit(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "node", []string{"node"}, false},
		{"two words", "node app.js", []string{"node", "app.js"}, false},
		{"double-quoted preserves spaces", `node "my app.js"`, []string{"node", "my app.js"}, false},
		{"single-quoted preserves spaces", `node 'my app.js'`, []string{"node", "my app.js"}, false},
		{"backslash escape", `node\ app.js`, []string{"node app.js"}, false},
		{"mixed flags", `node app.js --name "my app" --port 3000`, []string{"node", "app.js", "--name", "my app", "--port", "3000"}, false},
		{"backslash inside double-quote", `"a\"b"`, []string{`a"b`}, false},
		{"unterminated double", `node "app`, nil, true},
		{"unterminated single", `node 'app`, nil, true},
		{"empty token from quoted empty string", `node ""`, []string{"node", ""}, false},
		{"tab as separator", "node\tapp.js", []string{"node", "app.js"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := shellSplit(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if c.wantErr {
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %#v\nwant %#v", got, c.want)
			}
		})
	}
}

func TestKvFlag(t *testing.T) {
	k := kvFlag{}
	if err := k.Set("FOO=bar"); err != nil {
		t.Fatal(err)
	}
	if k["FOO"] != "bar" {
		t.Errorf("FOO=%q want bar", k["FOO"])
	}
	if err := k.Set("BAD"); err == nil {
		t.Error("missing = should fail")
	}
	if err := k.Set("FOO=a=b=c"); err != nil {
		t.Fatal(err)
	}
	if k["FOO"] != "a=b=c" {
		t.Errorf("FOO=%q want a=b=c", k["FOO"])
	}
}

func TestSliceFlag(t *testing.T) {
	var s sliceFlag
	_ = s.Set("a")
	_ = s.Set("b")
	if len(s) != 2 || s[0] != "a" || s[1] != "b" {
		t.Errorf("s=%v", s)
	}
}

func TestDeriveName(t *testing.T) {
	cases := []struct{ name, target, want string }{
		{"", "/tmp/foo.sh", "foo"},
		{"override", "/tmp/foo.sh", "override"},
		{"", "/tmp/foo", "foo"},
		{"", "./bar.baz.js", "bar-baz"}, // sanitize collapses dots
	}
	for _, c := range cases {
		got := deriveName(c.name, c.target)
		if got != c.want {
			t.Errorf("deriveName(%q,%q) = %q want %q", c.name, c.target, got, c.want)
		}
	}
}

func TestDeriveShellName(t *testing.T) {
	cases := []struct{ name, cmd, want string }{
		{"", "node app.js", "node"},
		{"foo", "node app.js", "foo"},
		{"", "/usr/bin/python3 -m http.server", "python3"},
		{"", "", "app"}, // fall-through
	}
	for _, c := range cases {
		got := deriveShellName(c.name, c.cmd)
		if got != c.want {
			t.Errorf("deriveShellName(%q,%q) = %q want %q", c.name, c.cmd, got, c.want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"abc":     "abc",
		"a-b_c":   "a-b_c",
		"a.b":     "a-b",
		"a/b":     "a-b",
		"":        "app",
		"foo bar": "foo-bar",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitize(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsEcosystem(t *testing.T) {
	cases := map[string]bool{
		"foo.json":              true,
		"foo.yaml":              true,
		"foo.yml":               true,
		"ecosystem.config.json": true,
		"server.js":             false,
		"foo.sh":                false,
		"foo":                   false,
	}
	for in, want := range cases {
		if got := isEcosystem(in); got != want {
			t.Errorf("isEcosystem(%q)=%v want %v", in, got, want)
		}
	}
}

func TestContainsSpace(t *testing.T) {
	if !containsSpace("a b") {
		t.Error("expected true for 'a b'")
	}
	if !containsSpace("a\tb") {
		t.Error("expected true for tab")
	}
	if containsSpace("abc") {
		t.Error("expected false for 'abc'")
	}
}
