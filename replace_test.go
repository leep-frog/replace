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

func TestRecursive(t *testing.T) {
	for _, test := range []struct {
		name      string
		etc       *command.ExecuteTestCase
		files     map[string][]string
		wantFiles map[string][]string
	}{
		{
			name: "requires regexp",
			etc: &command.ExecuteTestCase{
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
			},
		},
		{
			name: "requires replacement",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
				},
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantData: &command.Data{
					regexpArg: command.StringValue("abc"),
				},
			},
		},
		{
			name: "requires at least one file",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
				},
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantData: &command.Data{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
				},
			},
		},
		{
			name: "requires valid regex",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"[a-1]",
					"ABC",
					"one.txt",
				},
				WantStderr: []string{
					"invalid regex: error parsing regexp: invalid character class range: `a-1`",
				},
				WantErr: fmt.Errorf("invalid regex: error parsing regexp: invalid character class range: `a-1`"),
				WantData: &command.Data{
					regexpArg:      command.StringValue("[a-1]"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
			},
		},
		{
			name: "fails if file does not exist",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					"one.txt",
				},
				WantStderr: []string{
					`error while processing "one.txt": file "one.txt" does not exist`,
				},
				WantErr: fmt.Errorf(`error while processing "one.txt": file "one.txt" does not exist`),
				WantData: &command.Data{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
			},
		},
		{
			name: "makes no replacements",
			files: map[string][]string{
				"one.txt": {
					"",
				},
			},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					"one.txt",
				},
				WantData: &command.Data{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
			},
		},
		{
			name: "makes a replacement",
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
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					"one.txt",
				},
				WantStdout: []string{
					`Replacement made in "one.txt":`,
					"  123 abc DEF",
					"  123 ABC DEF",
				},
				WantData: &command.Data{
					regexpArg:      command.StringValue("abc"),
					replacementArg: command.StringValue("ABC"),
					fileArg:        command.StringListValue("one.txt"),
				},
			},
		},
		{
			name: "makes a replacement in files with matches",
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
			etc: &command.ExecuteTestCase{
				Args: []string{
					"T(.*)T",
					"T${1}T${1}T",
					"one.txt",
					"two.txt",
					"three.txt",
				},
				WantStdout: []string{
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
				WantData: &command.Data{
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
			test.etc.Node = r.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, r)

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
					t.Errorf("Replace: command.Execute(%v) produced file diff for %q (-want, +got):\n%s", test.etc.Args, f, diff)
				}
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	r := &Replace{}

	wantName := "r"
	if got := r.Name(); got != wantName {
		t.Fatalf("Name() returned %q; want %q", got, wantName)
	}
}
