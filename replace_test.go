package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

const (
	testDir = "testing"
)

func td(t *testing.T, fs ...string) string {
	return commandtest.FilepathAbs(t, append([]string{testDir}, fs...)...)
}

func TestReplace(t *testing.T) {
	for _, test := range []struct {
		name      string
		etc       *commandtest.ExecuteTestCase
		files     map[string][]string
		wantFiles map[string][]string
	}{
		{
			name: "requires regexp",
			etc: &commandtest.ExecuteTestCase{
				WantStderr: "Argument \"REGEXP\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "REGEXP" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "requires replacement",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"abc",
				},
				WantStderr: "Argument \"REPLACEMENT\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "REPLACEMENT" requires at least 1 argument, got 0`),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name(): "abc",
				}},
			},
		},
		{
			name: "requires at least one file",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
				},
				WantStderr: "Argument \"FILE\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name():      "abc",
					replacementArg.Name(): "ABC",
				}},
			},
		},
		{
			name: "requires valid regex",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"[a-1]",
					"ABC",
					td(t, "one.txt"),
				},
				WantStderr: "validation for \"REGEXP\" failed: [IsRegex] value \"[a-1]\" isn't a valid regex: error parsing regexp: invalid character class range: `a-1`\n",
				WantErr:    fmt.Errorf("validation for \"REGEXP\" failed: [IsRegex] value \"[a-1]\" isn't a valid regex: error parsing regexp: invalid character class range: `a-1`"),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name(): "[a-1]",
				}},
			},
		},
		{
			name: "fails if file does not exist",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td(t, "one.txt"),
				},
				WantStderr: fmt.Sprintf("validation for \"FILE\" failed: [FileExists] file %q does not exist\n", td(t, "one.txt")),
				WantErr:    fmt.Errorf(`validation for "FILE" failed: [FileExists] file %q does not exist`, td(t, "one.txt")),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name():      "abc",
					replacementArg.Name(): "ABC",
					fileArg.Name():        []string{td(t, "one.txt")},
				}},
			},
		},
		{
			name: "makes no replacements",
			files: map[string][]string{
				td(t, "one.txt"): {
					"",
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td(t, "one.txt"),
				},
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name():      "abc",
					replacementArg.Name(): "ABC",
					fileArg.Name():        []string{td(t, "one.txt")},
				}},
			},
		},
		{
			name: "makes a replacement",
			files: map[string][]string{
				td(t, "one.txt"): {
					"123 abc DEF",
				},
			},
			wantFiles: map[string][]string{
				td(t, "one.txt"): {
					"123 ABC DEF",
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"abc",
					"ABC",
					td(t, "one.txt"),
				},
				WantStdout: strings.Join([]string{
					fmt.Sprintf(`Replacement made in %q:`, td(t, "one.txt")),
					"  123 abc DEF",
					"  123 ABC DEF",
					"",
				}, "\n"),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name():      "abc",
					replacementArg.Name(): "ABC",
					fileArg.Name():        []string{td(t, "one.txt")},
				}},
			},
		},
		{
			name: "makes a replacement in files with matches",
			files: map[string][]string{
				td(t, "one.txt"): {
					"ToT",
					"Too cool",
					"prefix text Thank you very much, Tony",
				},
				td(t, "two.txt"): {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				td(t, "three.txt"): {
					"  T x T ",
				},
			},
			wantFiles: map[string][]string{
				td(t, "one.txt"): {
					"ToToT",
					"Too cool",
					"prefix text Thank you very much, Thank you very much, Tony",
				},
				td(t, "two.txt"): {
					"nothing to see here",
					"these are not the lines you are looking for",
				},
				td(t, "three.txt"): {
					"  T x T x T ",
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{
					"T(.*)T",
					"T${1}T${1}T",
					td(t, "one.txt"),
					td(t, "two.txt"),
					td(t, "three.txt"),
				},
				WantStdout: strings.Join([]string{
					fmt.Sprintf(`Replacement made in %q:`, td(t, "one.txt")),
					"  ToT",
					"  ToToT",
					fmt.Sprintf(`Replacement made in %q:`, td(t, "one.txt")),
					"  prefix text Thank you very much, Tony",
					"  prefix text Thank you very much, Thank you very much, Tony",
					fmt.Sprintf(`Replacement made in %q:`, td(t, "three.txt")),
					"    T x T ",
					"    T x T x T ",
					"",
				}, "\n"),
				WantData: &command.Data{Values: map[string]interface{}{
					regexpArg.Name():      "T(.*)T",
					replacementArg.Name(): "T${1}T${1}T",
					fileArg.Name():        []string{td(t, "one.txt"), td(t, "two.txt"), td(t, "three.txt")},
				}},
			},
		},
		/* Useful for commenting out tests. */
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
			commandertest.ExecuteTest(t, test.etc)
			commandertest.ChangeTest(t, nil, r)

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
	commandertest.ExecuteTest(t, &commandtest.ExecuteTestCase{
		Node: CLI().Node(),
		Args: []string{"--help"},
		WantStdout: strings.Join([]string{
			"Makes regex replacements in files",
			"REGEXP REPLACEMENT FILE [ FILE ... ] --whole-file|-w",
			"",
			"Arguments:",
			"  FILE: File(s) in which replacements should be made",
			"    FileExists()",
			"  REGEXP: Expression to replace",
			"    IsRegex()",
			"  REPLACEMENT: Replacement pattern",
			"",
			"Flags:",
			"  [w] whole-file: Whether or not to replace multi-line regexes",
			"",
		}, "\n"),
	})
}
