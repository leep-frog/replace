package replace

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
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
		wantResp   *command.ExecuteData
		wantErr    error
		wantData   *command.Data
		wantStdout []string
		wantStderr []string
		wantFiles  map[string][]string
	}{
		{
			name:       "requires regexp",
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg: command.StringValue(""),
				},
			},
		},
		{
			name: "requires replacement",
			args: []string{
				"abc",
			},
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue(""),
				},
			},
		},
		{
			name: "requires at least one file",
			args: []string{
				"abc",
				"ABC",
			},
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue(),
				},
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
			wantErr: fmt.Errorf("invalid regex: error parsing regexp: invalid character class range: `a-1`"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("[a-1]"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
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
			wantErr: fmt.Errorf(`error while processing "one.txt": file "one.txt" does not exist`),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
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
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
			},
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
			wantStdout: []string{
				`Replacement made in "one.txt":`,
				"  123 abc DEF",
				"  123 ABC DEF",
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
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
			wantData: &command.Data{
				Values: map[string]*command.Value{
					regexpArg:      command.StringValue("T(.*)T"),
					replacementArg: command.StringValue("T${1}T${1}T"),
					fileArg:        command.StringListValue("one.txt", "two.txt", "three.txt"),
				},
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

			r := &Replace{
				baseDirectory: dir,
			}
			command.ExecuteTest(t, r.Node(), test.args, test.wantErr, test.wantResp, test.wantData, test.wantStdout, test.wantStderr)
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
