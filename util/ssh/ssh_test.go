package ssh

import "testing"

var (
	username = "root"
	password = "xxx"
	host     = "118.24.12.180"
	port     = 22
)

//go test -run SSH
func TestSSH(t *testing.T) {
	c := New(username, password, host, port)
	c.Run("rm -f a.txt")
}
