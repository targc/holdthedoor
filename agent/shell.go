package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
)

type Shell struct {
	cmd *exec.Cmd
	pty *os.File
}

func NewShell() (*Shell, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe")
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell)
	}

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &Shell{cmd: cmd, pty: ptmx}, nil
}

func (s *Shell) Read(buf []byte) (int, error) {
	return s.pty.Read(buf)
}

func (s *Shell) Write(data []byte) (int, error) {
	return s.pty.Write(data)
}

func (s *Shell) Resize(cols, rows int) error {
	return pty.Setsize(s.pty, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func (s *Shell) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	if s.pty != nil {
		s.pty.Close()
	}
	return nil
}
