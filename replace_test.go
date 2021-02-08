package replace

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
	}{
		{
			name: "handles empty string",
		},
		{
			name: "handles invalid json",
			json: "}}",
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &Replace{}
			if err := r.Load(test.json); err != nil {
				t.Fatalf("Load(%v) should return nil; got %v", test.json, err)
			}
		})
	}
}

func TestRecursiveGrep(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		files      map[string][]string
		wantResp   *commands.ExecutorResponse
		wantOK     bool
		wantStdout []string
		wantStderr []string
		wantFiles  map[string][]string
	}{
		{
			name: "requires regexp",
			wantStderr: []string{
				`no argument provided for "REGEXP"`,
			},
		},
		{
			name: "requires replacement",
			args: []string{
				"abc",
			},
			wantStderr: []string{
				`no argument provided for "REPLACEMENT"`,
			},
		},
		{
			name: "requires at least one file",
			args: []string{
				"abc",
				"ABC",
			},
			wantStderr: []string{
				`no argument provided for "FILE"`,
			},
		},
		{
			name: "requires valid regex",
			args: []string{
				"[a-1]",
				"ABC",
				"one.txt",
			},
			wantStderr: []string{
				"invalid regex: error parsing regexp: invalid character class range: `a-1`",
			},
		},
		{
			name: "fails if file does not exist",
			args: []string{
				"abc",
				"ABC",
				"one.txt",
			},
			wantStderr: []string{
				`error while processing "one.txt": file "one.txt" does not exist`,
			},
		},
		{
			name: "makes no replacements",
			args: []string{
				"abc",
				"ABC",
				"one.txt",
			},
			files: map[string][]string{
				"one.txt": {
					"",
				},
			},
			wantOK: true,
		},
		{
			name: "makes a replacement",
			args: []string{
				"abc",
				"ABC",
				"one.txt",
			},
			files: map[string][]string{
				"one.txt": {
					"123 abc DEF",
				},
			},
			wantFiles: map[string][]string{
				"one.txt": {
					"123 ABC DEF",
				},
			},
			wantOK: true,
			wantStdout: []string{
				`Replacement made in "one.txt":`,
				"  123 abc DEF",
				"  123 ABC DEF",
			},
		},
		{
			name: "makes a replacement in files with matches",
			args: []string{
				"T(.*)T",
				"T${1}T${1}T",
				"one.txt",
				"two.txt",
				"three.txt",
			},
			files: map[string][]string{
				"one.txt": {
					"ToT",
					"Too cool",
					"prefix text Thank you very much, Tony",
				},
				"two.txt": {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				"three.txt": {
					"  T x T ",
				},
			},
			wantFiles: map[string][]string{
				"one.txt": {
					"ToToT",
					"Too cool",
					"prefix text Thank you very much, Thank you very much, Tony",
				},
				"two.txt": {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				"three.txt": {
					"  T x T x T ",
				},
			},
			wantOK: true,
			wantStdout: []string{
				`Replacement made in "one.txt":`,
				"  ToT",
				"  ToToT",
				`Replacement made in "one.txt":`,
				"  prefix text Thank you very much, Tony",
				"  prefix text Thank you very much, Thank you very much, Tony",
				`Replacement made in "three.txt":`,
				"    T x T ",
				"    T x T x T ",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "clis-replace-test")
			if err != nil {
				t.Fatalf("failed to create temporary directory: %v", err)
			}

			for f, contents := range test.files {
				data := []byte(strings.Join(contents, "\n"))
				if err := ioutil.WriteFile(filepath.Join(dir, f), data, 0644); err != nil {
					t.Fatalf("failed to write to file %q: %v", f, err)
				}
			}

			tcos := &commands.TestCommandOS{}
			r := &Replace{
				baseDirectory: dir,
			}
			got, ok := commands.Execute(tcos, r.Command(), test.args, nil)
			if ok != test.wantOK {
				t.Fatalf("Replace: commands.Execute(%v) returned %v for ok; want %v", test.args, ok, test.wantOK)
			}
			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("Replace: Execute(%v) produced response diff (-want, +got):\n%s", test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, tcos.GetStdout()); diff != "" {
				t.Errorf("Replace: command.Execute(%v) produced stdout diff (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, tcos.GetStderr()); diff != "" {
				t.Errorf("Replace: command.Execute(%v) produced stderr diff (-want, +got):\n%s", test.args, diff)
			}

			if r.Changed() {
				t.Errorf("Replace: command.Execute(%v) set changed to true, but should be false", test.args)
			}

			for f, originalContents := range test.files {
				wantContents, ok := test.wantFiles[f]
				if !ok {
					wantContents = originalContents
				}

				gotBytes, err := ioutil.ReadFile(filepath.Join(dir, f))
				if err != nil {
					t.Fatalf("failed to fetch file contents: %v", err)
				}
				gotContents := strings.Split(string(gotBytes), "\n")

				if diff := cmp.Diff(wantContents, gotContents); diff != "" {
					t.Errorf("Replace: command.Execute(%v) produced file diff for %q (-want, +got):\n%s", test.args, f, diff)
				}
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	r := &Replace{}

	wantName := "replace"
	if got := r.Name(); got != wantName {
		t.Fatalf("Name() returned %q; want %q", got, wantName)
	}

	wantAlias := "r"
	if got := r.Alias(); got != wantAlias {
		t.Fatalf("Alias() returned %q; want %q", got, wantAlias)
	}
}
