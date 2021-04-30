package main

import (
	"github.com/eleztian/go-scp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
)

var (
	addr     = "192.144.238.254:22"
	user     = "ubuntu"
	password = "xxx"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cfg := &ssh.ClientConfig{
		Config: ssh.Config{},
		User:   user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := scp.New(addr, cfg)
	if err != nil {
		log.Println(err)
	}
	defer client.Close()

	err = client.Upload("a.txt", "/home/ubuntu/b.txt")
	if err != nil {
		log.Fatalln(err)
	}

	err = client.Download("/home/ubuntu/b.txt", "./")
	if err != nil {
		log.Fatalln(err)
	}
}
