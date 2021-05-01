package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"net"
	"os"
	"time"
)

type Client struct {
	Username   string
	Password   string
	Host       string
	Port       int
	client     *ssh.Client
	LastResult string //last run result
}

//create client
//@param username username
//@param password password
//@param host hostname
//@param port port default 22
func New(username string, password string, host string, port ...int) *Client {
	cli := new(Client)
	cli.Host = host
	cli.Username = username
	cli.Password = password
	if len(port) <= 0 {
		cli.Port = 22
	} else {
		cli.Port = port[0]
	}
	return cli
}

func (c *Client) connect() error {
	conf := ssh.ClientConfig{
		Config:          ssh.Config{},
		User:            c.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(c.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         time.Second * 10,
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	client, err := ssh.Dial("tcp", addr, &conf)
	if err != nil {
		return err
	}

	c.client = client
	return nil
}

//execute command
//@param shell shell command
func (c Client) Run(shell string) (string, error) {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return "", err
		}
	}
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	buf, err := session.CombinedOutput(shell)

	c.LastResult = string(buf)
	return c.LastResult, err
}

func (c *Client) RunTerminal(shell string, stdout, stderr io.Writer) error {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(fd, oldState)

	session.Stdout = stdout
	session.Stderr = stderr
	session.Stdin = os.Stdin

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}

	// setup terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4 kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4 kbaud
	}

	// request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		return err
	}

	return session.Run(shell)
}
