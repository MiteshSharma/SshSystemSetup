package domain

import (
	"io"
	"fmt"
	"strings"
	"golang.org/x/crypto/ssh"
	"github.com/MiteshSharma/SshSystemSetup/modal"
)

type SSHMainClient struct {
	Config *ssh.ClientConfig
}

func (client *SSHMainClient) RunCommand(instance modal.InstanceDetail, cmd *modal.SSHCommand) error {
	var session *ssh.Session
	var err error

	if session, err = client.newSession(instance.PublicIp, 22); err != nil {
		fmt.Print("Failed during session creation")
		return err
	}
	defer session.Close()

	if err = client.prepareCommand(session, cmd); err != nil {
		return err
	}

	if (len(cmd.Path) > 0) {
		command := strings.Join(cmd.Path, "; ")
		err = session.Run(command)
	}

	return err
}

func (client *SSHMainClient) prepareCommand(session *ssh.Session, cmd *modal.SSHCommand) error {
	for _, env := range cmd.Env {
		variable := strings.Split(env, "=")
		if len(variable) != 2 {
			continue
		}

		if err := session.Setenv(variable[0], variable[1]); err != nil {
			return err
		}
	}

	if cmd.Stdin != nil {
		stdin, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stdin for session: %v", err)
		}
		go io.Copy(stdin, cmd.Stdin)
	}

	if cmd.Stdout != nil {
		stdout, err := session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stdout for session: %v", err)
		}
		go io.Copy(cmd.Stdout, stdout)
	}

	if cmd.Stderr != nil {
		stderr, err := session.StderrPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stderr for session: %v", err)
		}
		go io.Copy(cmd.Stderr, stderr)
	}

	return nil
}

func (client *SSHMainClient) newSession(host string, port int) (*ssh.Session, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), client.Config)
	if err != nil {
		fmt.Print("Failed to dial: %s", err)
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		fmt.Print("Failed to create session: %s", err)
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	modes := ssh.TerminalModes{
		// ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		fmt.Print("request for pseudo terminal failed: %s", err)
		return nil, fmt.Errorf("request for pseudo terminal failed: %s", err)
	}

	return session, nil
}