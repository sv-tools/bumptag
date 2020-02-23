// The bumptag creates a new tag to release a new version of your code.
//
// The tool finds the last git tag, increments it and create new tag with a changelog.
// https://github.com/sv-tools/bumptag/blob/master/README.md
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"

	"github.com/coreos/go-semver/semver"
)

var (
	version   = "0.0.0"
	tagPrefix = "v"
)

const (
	defaultRemote = "origin"
	defaultEditor = "vim"
)

func realGit(input string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	if len(input) > 0 {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		text := fmt.Sprintf(
			"command '%s' failed: %s",
			strings.Join(cmd.Args, " "),
			err.Error(),
		)
		errText := strings.TrimSpace(stderr.String())
		if len(errText) > 0 {
			text += "\n" + errText
		}
		return "", errors.New(text)
	}
	return strings.TrimSpace(stdout.String()), nil
}

var git = realGit

func noOutputGit(input string, arg ...string) error {
	_, err := git(input, arg...)
	return err
}

func disableGPG() (string, error) {
	output, _ := git("", "config", "--local", "--get", "log.showSignature")
	if err := noOutputGit("", "config", "--local", "log.showSignature", "false"); err != nil {
		return "", err
	}
	return output, nil
}

func restoreGPG(oldValue string) error {
	if len(oldValue) > 0 {
		return noOutputGit("", "config", "--local", "log.showSignature", oldValue)
	}
	return noOutputGit("", "config", "--local", "--unset", "log.showSignature")
}

func setUpGPG() (func(), error) {
	oldValue, err := disableGPG()
	if err != nil {
		return nil, err
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		_ = restoreGPG(oldValue)
		os.Exit(42)
	}()

	return func() {
		signal.Stop(signalChan)
		_ = restoreGPG(oldValue)
	}, nil
}

func gitConfig(name, defaultValue string) string {
	output, err := git("", "config", "--get", name)
	if err != nil {
		return defaultValue
	}
	return output
}

func gitConfigBool(name string, defaultValue bool) bool {
	output := gitConfig(name, strconv.FormatBool(defaultValue))
	value, err := strconv.ParseBool(output)
	if err != nil {
		return defaultValue
	}
	return value
}

func findTag() (*semver.Version, string, error) {
	output, err := git("", "tag")
	if err != nil {
		return nil, "", err
	}
	if output == "" {
		return &semver.Version{}, "", nil
	}
	output, err = git("", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return nil, "", err
	}
	currentTagName := output
	if !strings.HasPrefix(output, tagPrefix) {
		tagPrefix = ""
	}
	tag, err := semver.NewVersion(strings.TrimPrefix(output, tagPrefix))
	if err != nil {
		return nil, "", err
	}
	return tag, currentTagName, nil
}

func createTag(tagName, annotation string, sign bool) error {
	args := []string{"tag", "-F-"}
	if sign {
		args = append(args, "--sign")
	}
	args = append(args, tagName)
	return noOutputGit(annotation, args...)
}

func showTag(tagName string) (string, error) {
	return git("", "show", tagName)
}

func getChangeLog(tagName string) (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		output, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(output), nil
	}

	args := []string{"log", "--pretty=%h %s", "--no-merges"}
	if len(tagName) > 0 {
		args = append(args, tagName+"..HEAD")
	}
	output, err := git("", args...)
	if err != nil {
		return "", err
	}
	var res []string
	for _, line := range strings.Split(output, "\n") {
		res = append(res, "* "+line)
	}
	return strings.Join(res, "\n"), nil
}

func parseRemote(remote string) (string, error) {
	for _, part := range strings.Split(remote, " ") {
		if strings.HasPrefix(part, "[") {
			part = strings.Trim(part, "[]")
			names := strings.SplitN(part, "/", 2)
			if len(names) != 2 {
				return "", fmt.Errorf("cannot determine a remote name: %s", part)
			}
			return names[0], nil
		}
	}
	return "", fmt.Errorf("remote for the active branch '%s' not found", remote)
}

func getRemote() (string, error) {
	output, err := git("", "branch", "--list", "-vv")
	if err != nil {
		return "", err
	}
	for _, remote := range strings.Split(output, "\n") {
		remote = strings.TrimSpace(remote)
		if strings.HasPrefix(remote, "*") {
			return parseRemote(remote)
		}
	}
	return defaultRemote, nil
}

func pushTag(remote, tagName string) error {
	return noOutputGit("", "push", remote, tagName)
}

func makeAnnotation(changeLog string, tagName string) string {
	output := []string{
		"Bump version " + tagName,
		"",
		changeLog,
	}
	return strings.Join(output, "\n")
}

func createFlag(flagSet *flag.FlagSet, name, short string, value bool, usage string) *bool {
	p := flagSet.Bool(name, value, usage)
	if len(short) > 0 {
		flagSet.BoolVar(p, short, value, usage)
	}
	return p
}

type bumptagArgs struct {
	flagSet  *flag.FlagSet
	edit     *bool
	dryRun   *bool
	silent   *bool
	autoPush *bool
	major    *bool
	minor    *bool
	patch    *bool
	version  *bool
	findTag  *bool
}

func (f *bumptagArgs) usage() {
	output := `Usage: bumptag [<tagname>]

    <tagname>       The name of the tag to create, must be Semantic Versions 2.0.0 (http://semver.org)
    -e, --edit      Edit an annotation
    -r, --dry-run   Prints an annotation for the new tag
    -s, --silent    Do not show the created tag
    -a, --auto-push Push the created tag automatically
    -m, --major     Increment the MAJOR version
    -n, --minor     Increment the MINOR version (default)
    -p, --patch     Increment the PATCH version
        --version   Show a version of the bumptag tool
        --find-tag  Show the last tag, can be useful for CI tools

    The change log is automatically generated from git commits from the previous tag or can be passed by <stdin>.`
	fmt.Println(output)
}

func (f *bumptagArgs) parse() error {
	f.flagSet.Usage = f.usage
	return f.flagSet.Parse(os.Args[1:])
}

func newBumptagArgs() *bumptagArgs {
	flagSet := flag.NewFlagSet("Bumptag", flag.ExitOnError)
	return &bumptagArgs{
		flagSet:  flagSet,
		edit:     createFlag(flagSet, "edit", "e", false, "Edit an annotation"),
		dryRun:   createFlag(flagSet, "dry-run", "r", false, "Prints an annotation for the new tag"),
		silent:   createFlag(flagSet, "silent", "s", false, "Do not show the created tag"),
		autoPush: createFlag(flagSet, "auto-push", "a", false, "Push the created tag automatically"),
		major:    createFlag(flagSet, "major", "m", false, "Increment the MAJOR version"),
		minor:    createFlag(flagSet, "minor", "n", false, "Increment the MINOR version (default)"),
		patch:    createFlag(flagSet, "patch", "p", false, "Increment the PATCH version"),
		version:  createFlag(flagSet, "version", "", false, "Show a version of the bumptag tool"),
		findTag:  createFlag(flagSet, "find-tag", "", false, "Show the latest tag, can be useful for CI tools"),
	}
}

func setTag(flagSet *flag.FlagSet, tag *semver.Version, args *bumptagArgs) {
	if flagSet.NArg() > 0 {
		if err := tag.Set(strings.TrimPrefix(flagSet.Arg(0), tagPrefix)); err != nil {
			panic(err)
		}
	} else {
		switch true {
		case *args.major:
			tag.BumpMajor()
		case *args.minor:
			tag.BumpMinor()
		case *args.patch:
			tag.BumpPatch()
		default:
			tag.BumpMinor()
		}
	}
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func openEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = defaultEditor
	}
	executable, err := exec.LookPath(editor)
	if err != nil {
		return err
	}

	tty := os.Stdin
	stat, _ := tty.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		tty, err = os.Open("/dev/tty")
		if err != nil {
			return err
		}
		defer tty.Close()
	}

	cmd := exec.Command(executable, filename)
	cmd.Stdin = tty
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func edit(annotation string) (string, error) {
	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return "", err
	}
	filename := file.Name()
	defer os.Remove(filename)

	if _, err := file.WriteString(annotation); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	if err := openEditor(filename); err != nil {
		return "", err
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func main() {
	args := newBumptagArgs()
	panicIfError(args.parse())

	if *args.version {
		fmt.Print(version)
		return
	}

	tearDownGPG, err := setUpGPG()
	panicIfError(err)
	defer tearDownGPG()

	tag, currentTagName, err := findTag()
	panicIfError(err)

	if *args.findTag {
		fmt.Print(currentTagName)
		return
	}

	changeLog, err := getChangeLog(currentTagName)
	panicIfError(err)

	setTag(args.flagSet, tag, args)
	tagName := tagPrefix + tag.String()
	annotation := makeAnnotation(changeLog, tagName)

	if *args.edit {
		annotation, err = edit(annotation)
		panicIfError(err)
	}

	if *args.dryRun {
		fmt.Println(annotation)
		return
	}

	sign := gitConfigBool("commit.gpgsign", false)
	panicIfError(createTag(tagName, annotation, sign))

	if *args.autoPush {
		remote, err := getRemote()
		panicIfError(err)
		panicIfError(pushTag(remote, tagName))
		if !*args.silent {
			fmt.Printf(
				"The tag '%s' has been pushed to the remote '%s'",
				tagName,
				remote,
			)
		}
	}
	if !*args.silent {
		output, err := showTag(tagName)
		panicIfError(err)
		fmt.Println(output)
	}
}
