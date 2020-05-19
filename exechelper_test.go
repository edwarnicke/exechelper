// +build !windows

package exechelper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/edwarnicke/exechelper"
)

const (
	key1 = "key1"
)

func TestRunCmdDoesNotExist(t *testing.T) {
	_, err := exechelper.CombinedOutput("fgipunergibergjefbg")
	assert.Error(t, err)
}

func TestOutputCmdDoesExist(t *testing.T) {
	_, err := exechelper.Output("ls")
	assert.NoError(t, err)
}

func TestRunCmdNonZeroExitCode(t *testing.T) {
	err := exechelper.Run("false")
	assert.Error(t, err)
}

func TestCombinedOutputWithDir(t *testing.T) {
	// Try with existing dir
	dir := "/"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		defer func() { _ = os.Remove(dir) }()
	}
	b, err := exechelper.CombinedOutput("pwd", exechelper.WithDir(dir))
	assert.NoError(t, err)
	assert.Equal(t, dir, path.Base(strings.TrimSpace(string(b))))

	// Try with *hopefully* non-existent dir
	dir = fmt.Sprintf("testdir-%d", rand.Int()) // #nosec
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		defer func() { _ = os.Remove(dir) }()
	}
	b, err = exechelper.CombinedOutput("pwd", exechelper.WithDir(dir))
	assert.NoError(t, err)
	assert.Equal(t, dir, path.Base(strings.TrimSpace(string(b))))
}

func TestStartWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	errCh := exechelper.Start("sleep 600", exechelper.WithContext(ctx), exechelper.CmdOption(func(cmd *exec.Cmd) error {
		return nil
	}))
	cancel()
	select {
	case err := <-errCh:
		assert.IsType(t, &exec.ExitError{}, err) // Because we canceled we will get an exec.ExitError{}
		assert.Empty(t, errCh)
	case <-time.After(time.Second):
		assert.Fail(t, "Failed to cancel context")
	}
}

func TestCmdOptionErr(t *testing.T) {
	_, err := exechelper.Output("ls")
	assert.NoError(t, err)
	_, err = exechelper.Output("ls", exechelper.CmdOption(func(cmd *exec.Cmd) error {
		return errors.New("Test Error")
	}))
	assert.Error(t, err)
}

func TestWithArgs(t *testing.T) {
	output, err := exechelper.Output("echo", exechelper.WithArgs("foo"))
	assert.NoError(t, err)
	assert.Equal(t, "foo", strings.TrimSpace(string(output)))

	output, err = exechelper.Output("echo", exechelper.WithArgs("foo", "bar"))
	assert.NoError(t, err)
	assert.Equal(t, "foo bar", strings.TrimSpace(string(output)))
}

func TestWithStdin(t *testing.T) {
	testStr := "hello world"
	bufferIn := bytes.NewBuffer([]byte(testStr))
	b, err := exechelper.Output("cat -", exechelper.WithStdin(bufferIn))
	assert.NoError(t, err)
	assert.Equal(t, testStr, strings.TrimSpace(string(b)))
}

func TestWithStdout(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	output, err := exechelper.Output("ls", exechelper.WithStdout(buffer))
	assert.NoError(t, err)
	assert.Equal(t, string(output), buffer.String())
}

func TestWithStderr(t *testing.T) {
	buffer1 := bytes.NewBuffer([]byte{})
	buffer2 := bytes.NewBuffer([]byte{})
	err := exechelper.Run("ls fdhdhdhahdr",
		exechelper.WithStderr(buffer1),
		exechelper.WithStderr(buffer2),
	)
	assert.Error(t, err)
	assert.True(t, buffer1.Len() > 0)
	assert.Equal(t, buffer1.String(), buffer2.String())
}

func TestWithEnvMap(t *testing.T) {
	// Try one
	key1 := "key1"
	value1 := "value1"
	one := map[string]string{key1: value1}
	b, err := exechelper.Output("printenv", exechelper.WithEnvMap(one))
	assert.NoError(t, err)
	assert.Equal(t, key1+"="+value1, strings.TrimSpace(string(b)))

	// Try more than one
	key2 := "key2"
	value2 := "value2"
	two := map[string]string{key1: value1, key2: value2}
	b, err = exechelper.Output("printenv", exechelper.WithEnvMap(two))
	assert.NoError(t, err)
	assert.Contains(t, string(b), key1+"="+value1)
	assert.Contains(t, string(b), key2+"="+value2)

	// Try more than one sequentially
	andThenTwo := map[string]string{key2: value2}
	b, err = exechelper.Output("printenv", exechelper.WithEnvMap(one), exechelper.WithEnvMap(andThenTwo))
	assert.NoError(t, err)
	assert.Contains(t, string(b), key1+"="+value1)
	assert.Contains(t, string(b), key2+"="+value2)

	// Overwrite value
	overwrite := map[string]string{key1: value2}
	b, err = exechelper.Output("printenv", exechelper.WithEnvMap(one), exechelper.WithEnvMap(overwrite))
	assert.NoError(t, err)
	assert.Contains(t, string(b), key1+"="+value2)
	assert.NotContains(t, string(b), key1+"="+value1)
}

func TestWithEnviron(t *testing.T) {
	one := "key1=value1"
	b, err := exechelper.Output("printenv", exechelper.WithEnvirons(one))
	assert.NoError(t, err)
	assert.Equal(t, one, strings.TrimSpace(string(b)))

	invalid := key1
	_, err = exechelper.Output("printenv", exechelper.WithEnvirons(invalid))
	assert.Error(t, err)
}

func TestWithEnvKV(t *testing.T) {
	_, err := exechelper.Output("printenv", exechelper.WithEnvKV(key1))
	assert.Error(t, err)
}
