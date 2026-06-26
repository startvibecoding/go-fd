package format

import "testing"

func TestParseAndGenerate(t *testing.T) {
	tmpl := Parse("{path={} basename={/} parent={//} noExt={.} basenameNoExt={/.}}")
	if !tmpl.HasTokens() {
		t.Fatal("expected tokens")
	}
	got := tmpl.Generate("a/folder/file.txt", "/")
	want := "{path=a/folder/file.txt basename=file.txt parent=a/folder noExt=a/folder/file basenameNoExt=file}"
	if got != want {
		t.Errorf("Generate =\n %q\nwant\n %q", got, want)
	}
}

func TestNoPlaceholders(t *testing.T) {
	tmpl := Parse("just text")
	if tmpl.HasTokens() {
		t.Error("should not have tokens")
	}
	if got := tmpl.Generate("x", ""); got != "just text" {
		t.Errorf("got %q", got)
	}
}

func TestBraceEscapes(t *testing.T) {
	tmpl := Parse("{{ and }}")
	if got := tmpl.Generate("x", ""); got != "{ and }" {
		t.Errorf("got %q", got)
	}
}

func TestCustomSeparator(t *testing.T) {
	tmpl := PlaceholderTemplate()
	if got := tmpl.Generate("/foo/bar/baz", "#"); got != "#foo#bar#baz" {
		t.Errorf("got %q", got)
	}
	if got := tmpl.Generate(`C:\foo\bar`, "#"); got != "C:#foo#bar" {
		t.Errorf("got %q", got)
	}
}
