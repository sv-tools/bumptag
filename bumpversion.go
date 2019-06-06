package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/coreos/go-semver/semver"
	flag "github.com/spf13/pflag"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
)

var version = "0.0.0"

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
	header := "Bump version " + tagName
	output := []string{
		header,
		strings.Repeat("-", len(header)),
		"",
	}
	for _, line := range changeLog {
		output = append(output, "* "+line)
	}
	return strings.Join(output, "\n")
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: bumpversion [<version>]\n")
	_, _ = fmt.Fprint(os.Stderr, "  <version>       The name of the tag to create, must be Semantic Versions 2.0.0 http://semver.org\n")
	flag.PrintDefaults()
	_, _ = fmt.Fprint(os.Stderr, "\nTo configure a Tag's prefix use git command:\n")
	_, _ = fmt.Fprint(os.Stderr, "  git config bumpversion.Prefix v\n")
}

func main() {
	flag.Usage = usage
	dryRun := flag.BoolP("dry-run", "r", false, "Prints an annotation for the new tag")
	major := flag.BoolP("major", "m", false, "Increment the MAJOR version")
	minor := flag.BoolP("minor", "n", false, "Increment the MINOR version (default)")
	patch := flag.BoolP("patch", "p", false, "Increment the PATCH version")
	showVersion := flag.Bool("version", false, "Show a version of bumpversion tool")
	printTag := flag.Bool("find-tag", false, "Show the latest tag, can be useful for CI tools")
	flag.Parse()

	if *showVersion {
		println("version", version)
		os.Exit(0)
	}

	tagPrefix := gitConfig("bumpversion.Prefix", "")
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
		os.Exit(0)
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
		os.Exit(0)
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
