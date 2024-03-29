package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	regexpArg      = commander.Arg[string]("REGEXP", "Expression to replace", commander.IsRegex())
	replacementArg = commander.Arg[string]("REPLACEMENT", "Replacement pattern")
	fileArg        = commander.FileListArgument("FILE", "File(s) in which replacements should be made", 1, command.UnboundedList, commander.ListifyValidatorOption(commander.FileExists()))
	wholeFile      = commander.BoolFlag("whole-file", 'w', "Whether or not to replace multi-line regexes")
)

func CLI() *Replace {
	return &Replace{}
}

type Replace struct{}

func (*Replace) Changed() bool   { return false }
func (*Replace) Setup() []string { return nil }
func (*Replace) Name() string {
	return "r"
}

func (r *Replace) replace(output command.Output, rx *regexp.Regexp, rp, filename string) error {
	fi, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("unknown error when fetching file %q: %v", filename, err)
	}

	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		lines[i] = rx.ReplaceAllString(line, rp)
		if line != lines[i] {
			output.Stdoutf("Replacement made in %q:\n", filename)
			output.Stdoutf("  %s\n", line)
			output.Stdoutf("  %s\n", lines[i])
		}
	}

	op := strings.Join(lines, "\n")
	if err = ioutil.WriteFile(filename, []byte(op), fi.Mode()); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func (r *Replace) Replace(output command.Output, data *command.Data) error {
	rx := regexp.MustCompile(regexpArg.Get(data))
	rp := data.String(replacementArg.Name())
	filenames := data.StringList(fileArg.Name())

	var err error
	for _, filename := range filenames {
		if err = r.replace(output, rx, rp, filename); err != nil {
			err = fmt.Errorf("error while processing %q: %v", filename, err)
			output.Err(err)
		}
	}
	return err
}

func (r *Replace) Node() command.Node {
	return commander.SerialNodes(
		commander.Description("Makes regex replacements in files"),
		commander.FlagProcessor(wholeFile),
		regexpArg,
		replacementArg,
		fileArg,
		&commander.ExecutorProcessor{F: r.Replace},
	)
}
