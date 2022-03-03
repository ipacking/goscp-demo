package main

import (
	"goscp-demo/tools/scp"
	"goscp-demo/tools/ssh"
	"log"
)

var (
	username = "root"
	password = "xxx"
	host     = "118.24.12.180"
	port     = 22
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	c1 := scp.New(username, password, host, port)
	c1.Upload("a.txt", "b.txt")
	c1.Download("/root/b.txt", ".")

	c2 := ssh.New(username, password, host, port)
	c2.Run("rm -f b.txt")
}
