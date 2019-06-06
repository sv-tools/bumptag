package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/coreos/go-semver/semver"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
)

var version = "0.0.0"

const tagPrefix = "v"

func git(input string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	if len(input) > 0 {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		text := fmt.Sprintf(
			"Command '%s' failed: %s",
			strings.Join(cmd.Args, " "),
			err.Error(),
		)
		errText := strings.TrimSpace(stderr.String())
		if len(errText) > 0 {
			text += "\n" + errText
		}
		err = errors.New(text)
	}
	return strings.TrimSpace(stdout.String()), err
}

func disableGPG() (string, error) {
	output, _ := git("", "config", "--local", "--get", "log.showSignature")
	_, err := git("", "config", "--local", "log.showSignature", "false")
	if err != nil {
		return "", err
	}
	return output, nil
}

func restoreGPG(oldValue string) (err error) {
	if len(oldValue) > 0 {
		_, err = git("", "config", "--local", "log.showSignature", "false")
	} else {
		_, err = git("", "config", "--local", "--unset", "log.showSignature")
	}
	return
}

func setUpGPG() (func(), error) {
	oldVlue, err := disableGPG()
	if err != nil {
		return nil, err
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		_ = restoreGPG(oldVlue)
		os.Exit(1)
	}()

	return func() {
		_ = restoreGPG(oldVlue)
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
	value, _ := strconv.ParseBool(output)
	return value
}

func findTag(prefix string) (*semver.Version, string, error) {
	currentTag := &semver.Version{}
	currentTagName := ""
	output, err := git("", "log", "--pretty=%D")
	if err != nil {
		return nil, "", err
	}
	for _, line := range strings.Split(output, "\n") {
		for _, ref := range strings.Split(line, ",") {
			ref = strings.TrimSpace(ref)
			if strings.HasPrefix(ref, "tag:") {
				rawTag := strings.TrimPrefix(ref, "tag:")
				rawTag = strings.TrimSpace(rawTag)
				tag, err := semver.NewVersion(strings.TrimPrefix(rawTag, prefix))
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

func makeTagName(tag *semver.Version, prefix string) string {
	return prefix + tag.String()
}

func createTag(tagName, annotation string, sign bool) error {
	args := []string{"tag", "-F-"}
	if sign {
		args = append(args, "--sign")
	}
	args = append(args, tagName)
	_, err := git(annotation, args...)
	if err != nil {
		return err
	}
	return nil
}

func showTag(tagName string) (string, error) {
	output, err := git("", "show", tagName)
	if err != nil {
		return "", err
	}
	return output, nil
}

func getChangeLog(tagName string) ([]string, error) {
	args := []string{"log", "--pretty=%h %s", "--no-merges"}
	if len(tagName) > 0 {
		args = append(args, tagName+"..HEAD")
	}
	output, err := git("", args...)
	if err != nil {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
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

func usage() {
	output := `Usage: bumpversion [<tagname>]

    <tagname>       The name of the tag to create, must be Semantic Versions 2.0.0 (http://semver.org)",
    -r, --dry-run   Prints an annotation for the new tag",
    -m, --major     Increment the MAJOR version",
    -n, --minor     Increment the MINOR version (default)",
        --version   Show a version of the bumpversion tool",
        --find-tag  Show the last tag, can be useful for CI tools`
	_, _ = fmt.Fprintln(os.Stderr, output)
}

func createFlag(name, short string, value bool, usage string) *bool {
	p := flag.Bool(name, value, usage)
	if len(short) > 0 {
		flag.BoolVar(p, short, value, name)
	}
	return p
}

func main() {
	flag.Usage = usage
	dryRun := createFlag("dry-run", "r", false, "")
	major := createFlag("major", "m", false, "")
	minor := createFlag("minor", "n", false, "")
	patch := createFlag("patch", "p", false, "")
	showVersion := createFlag("version", "", false, "")
	printTag := createFlag("find-tag", "", false, "Show the latest tag, can be useful for CI tools")
	flag.Parse()

	if *showVersion {
		println("version", version)
		return
	}

	sign := gitConfigBool("commit.gpgsign", false)

	tearDownGPG, err := setUpGPG()
	if err != nil {
		panic(err)
	}
	defer tearDownGPG()

	tag, tagName, err := findTag(tagPrefix)
	if err != nil {
		panic(err)
	}

	if *printTag {
		print(tagName)
		return
	}

	changeLog, err := getChangeLog(tagName)

	if flag.NArg() > 0 {
		err := tag.Set(strings.TrimPrefix(flag.Arg(0), tagPrefix))
		if err != nil {
			panic(err)
		}
	} else {
		if *major {
			tag.BumpMajor()
		} else {
			if *minor {
				tag.BumpMinor()
			} else {
				if *patch {
					tag.BumpPatch()
				} else {
					tag.BumpMinor()
				}
			}
		}
	}

	tagName = makeTagName(tag, tagPrefix)
	if err != nil {
		panic(err)
	}
	annotation := makeAnnotation(changeLog, tagName)

	if *dryRun {
		println(annotation)
		return
	}

	err = createTag(tagName, annotation, sign)
	if err != nil {
		panic(err)
	}
	output, err := showTag(tagName)
	if err != nil {
		panic(err)
	}
	println(output)
}
