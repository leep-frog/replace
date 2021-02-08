package replace

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/leep-frog/commands/commands"
)

const (
	regexpArg      = "REGEXP"
	replacementArg = "REPLACEMENT"
	fileArg        = "FILE"
)

var (
	osStat = os.Stat
)

type Replace struct{}

func (*Replace) Load(jsn string) error    { return nil }
func (*Replace) Changed() bool            { return false }
func (*Replace) Option() *commands.Option { return nil }
func (*Replace) Name() string {
	return "replace"
}
func (*Replace) Alias() string {
	return "r"
}

func (*Replace) replace(cos commands.CommandOS, rx *regexp.Regexp, rp, filename string) error {
	fi, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to check file: %v", err)
	}
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		lines[i] = rx.ReplaceAllString(line, rp)
		if line != lines[i] {
			cos.Stdout("Replacement made:")
			cos.Stdout("  " + line)
			cos.Stdout("  " + lines[i])
		}
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile("myfile", []byte(output), fi.Mode())
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func (r *Replace) Replace(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo) (*commands.ExecutorResponse, bool) {
	rx, err := regexp.Compile(*args[regexpArg].String())
	if err != nil {
		cos.Stderr("invalid regex: %v", err)
		return nil, false
	}
	rp := *args[replacementArg].String()
	filenames := *args[fileArg].StringList()

	ok := true
	for _, filename := range filenames {
		if err := r.replace(cos, rx, rp, filename); err != nil {
			cos.Stdout("%s:", filename)
			cos.Stderr("error while processing %q", filename)
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
			commands.StringArg(regexpArg, false, nil),
			commands.StringArg(replacementArg, false, nil),
			commands.StringListArg(fileArg, 1, commands.UnboundedList, cmp),
		},
	}
}
