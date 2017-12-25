package pkg_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os/exec"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type txfn func(ctx context.Context, acc string, in string) (string, string, error)

func NewJSTransform(ctx context.Context, program string) (txfn, error) {
	f, err := ioutil.TempFile("", "emitio")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, bytes.NewReader([]byte(program)))
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("node", f.Name())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, acc string, in string) (string, string, error) {
		buf := make([]byte, 32*1024)
		_, err := stdin.Write([]byte(in))
		if err != nil {
			return "", "", errors.Wrap(err, "write")
		}
		nr, err := stdout.Read(buf)
		if err != nil {
			if err == io.EOF {
				return "", "", cmd.Wait()
			}
			return "", "", errors.Wrap(err, "read")
		}
		return "", string(buf[:nr]), nil
	}, nil
}

func TestTransform(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	program := `
	process.stdin.setEncoding("utf8");
	process.stdin.on("readable", () => {
	  const chunk = process.stdin.read();
	  if (chunk !== null) {
		// TODO handle buffering
		process.stdout.write(JSON.stringify([chunk]));
	  }
	});
	process.stdin.on("end", () => {
	  process.stdout.write("[]");
	});
	
`
	tx, err := NewJSTransform(ctx, program)
	require.Nil(t, err)
	acc, out, err := tx(ctx, "", `[{"ingress":"test"}]`)
	require.Nil(t, err)
	assert.Equal(t, "", acc)
	assert.Equal(t, `[]`, out)
}
