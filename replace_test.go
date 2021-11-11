package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

const (
	testDir = "testing"
)

func td(fs string) string {
	return filepath.Join(testDir, fs)
}

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

func TestReplace(t *testing.T) {
	for _, test := range []struct {
		name      string
		etc       *command.ExecuteTestCase
		files     map[string][]string
		wantFiles map[string][]string
	}{
		{
			name: "requires regexp",
			etc: &command.ExecuteTestCase{
				WantStderr: []string{`Argument "REGEXP" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "REGEXP" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "requires replacement",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
				},
				WantStderr: []string{`Argument "REPLACEMENT" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "REPLACEMENT" requires at least 1 argument, got 0`),
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name(): command.StringValue("abc"),
				}},
			},
		},
		{
			name: "requires at least one file",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
				},
				WantStderr: []string{`Argument "FILE" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name():      command.StringValue("abc"),
					replacementArg.Name(): command.StringValue("ABC"),
				}},
			},
		},
		{
			name: "requires valid regex",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"[a-1]",
					"ABC",
					td("one.txt"),
				},
				WantStderr: []string{
					"validation failed: [IsRegex] value isn't a valid regex: error parsing regexp: invalid character class range: `a-1`",
				},
				WantErr: fmt.Errorf("validation failed: [IsRegex] value isn't a valid regex: error parsing regexp: invalid character class range: `a-1`"),
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name(): command.StringValue("[a-1]"),
				}},
			},
		},
		{
			name: "fails if file does not exist",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td("one.txt"),
				},
				WantStderr: []string{
					fmt.Sprintf(`validation failed: [AreFiles] file %q does not exist`, td("one.txt")),
				},
				WantErr: fmt.Errorf(`validation failed: [AreFiles] file %q does not exist`, td("one.txt")),
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name():      command.StringValue("abc"),
					replacementArg.Name(): command.StringValue("ABC"),
					fileArg.Name():        command.StringListValue(td("one.txt")),
				}},
			},
		},
		{
			name: "makes no replacements",
			files: map[string][]string{
				td("one.txt"): {
					"",
				},
			},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td("one.txt"),
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name():      command.StringValue("abc"),
					replacementArg.Name(): command.StringValue("ABC"),
					fileArg.Name():        command.StringListValue(td("one.txt")),
				}},
			},
		},
		{
			name: "makes a replacement",
			files: map[string][]string{
				td("one.txt"): {
					"123 abc DEF",
				},
			},
			wantFiles: map[string][]string{
				td("one.txt"): {
					"123 ABC DEF",
				},
			},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td("one.txt"),
				},
				WantStdout: []string{
					fmt.Sprintf(`Replacement made in %q:`, td("one.txt")),
					"  123 abc DEF",
					"  123 ABC DEF",
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name():      command.StringValue("abc"),
					replacementArg.Name(): command.StringValue("ABC"),
					fileArg.Name():        command.StringListValue(td("one.txt")),
				}},
			},
		},
		{
			name: "makes a replacement in files with matches",
			files: map[string][]string{
				td("one.txt"): {
					"ToT",
					"Too cool",
					"prefix text Thank you very much, Tony",
				},
				td("two.txt"): {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				td("three.txt"): {
					"  T x T ",
				},
			},
			wantFiles: map[string][]string{
				td("one.txt"): {
					"ToToT",
					"Too cool",
					"prefix text Thank you very much, Thank you very much, Tony",
				},
				td("two.txt"): {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				td("three.txt"): {
					"  T x T x T ",
				},
			},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"T(.*)T",
					"T${1}T${1}T",
					td("one.txt"),
					td("two.txt"),
					td("three.txt"),
				},
				WantStdout: []string{
					fmt.Sprintf(`Replacement made in %q:`, td("one.txt")),
					"  ToT",
					"  ToToT",
					fmt.Sprintf(`Replacement made in %q:`, td("one.txt")),
					"  prefix text Thank you very much, Tony",
					"  prefix text Thank you very much, Thank you very much, Tony",
					fmt.Sprintf(`Replacement made in %q:`, td("three.txt")),
					"    T x T ",
					"    T x T x T ",
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					regexpArg.Name():      command.StringValue("T(.*)T"),
					replacementArg.Name(): command.StringValue("T${1}T${1}T"),
					fileArg.Name():        command.StringListValue(td("one.txt"), td("two.txt"), td("three.txt")),
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.Mkdir(testDir, 0644); err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			for f, contents := range test.files {
				data := []byte(strings.Join(contents, "\n"))
				if err := ioutil.WriteFile(f, data, 0644); err != nil {
					t.Fatalf("failed to write to file %q: %v", f, err)
				}
			}

			r := &Replace{}
			test.etc.Node = r.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, r)

			for f, originalContents := range test.files {
				wantContents, ok := test.wantFiles[f]
				if !ok {
					wantContents = originalContents
				}

				gotBytes, err := ioutil.ReadFile(f)
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

func TestUsage(t *testing.T) {
	command.UsageTest(t, &command.UsageTestCase{
		Node: CLI().Node(),
		WantString: []string{
			"Makes regex replacements in files",
			"REGEXP REPLACEMENT FILE [ FILE ... ]",
			"",
			"Arguments:",
			"  FILE: File in which replacements should be made",
			"  REGEXP: Expression to replace",
			"  REPLACEMENT: Replacement pattern",
		},
	})
}
