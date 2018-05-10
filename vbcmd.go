package virtualbox

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
)

type option func(Command)

// Command is the mock-able interface to run VirtualBox commands
// such as VBoxManage (host side) or VBoxControl (guest side)
type Command interface {
	setOpts(opts ...option)
	path() string
	run(args ...string) error
	runOut(args ...string) (string, error)
	runOutErr(args ...string) (string, string, error)
}

var (
	// Verbose toggles the library in verbose execution mode.
	Verbose bool
	// ErrMachineExist holds the error message when the machine already exists.
	ErrMachineExist = errors.New("machine already exists")
	// ErrMachineNotExist holds the error message when the machine does not exist.
	ErrMachineNotExist = errors.New("machine does not exist")
	// ErrCommandNotFound holds the error message when the VBoxManage commands was not found.
	ErrCommandNotFound = errors.New("command not found")
)

type command struct {
	program string
	// Is current user a sudoer?
	sudoer bool
	// Is current command expected to be run under sudo?
	sudo bool
}

func isSudoer() (bool, error) {
	me, err := user.Current()
	if err != nil {
		return false, err
	}
	if groupIDs, err := me.GroupIds(); runtime.GOOS == "linux" {
		if err != nil {
			return false, err
		}
		for _, groupID := range groupIDs {
			if groupID == "sudo" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (vbcmd command) setOpts(opts ...option) {
	for _, opt := range opts {
		var cmd Command = &vbcmd
		opt(cmd)
	}
}

func sudo(sudo bool) option {
	return func(cmd Command) {
		vbcmd := cmd.(*command)
		vbcmd.sudo = sudo
	}
}

func (vbcmd command) path() string {
	return vbcmd.program
}

func (vbcmd command) prepare(args []string) *exec.Cmd {
	program := vbcmd.program
	argv := []string{}
	if vbcmd.sudoer && vbcmd.sudo && runtime.GOOS != "windows" {
		program = "sudo"
		argv = append(argv, vbcmd.program)
	}
	argv = append(argv, args...)
	if Verbose {
		log.Printf("executing: %v %v", program, argv)
	}
	return exec.Command(program, argv...)
}

func (vbcmd command) run(args ...string) error {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	if Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			return ErrCommandNotFound
		}
		return err
	}
	return nil
}

func (vbcmd command) runOut(args ...string) (string, error) {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	if Verbose {
		cmd.Stderr = os.Stderr
	}

	b, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			err = ErrCommandNotFound
		}
	}
	return string(b), err
}

func (vbcmd command) runOutErr(args ...string) (string, string, error) {
	defer vbcmd.setOpts(sudo(false))
	cmd := vbcmd.prepare(args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee == exec.ErrNotFound {
			err = ErrCommandNotFound
		}
	}
	return stdout.String(), stderr.String(), err
}
