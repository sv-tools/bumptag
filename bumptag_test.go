package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

// MockGit is a mock of MockGit function
type MockGit struct {
	ctrl     *gomock.Controller
	recorder *MockGitMockRecorder
}

// MockGitMockRecorder is the mock recorder for MockGit
type MockGitMockRecorder struct {
	mock *MockGit
}

// NewMockGit creates a new mock instance
func NewMockGit(ctrl *gomock.Controller) *MockGit {
	mock := &MockGit{ctrl: ctrl}
	mock.recorder = &MockGitMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGit) EXPECT() *MockGitMockRecorder {
	return m.recorder
}

// Git mocks base method
func (m *MockGit) Git(arg0 string, arg1 ...string) (string, error) {
	m.ctrl.T.Helper()
	msg := fmt.Sprintf("git %s", strings.Join(arg1, " "))
	if len(arg0) > 0 {
		msg += "\nInput: " + arg0
	}
	m.ctrl.T.(testing.TB).Log(msg)
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Git", varargs...)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Git indicates an expected call of Git
func (mr *MockGitMockRecorder) Git(arg0 string, arg1 ...string) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Git", reflect.TypeOf((*MockGit)(nil).Git), varargs...)
}

func mockGit(t testing.TB) (*MockGit, func()) {
	ctrl := gomock.NewController(t)
	mockGit := NewMockGit(ctrl)
	git = mockGit.Git
	return mockGit, func() {
		ctrl.Finish()
		git = realGit
	}
}

func mockStderr(t testing.TB) (read func() string, tearDown func()) {
	reader, writer, err := os.Pipe()
	assert.NoError(t, err)

	realStderr := os.Stderr
	os.Stderr = writer
	read = func() string {
		var buf bytes.Buffer
		err = writer.Close()
		assert.NoError(t, err)
		_, err = io.Copy(&buf, reader)
		assert.NoError(t, err)
		err = reader.Close()
		assert.NoError(t, err)
		output := buf.String()
		t.Logf("mockStderr output: %s", output)
		return output
	}
	tearDown = func() {
		os.Stderr = realStderr
	}
	return
}

func mockStdout(t testing.TB) (read func() string, tearDown func()) {
	reader, writer, err := os.Pipe()
	assert.NoError(t, err)

	realStdout := os.Stdout
	os.Stdout = writer
	read = func() string {
		var buf bytes.Buffer
		err = writer.Close()
		assert.NoError(t, err)
		_, err = io.Copy(&buf, reader)
		assert.NoError(t, err)
		err = reader.Close()
		assert.NoError(t, err)
		output := buf.String()
		t.Logf("mockStdout output: %s", output)
		return output
	}
	tearDown = func() {
		os.Stdout = realStdout
	}
	return
}

func mockArgs(t testing.TB, arg ...string) func() {
	realArgs := os.Args
	os.Args = append([]string{"bumptag"}, arg...)
	t.Logf("Args: %s", strings.Join(os.Args, " "))
	return func() {
		os.Args = realArgs
	}
}

func TestRealGit(t *testing.T) {
	output, err := realGit("", "status")
	assert.NoError(t, err)
	t.Logf(output)

	output, err = realGit("fake", "fail-cmd")
	assert.Error(t, err)
	t.Logf(output)
}

func TestMockedGit(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("").Return("test output", nil)
	output, err := git("")
	assert.NoError(t, err)
	assert.Equal(t, "test output", output)
}

func TestDisabelGPG(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()
	ctrl.EXPECT().
		Git("", "config", "--local", "--get", "log.showSignature").
		Return("true", nil)
	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "false").
		Return("", nil)
	output, err := disableGPG()
	assert.NoError(t, err)
	assert.Equal(t, "true", output)

	ctrl.EXPECT().
		Git("", "config", "--local", "--get", "log.showSignature").
		Return("", errors.New("error 1"))
	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "false").
		Return("", errors.New("error 2"))
	output, err = disableGPG()
	assert.Error(t, err)
	assert.Equal(t, "", output)
	assert.Equal(t, "error 2", err.Error())
}

func TestRestoreGPG(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()
	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "test-value").
		Return("true", nil)
	err := restoreGPG("test-value")
	assert.NoError(t, err)

	ctrl.EXPECT().
		Git("", "config", "--local", "--unset", "log.showSignature").
		Return("", nil)
	err = restoreGPG("")
	assert.NoError(t, err)
}

func TestSetUpGPG(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	if os.Getenv("GO_TEST_SETUPGPG") == "1" {
		ctrl.EXPECT().
			Git("", "config", "--local", "--get", "log.showSignature").
			Return("test-value-signal", nil)
		ctrl.EXPECT().
			Git("", "config", "--local", "log.showSignature", "false").
			Return("", nil)
		_, err := setUpGPG()
		assert.NoError(t, err)

		ctrl.EXPECT().
			Git("", "config", "--local", "log.showSignature", "test-value-signal").
			Return("true", nil)
		err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		assert.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
		t.FailNow()
	}

	// Check failing at disabling the log.showSignature
	ctrl.EXPECT().
		Git("", "config", "--local", "--get", "log.showSignature").
		Return("", nil)
	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "false").
		Return("", errors.New("test-error"))
	_, err := setUpGPG()
	assert.Error(t, err)

	// Check right behavior
	ctrl.EXPECT().
		Git("", "config", "--local", "--get", "log.showSignature").
		Return("test-value", nil)
	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "false").
		Return("", nil)
	tearDownGPG, err := setUpGPG()
	assert.NoError(t, err)

	ctrl.EXPECT().
		Git("", "config", "--local", "log.showSignature", "test-value").
		Return("true", nil)
	tearDownGPG()

	// Check restoring at signal
	cmd := exec.Command(os.Args[0], "-test.run=TestSetUpGPG", "-test.v")
	cmd.Env = append(os.Environ(), "GO_TEST_SETUPGPG=1")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if e, ok := err.(*exec.ExitError); ok && e.ProcessState.ExitCode() == 42 {
		return
	}
	assert.NoError(t, err)
}

func TestGitConfig(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "config", "--get", "test-name").
		Return("test-value", nil)
	output := gitConfig("test-name", "test-default-value")
	assert.Equal(t, "test-value", output)

	ctrl.EXPECT().
		Git("", "config", "--get", "test-name").
		Return("", errors.New("test-random-error"))
	output = gitConfig("test-name", "test-default-value")
	assert.Equal(t, "test-default-value", output)
}

func TestGitConfigBool(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "config", "--get", "test-name").
		Return("false", nil)
	output := gitConfigBool("test-name", true)
	assert.Equal(t, false, output)

	ctrl.EXPECT().
		Git("", "config", "--get", "test-name").
		Return("test-string", nil)
	output = gitConfigBool("test-name", true)
	assert.Equal(t, true, output)
}

func TestFindTag(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "log", "--pretty=%D").
		Return("", errors.New("test-error"))
	_, _, err := findTag()
	assert.Error(t, err)

	ctrl.EXPECT().
		Git("", "log", "--pretty=%D").
		Return("", nil)
	tag, tagName, err := findTag()
	assert.NoError(t, err)
	assert.Equal(t, "", tagName)
	assert.Equal(t, "0.0.0", tag.String())

	ctrl.EXPECT().
		Git("", "log", "--pretty=%D").
		Return(
			`
HEAD -> master, tag: v3.1.0, tag: v3.0.1, origin/master
test_branch
tag: v3.0.0
tag: v2.1.0, sv/master, origin/v2
tag: v2.0.1
tag: v2.0.0
tag: not_a_version
origin/test_branch
tag: v1.0.0, origin/v1.0`, nil)
	tag, tagName, err = findTag()
	assert.NoError(t, err)
	assert.Equal(t, "v3.1.0", tagName)
	assert.Equal(t, "3.1.0", tag.String())
}

func TestCreateTag(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("test-annotation", "tag", "-F-", "test-tag").
		Return("", nil)
	err := createTag("test-tag", "test-annotation", false)
	assert.NoError(t, err)

	ctrl.EXPECT().
		Git("test-annotation", "tag", "-F-", "--sign", "test-tag").
		Return("", nil)
	err = createTag("test-tag", "test-annotation", true)
	assert.NoError(t, err)

	ctrl.EXPECT().
		Git("test-annotation", "tag", "-F-", "--sign", "test-tag").
		Return("", errors.New("test-error"))
	err = createTag("test-tag", "test-annotation", true)
	assert.Error(t, err)
	assert.Equal(t, "test-error", err.Error())
}

func TestShowTag(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "show", "test-tag").
		Return("test-output", nil)
	output, err := showTag("test-tag")
	assert.NoError(t, err)
	assert.Equal(t, "test-output", output)

	ctrl.EXPECT().
		Git("", "show", "test-tag").
		Return("", errors.New("test-error"))
	_, err = showTag("test-tag")
	assert.Error(t, err)
	assert.Equal(t, "test-error", err.Error())
}

func TestGetChangeLog(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "log", "--pretty=%h %s", "--no-merges", "test-tag..HEAD").
		Return("test-output", nil)
	output, err := getChangeLog("test-tag")
	assert.NoError(t, err)
	assert.Equal(t, []string{"test-output"}, output)

	ctrl.EXPECT().
		Git("", "log", "--pretty=%h %s", "--no-merges").
		Return("test-output", nil)
	output, err = getChangeLog("")
	assert.NoError(t, err)
	assert.Equal(t, []string{"test-output"}, output)

	ctrl.EXPECT().
		Git("", "log", "--pretty=%h %s", "--no-merges").
		Return("", errors.New("test-error"))
	_, err = getChangeLog("")
	assert.Error(t, err)
	assert.Equal(t, "test-error", err.Error())
}

func TestMakeAnnotation(t *testing.T) {
	output := makeAnnotation([]string{"test-changelog"}, "test-tag")
	assert.Contains(t, output, "test-changelog")
	assert.Contains(t, output, "test-tag")
}

func TestUsage(t *testing.T) {
	read, tearDown := mockStderr(t)
	defer tearDown()
	usage()
	assert.Contains(t, read(), "bumptag")
}

func TestCreateFlag(t *testing.T) {
	realCommandLine := flag.CommandLine
	defer func() {
		flag.CommandLine = realCommandLine
	}()

	flag.CommandLine = flag.NewFlagSet("test-flag-set-1", flag.ExitOnError)
	_ = createFlag("test-flag", "t", true, "test-usage")
	cnt := 0
	names := []string{"test-flag", "t"}
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		cnt++
		assert.Equal(t, "true", f.DefValue)
		assert.Equal(t, "test-usage", f.Usage)
		assert.Contains(t, names, f.Name)
	})
	assert.Equal(t, 2, cnt)

	flag.CommandLine = flag.NewFlagSet("test-flag-set-2", flag.ExitOnError)
	_ = createFlag("test-flag", "", true, "test-usage")
	cnt = 0
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		cnt++
		assert.Equal(t, "true", f.DefValue)
		assert.Equal(t, "test-usage", f.Usage)
		assert.Equal(t, f.Name, "test-flag")
	})
	assert.Equal(t, 1, cnt)
}

func TestGetRemote(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "branch", "--list", "-vv").
		Return("", errors.New("test-error"))
	output, err := getRemote()
	assert.Error(t, err)
	assert.Empty(t, output)

	ctrl.EXPECT().
		Git("", "branch", "--list", "-vv").
		Return("", nil)
	output, err = getRemote()
	assert.NoError(t, err)
	assert.Equal(t, defaultRemote, output)

	ctrl.EXPECT().
		Git("", "branch", "--list", "-vv").
		Return(`
  master       cc51028 [origin/master] Merge pull request #6 from SVilgelm/tests
* new_features b2fedca Add silent mode`, nil)
	output, err = getRemote()
	assert.Error(t, err)
	assert.Empty(t, output)

	ctrl.EXPECT().
		Git("", "branch", "--list", "-vv").
		Return(`
  master       cc51028 [origin/master] Merge pull request #6 from SVilgelm/tests
* new_features b2fedca [Add silent mode]`, nil)
	output, err = getRemote()
	assert.Error(t, err)
	assert.Empty(t, output)

	ctrl.EXPECT().
		Git("", "branch", "--list", "-vv").
		Return(`
* master       cc51028 [test-origin/master] Merge pull request #6 from SVilgelm/tests
  new_features b2fedca [Add silent mode]`, nil)
	output, err = getRemote()
	assert.NoError(t, err)
	assert.Equal(t, "test-origin", output)
}

func TestPushTag(t *testing.T) {
	ctrl, tearDown := mockGit(t)
	defer tearDown()

	ctrl.EXPECT().
		Git("", "push", "test-remote", "test-tag").
		Return("", errors.New("test-error"))
	err := pushTag("test-remote", "test-tag")
	assert.EqualError(t, err, "test-error")
}

// Scenarios

func execMain(t testing.TB, arg ...string) (stdout, stderr string) {
	realCommandLine := flag.CommandLine
	defer func() {
		flag.CommandLine = realCommandLine
	}()
	flag.CommandLine = flag.NewFlagSet("test-flag-set", flag.ContinueOnError)
	tearDownArgs := mockArgs(t, arg...)
	defer tearDownArgs()
	readStdout, tearDownStdout := mockStdout(t)
	defer tearDownStdout()
	readStderr, tearDownStderr := mockStderr(t)
	defer tearDownStderr()
	main()
	return readStdout(), readStderr()
}

func prepareGit(t testing.TB) (func(), func()) {
	dir, err := ioutil.TempDir("", "bumptag")
	assert.NoError(t, err)
	t.Logf("Dir: %s", dir)
	remoteDir, err := ioutil.TempDir("", "bumptag")
	assert.NoError(t, err)
	t.Logf("Remote dir: %s", dir)
	newGit := func(input string, arg ...string) (string, error) {
		msg := fmt.Sprintf("git %s", strings.Join(arg, " "))
		if len(input) > 0 {
			msg += "\nInput: " + input
		}
		t.Log(msg)
		cmd := exec.Command("git", arg...)
		cmd.Dir = dir
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
		output := strings.TrimSpace(stdout.String())
		t.Logf("Output: %s", output)
		if err != nil {
			t.Logf("Error: %s", err.Error())
		}
		return output, err
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	err = cmd.Run()
	assert.NoError(t, err)

	git = newGit
	_, err = git("", "init")
	assert.NoError(t, err)
	_, err = git("", "remote", "add", "origin", remoteDir)
	assert.NoError(t, err)
	_, err = git("", "config", "--local", "commit.gpgsign", "false")
	assert.NoError(t, err)

	var commitNumber int
	prepareCommit := func() {
		msg := fmt.Sprintf("commit-#%d", commitNumber)
		commitNumber++
		f, err := ioutil.TempFile(dir, msg+"-*.txt")
		assert.NoError(t, err)
		defer func() {
			err := f.Close()
			assert.NoError(t, err)
		}()
		_, err = f.WriteString(msg)
		assert.NoError(t, err)
		err = f.Sync()
		assert.NoError(t, err)
		_, err = git("", "add", filepath.Base(f.Name()))
		assert.NoError(t, err)
		_, err = git("", "commit", "-m", msg)
		assert.NoError(t, err)

	}

	prepareCommit()
	_, err = git("", "push", "--set-upstream", "origin", "master")
	assert.NoError(t, err)

	return prepareCommit, func() {
		git = realGit
		t.Logf("Remove dir: %s", dir)
		err := os.RemoveAll(dir)
		assert.NoError(t, err)
		t.Logf("Remove remote dir: %s", remoteDir)
		err = os.RemoveAll(remoteDir)
		assert.NoError(t, err)
	}
}

func TestMainVersion(t *testing.T) {
	realVersion := version
	version = "test-version"
	defer func() {
		version = realVersion
	}()
	stdout, _ := execMain(t, "--version")
	assert.Contains(t, stdout, "test-version")
}

func TestMainFindTag(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	stdout, _ := execMain(t, "--find-tag")
	assert.Empty(t, stdout)
	_, err := git("", "tag", "v1.0.1")
	assert.NoError(t, err)
	stdout, _ = execMain(t, "--find-tag")
	assert.Equal(t, "v1.0.1", stdout)
}

func TestMainDryRun(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	stdout, _ := execMain(t, "--dry-run")
	assert.Contains(t, stdout, "commit-#0")
	assert.Contains(t, stdout, "v0.1.0")
}

func TestMainTagAuto(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	stdout, _ := execMain(t)
	assert.Contains(t, stdout, "commit-#0")
	assert.Contains(t, stdout, "v0.1.0")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v0.1.0")
}

func TestMainTagSilent(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	stdout, _ := execMain(t, "--silent")
	assert.Empty(t, stdout)
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v0.1.0")
}

func TestMainTagSpecified(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	_, _ = execMain(t, "v3.0.3")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v3.0.3")
}

func TestMainTagMajor(t *testing.T) {
	prepareCommit, tearDown := prepareGit(t)
	defer tearDown()
	_, err := git("", "tag", "v1.1.1")
	assert.NoError(t, err)
	prepareCommit()
	_, _ = execMain(t, "--major")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v2.0.0")
}

func TestMainTagMinor(t *testing.T) {
	prepareCommit, tearDown := prepareGit(t)
	defer tearDown()
	_, err := git("", "tag", "v1.1.1")
	assert.NoError(t, err)
	prepareCommit()
	_, _ = execMain(t, "--minor")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v1.2.0")
}

func TestMainTagPatch(t *testing.T) {
	prepareCommit, tearDown := prepareGit(t)
	defer tearDown()
	_, err := git("", "tag", "v1.1.1")
	assert.NoError(t, err)
	prepareCommit()
	_, _ = execMain(t, "--patch")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v1.1.2")
}

func TestMainTagSpecifiedWrong(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	assert.Panics(t, func() {
		_, _ = execMain(t, "v3.0")
	})
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.NotContains(t, output, "v3.0")
}

func TestMainTagAutoPush(t *testing.T) {
	_, tearDown := prepareGit(t)
	defer tearDown()
	stdout, _ := execMain(t, "--auto-push")
	assert.Contains(t, stdout, "pushed")
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v0.1.0")
	output, err = git("", "ls-remote", "--tags", "origin")
	assert.NoError(t, err)
	assert.Contains(t, output, "v0.1.0")
}

func TestMainTagBranch(t *testing.T) {
	prepareCommit, tearDown := prepareGit(t)
	defer tearDown()

	_, err := git("", "tag", "v1.0.0")
	assert.NoError(t, err)
	_, err = git("", "checkout", "-b", "v1")
	assert.NoError(t, err)
	_, err = git("", "checkout", "master")
	assert.NoError(t, err)
	prepareCommit()
	_, err = git("", "tag", "v2.0.0")
	assert.NoError(t, err)
	_, err = git("", "checkout", "v1")
	assert.NoError(t, err)
	prepareCommit()

	_, _ = execMain(t)
	output, err := git("", "tag", "--list")
	assert.NoError(t, err)
	assert.Contains(t, output, "v1.1.0")
	output, err = git("", "log", "--pretty=%h %d")
	assert.NoError(t, err)
	assert.NotContains(t, output, "v2.0.0")
}
