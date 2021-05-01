package scp

import "testing"

var (
	username = "root"
	password = "xxx"
	host     = "118.24.12.180"
	port     = 22
)

//go test -run SCP
func TestSCP(t *testing.T) {
	c := New(username, password, host, port)
	c.Upload("a.txt", "/root/b.txt")
	c.Download("/root/b.txt", "./")
}
