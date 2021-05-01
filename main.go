package main

import (
	"goscp-demo/utils/scp"
	"goscp-demo/utils/ssh"
	"log"
)

var (
	username = "root"
	password = "Ighost@2019"
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
