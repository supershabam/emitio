package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
)

type transformFn func(acc string, msg Message) (string, []string, error)

type node struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	script string
}

func (n *node) run() error {
	f, err := ioutil.TempFile("", "emitio")
	if err != nil {
		return err
	}
	_, err = io.Copy(f, bytes.NewReader([]byte(n.script)))
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	cmd := exec.Command("node", f.Name())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	n.stdin = stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	n.stdout = stdout
	return nil
}

func (n *node) transform(acc string, msg Message) (string, []string, error) {
	in := struct {
		Acc string `json:"acc"`
		Msg string `json:"msg"`
	}{
		Acc: acc,
		Msg: "bullshit",
	}
	b, err := json.Marshal(in)
	if err != nil {
		return "", nil, err
	}
	_, err = n.stdin.Write(b)
	if err != nil {
		return "", nil, err
	}
	buf := make([]byte, 1024*32)
	_, err = n.stdout.Read(buf)
	if err != nil {
		return "", nil, err
	}
	out := struct {
		Acc   string   `json:"acc"`
		Lines []string `json:"lines"`
	}{}
	err = json.Unmarshal(buf, &out)
	if err != nil {
		return "", nil, err
	}
	return out.Acc, out.Lines, nil
}

func ParseTransformer(token string) (transformFn, error) {
	var s struct {
		Type        string `json:"type"`
		Start       []byte `json:"start"`
		Script      string `json:"script"`
		Accumulator string `json:"accumulator"`
	}
	err := json.Unmarshal([]byte(token), &s)
	if err != nil {
		return nil, err
	}
	switch s.Type {
	case "node":
		n := node{script: s.Script}
		err := n.run()
		if err != nil {
			return nil, err
		}
		return n.transform, nil
	default:
		return nil, errors.New("unhandled transformer type")
	}
}
