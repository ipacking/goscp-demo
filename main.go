package main

import (
	"github.com/eleztian/go-scp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
)

var (
	addr     = "192.144.238.254:22"
	username = "root"
	password = "xxx"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	conf := &ssh.ClientConfig{
		Config: ssh.Config{},
		User:   username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	cli, err := scp.New(addr, conf)
	if err != nil {
		log.Println(err)
	}
	defer cli.Close()

	err = cli.Upload("a.txt", "/root/b.txt")
	if err != nil {
		log.Fatalln(err)
	}

	err = cli.Download("/root/b.txt", ".")
	if err != nil {
		log.Fatalln(err)
	}
}
