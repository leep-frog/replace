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

var (
	regexpArg      = command.StringNode("REGEXP", "Expression to replace")
	replacementArg = command.StringNode("REPLACEMENT", "Replacement pattern")
	fileArg        = command.FileListNode("FILE", "File in which replacements should be made", 1, command.UnboundedList)
)

func CLI() *Replace {
	return &Replace{}
}

type Replace struct {
	// Used for testing.
	baseDirectory string
}

func (*Replace) Load(jsn string) error { return nil }
func (*Replace) Changed() bool         { return false }
func (*Replace) Setup() []string       { return nil }
func (*Replace) Name() string {
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
			output.Stdoutf("Replacement made in %q:", shortFile)
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
	rx, err := regexp.Compile(data.String(regexpArg.Name()))
	if err != nil {
		return output.Stderrf("invalid regex: %v", err)
	}
	rp := data.String(replacementArg.Name())
	filenames := data.StringList(fileArg.Name())

	for _, filename := range filenames {
		if err = r.replace(output, rx, rp, filename); err != nil {
			err = fmt.Errorf("error while processing %q: %v", filename, err)
			output.Err(err)
		}
	}
	return err
}

func (r *Replace) Node() *command.Node {
	return command.SerialNodes(
		command.Description("Makes regex replacements in files"),
		regexpArg,
		replacementArg,
		fileArg,
		command.ExecutorNode(r.Replace),
	)
}
