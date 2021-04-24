package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leep-frog/command"
)

const (
	regexpArg      = "REGEXP"
	replacementArg = "REPLACEMENT"
	fileArg        = "FILE"
)

type Replace struct {
	// Used for testing.
	baseDirectory string
}

func (*Replace) Load(jsn string) error { return nil }
func (*Replace) Changed() bool         { return false }
func (*Replace) Setup() []string       { return nil }
func (*Replace) Name() string {
	return "replace"
}
func (*Replace) Alias() string {
	return "r"
}

func (r *Replace) replace(output command.Output, rx *regexp.Regexp, rp, shortFile string) error {
	filename := shortFile
	if r.baseDirectory != "" {
		filename = filepath.Join(r.baseDirectory, filename)
	}
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %q does not exist", shortFile)
	} else if err != nil {
		return fmt.Errorf("unknown error when fetching file %q: %v", shortFile, err)
	}

	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		lines[i] = rx.ReplaceAllString(line, rp)
		if line != lines[i] {
			output.Stdout("Replacement made in %q:", shortFile)
			output.Stdout("  " + line)
			output.Stdout("  " + lines[i])
		}
	}

	op := strings.Join(lines, "\n")
	if err = ioutil.WriteFile(filename, []byte(op), fi.Mode()); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func (r *Replace) Replace(output command.Output, data *command.Data) error {
	rx, err := regexp.Compile(data.Values[regexpArg].String())
	if err != nil {
		return output.Stderr("invalid regex: %v", err)
	}
	rp := data.Values[replacementArg].String()
	filenames := data.Values[fileArg].StringList()

	for _, filename := range filenames {
		if err = r.replace(output, rx, rp, filename); err != nil {
			err = fmt.Errorf("error while processing %q: %v", filename, err)
			output.Err(err)
		}
	}
	return err
}

func (r *Replace) Node() *command.Node {
	ao := &command.ArgOpt{
		Completor: &command.Completor{
			SuggestionFetcher: &command.FileFetcher{},
		},
	}

	return command.SerialNodes(
		command.StringNode(regexpArg, nil),
		command.StringNode(replacementArg, nil),
		command.StringListNode(fileArg, 1, command.UnboundedList, ao),
		command.ExecutorNode(r.Replace),
	)
}
