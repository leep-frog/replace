package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leep-frog/commands/commands"
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

func (*Replace) Load(jsn string) error    { return nil }
func (*Replace) Changed() bool            { return false }
func (*Replace) Option() *commands.Option { return nil }
func (*Replace) Name() string {
	return "replace"
}
func (*Replace) Alias() string {
	return "r"
}

func (r *Replace) replace(cos commands.CommandOS, rx *regexp.Regexp, rp, shortFile string) error {
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
			cos.Stdout("Replacement made in %q:", shortFile)
			cos.Stdout("  " + line)
			cos.Stdout("  " + lines[i])
		}
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filename, []byte(output), fi.Mode())
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func (r *Replace) Replace(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo) (*commands.ExecutorResponse, bool) {
	rx, err := regexp.Compile(args[regexpArg].String())
	if err != nil {
		cos.Stderr("invalid regex: %v", err)
		return nil, false
	}
	rp := args[replacementArg].String()
	filenames := args[fileArg].StringList()

	ok := true
	for _, filename := range filenames {
		if err := r.replace(cos, rx, rp, filename); err != nil {
			cos.Stderr("error while processing %q: %v", filename, err)
			ok = false
		}
	}

	return nil, ok
}

func (r *Replace) Command() commands.Command {
	cmp := &commands.Completor{
		SuggestionFetcher: &commands.FileFetcher{},
	}

	return &commands.TerminusCommand{
		Executor: r.Replace,
		Args: []commands.Arg{
			commands.StringArg(regexpArg, true, nil),
			commands.StringArg(replacementArg, true, nil),
			commands.StringListArg(fileArg, 1, commands.UnboundedList, cmp),
		},
	}
}
