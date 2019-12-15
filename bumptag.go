// The bumptag creates a new tag to release a new version of your code.
//
// The tool finds the last git tag, increments it and create new tag with a changelog.
// https://github.com/SVilgelm/bumptag/blob/master/README.md
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
	"regexp"
	"strconv"
	"strings"

	"github.com/coreos/go-semver/semver"
)

var version = "0.0.0"

const (
	tagPrefix     = "v"
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
	output, _ := git("", cConfig, cLocal, cGet, cLogShowSignature)
	if err := noOutputGit("", cConfig, cLocal, cLogShowSignature, "false"); err != nil {
		return "", err
	}
	return output, nil
}

func restoreGPG(oldValue string) error {
	if len(oldValue) > 0 {
		return noOutputGit("", cConfig, cLocal, cLogShowSignature, oldValue)
	}
	return noOutputGit("", cConfig, cLocal, cUnset, cLogShowSignature)
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
	output, err := git("", cConfig, cGet, name)
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
	currentTag := &semver.Version{}
	currentTagName := ""
	output, err := git("", cLog, "--pretty=%D")
	if err != nil {
		return nil, "", err
	}
	for _, line := range strings.Split(output, "\n") {
		for _, ref := range strings.Split(line, ",") {
			ref = strings.TrimSpace(ref)
			if strings.HasPrefix(ref, "tag:") {
				rawTag := strings.TrimPrefix(ref, "tag:")
				rawTag = strings.TrimSpace(rawTag)
				tag, err := semver.NewVersion(strings.TrimPrefix(rawTag, tagPrefix))
				if err != nil {
					continue
				}
				if currentTag.LessThan(*tag) {
					currentTag = tag
					currentTagName = rawTag
				}
			}
		}
	}
	return currentTag, currentTagName, nil
}

func createTag(tagName, annotation string, sign bool) error {
	args := []string{cTag, "-F-"}
	if sign {
		args = append(args, "--sign")
	}
	args = append(args, tagName)
	return noOutputGit(annotation, args...)
}

func showTag(tagName string) (string, error) {
	return git("", cShow, tagName)
}

func getChangeLog(tagName string) ([]string, error) {
	args := []string{cLog, "--pretty=%h %s", "--no-merges"}
	if len(tagName) > 0 {
		args = append(args, tagName+"..HEAD")
	}
	output, err := git("", args...)
	if err != nil {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
}

var branchRE = regexp.MustCompile(`^\* .+ \[(.+)/.+\]`)

func getRemote() (string, error) {
	output, err := git("", cBranch, "--list", "-vv")
	if err != nil {
		return "", err
	}
	for _, branch := range strings.Split(output, "\n") {
		matches := branchRE.FindStringSubmatch(branch)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", errors.New("remote for current branch not found")
}

func pushTag(remote, tagName string) error {
	return noOutputGit("", cPush, remote, tagName)
}

func makeAnnotation(changeLog []string, tagName string) string {
	output := []string{
		"Bump version " + tagName,
		"",
	}
	for _, line := range changeLog {
		output = append(output, "* "+line)
	}
	return strings.Join(output, "\n")
}

func createFlag(name, short string, value bool, usage string) *bool {
	p := flag.Bool(name, value, usage)
	if len(short) > 0 {
		flag.BoolVar(p, short, value, usage)
	}
	return p
}

type bumptagArgs struct {
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
        --find-tag  Show the last tag, can be useful for CI tools`
	fmt.Println(output)
}

func newBumptagArgs() *bumptagArgs {
	return &bumptagArgs{
		edit:     createFlag("edit", "e", false, "Edit an annotation"),
		dryRun:   createFlag("dry-run", "r", false, "Prints an annotation for the new tag"),
		silent:   createFlag("silent", "s", false, "Do not show the created tag"),
		autoPush: createFlag("auto-push", "a", false, "Push the created tag automatically"),
		major:    createFlag("major", "m", false, "Increment the MAJOR version"),
		minor:    createFlag("minor", "n", false, "Increment the MINOR version (default)"),
		patch:    createFlag("patch", "p", false, "Increment the PATCH version"),
		version:  createFlag("version", "", false, "Show a version of the bumptag tool"),
		findTag:  createFlag("find-tag", "", false, "Show the latest tag, can be useful for CI tools"),
	}
}

func setTag(tag *semver.Version, args *bumptagArgs) {
	if flag.NArg() > 0 {
		if err := tag.Set(strings.TrimPrefix(flag.Arg(0), tagPrefix)); err != nil {
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

	cmd := exec.Command(executable, filename)
	cmd.Stdin = os.Stdin
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

	flag.Usage = args.usage
	flag.Parse()

	if *args.version {
		fmt.Print(version)
		return
	}

	tearDownGPG, err := setUpGPG()
	panicIfError(err)
	defer tearDownGPG()

	tag, tagName, err := findTag()
	panicIfError(err)

	if *args.findTag {
		fmt.Print(tagName)
		return
	}

	changeLog, err := getChangeLog(tagName)
	panicIfError(err)

	setTag(tag, args)
	tagName = tagPrefix + tag.String()
	annotation := makeAnnotation(changeLog, tagName)

	if *args.edit {
		annotation, err = edit(annotation)
		panicIfError(err)
	}

	if *args.dryRun {
		fmt.Println(annotation)
		return
	}

	sign := gitConfigBool(cCommitGPGSign, false)
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
