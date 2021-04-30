package util

import "testing"

var (
	host     = "192.144.238.254"
	port     = 22
	username = "root"
	password = "xxx"
)

//go test -run Ssh
func TestSsh(t *testing.T) {
	c := New(host, username, password, port)
	c.Run("rm -f a.txt")
}
