package scp

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

const defaultRemoteBinary = "/usr/bin/scp"

type Client struct {
	Username     string
	Password     string
	Host         string
	Port         int
	client       *ssh.Client
	RemoteBinary string
}

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

	cli.RemoteBinary = defaultRemoteBinary
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

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Upload(local string, remote string) error {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}

	ses, err := c.newSession()
	if err != nil {
		return err
	}
	defer ses.Close()

	return ses.Send(local, remote)
}

func (c *Client) Download(remote string, local string) error {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}

	ses, err := c.newSession()
	if err != nil {
		return err
	}
	defer ses.Close()

	return ses.Recv(remote, local)
}

type session struct {
	*ssh.Session
	stdIn        io.WriteCloser
	stdOut       io.Reader
	remoteBinary string
}

func (c *Client) newSession() (*session, error) {
	ses, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	res := &session{Session: ses, remoteBinary: c.RemoteBinary}

	res.stdIn, err = res.StdinPipe()
	if err != nil {
		return nil, err
	}

	res.stdOut, err = res.Session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	return res, nil
}
func (s *session) Close() {
	_ = s.Session.Close()
}

func (s *session) Send(local string, remote string) error {
	info, err := os.Stat(local)
	if err != nil {
		return err
	}

	wg, _ := errgroup.WithContext(context.Background())
	wg.Go(func() error {
		defer s.stdIn.Close()

		rsp, err := ReadResp(s.stdOut)
		if err != nil {
			return err
		}
		if rsp.IsFailure() {
			return errors.New(rsp.GetMessage().String())
		}

		if info.IsDir() {
			return s.sendDir(local, remote)
		} else {
			return s.sendFile(local, remote)
		}
	})
	var rE error
	wg.Go(func() error {
		rE = s.Run(fmt.Sprintf("%s -rt %s", s.remoteBinary, filepath.Dir(remote)))
		return nil
	})

	if err = wg.Wait(); err != nil {
		return err
	}
	return rE
}

func (s *session) sendFile(local string, remote string) error {
	_, remoteName := filepath.Split(remote)
	info, err := os.Stat(local)
	if err != nil {
		return err
	}

	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer f.Close()

	err = NewFile(info.Mode(), remoteName, info.Size()).WriteStream(s.stdIn, f)
	if err != nil {
		return err
	}

	rsp, err := ReadResp(s.stdOut)
	if err != nil {
		return err
	}
	if rsp.IsFailure() {
		return errors.New(rsp.GetMessage().String())
	}

	//fmt.Printf("FILE: %s => %s %d\n", local, remote, info.Size())

	return err
}

func (s *session) sendDir(local string, remotePath string) error {
	info, err := os.Stat(local)
	if err != nil {
		return err
	}

	err = NewDirBegin(info.Mode(), info.Name()).Write(s.stdIn)
	if err != nil {
		return err
	}
	rsp, err := ReadResp(s.stdOut)
	if err != nil {
		return err
	}
	if rsp.IsFailure() {
		return errors.New(rsp.GetMessage().String())
	}

	fs, err := os.ReadDir(local)
	if err != nil {
		return err
	}
	for _, f := range fs {
		src, remote := filepath.Join(local, f.Name()), filepath.Join(remotePath, f.Name())

		if f.IsDir() {
			err = s.sendDir(src, remote)
		} else {
			err = s.sendFile(src, remote)
		}
		if err != nil {
			return err
		}
	}

	err = NewDirEnd().Write(s.stdIn)
	if err != nil {
		return err
	}

	rsp, err = ReadResp(s.stdOut)
	if err != nil {
		return err
	}
	if rsp.IsFailure() {
		return errors.New(rsp.GetMessage().String())
	}

	return err
}

func (s *session) Recv(remote string, local string) error {
	wg, _ := errgroup.WithContext(context.Background())
	wg.Go(func() error {
		defer s.stdIn.Close()

		err := NewOkRsp().Write(s.stdIn)
		if err != nil {
			return err
		}

		err = s.recvCmd(local, remote)
		if err != nil && err != io.EOF {
			return err
		}
		return nil
	})
	var err error
	wg.Go(func() error {
		err = s.Run(fmt.Sprintf("%s -rf %s", s.remoteBinary, remote))
		return nil
	})

	if err := wg.Wait(); err != nil {
		return err
	}
	return err
}
func (s *session) recvCmd(local string, remote string) error {
	rsp, err := ReadResp(s.stdOut)
	if err != nil {
		return err
	}

	if rsp.IsDir() {
		mode, _, filename, err := rsp.GetMessage().FileInfo()
		if err != nil {
			return err
		}
		return s.recvDir(mode, filepath.Join(local, filename), filepath.Join(remote, filename))
	} else if rsp.IsFile() {
		mode, size, filename, err := rsp.GetMessage().FileInfo()
		if err != nil {
			return err
		}
		return s.recvFile(mode, size, filepath.Join(local, filename), filepath.Join(remote, filename))
	} else if rsp.IsEndDir() {
		return io.EOF
	} else {
		return errors.New("invalid protocol")
	}

}

func (s *session) recvDir(mode os.FileMode, local string, remote string) error {
	err := os.MkdirAll(local, mode)
	if err != nil {
		_ = NewErrorRsp(err.Error()).Write(s.stdIn)
		return err
	}
	err = NewOkRsp().Write(s.stdIn)
	if err != nil {
		return err
	}

	for {
		err = s.recvCmd(local, remote)
		if err != nil {
			if err == io.EOF { // dir end
				err = NewOkRsp().Write(s.stdIn)
				if err != nil {
					return err
				}
			}
			return err
		}
	}
}

func (s *session) recvFile(mode os.FileMode, size int64, local string, remote string) error {
	f, err := os.OpenFile(local, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		_ = NewErrorRsp(err.Error()).Write(s.stdIn)
		return err
	}
	defer f.Close()

	err = NewOkRsp().Write(s.stdIn)
	if err != nil {
		return err
	}

	_, err = io.CopyN(f, s.stdOut, size)
	if err != nil {
		_ = NewErrorRsp(err.Error()).Write(s.stdIn)
		return err
	}

	rsp, err := ReadResp(s.stdOut)
	if err != nil {
		_ = NewErrorRsp(err.Error()).Write(s.stdIn)
		return err
	}
	if rsp.IsFailure() {
		return errors.New(rsp.GetMessage().String())
	}

	return NewOkRsp().Write(s.stdIn)
}
