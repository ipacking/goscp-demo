package util

import "testing"

//go test -run Ssh
func TestSsh(t *testing.T) {
	c := New("192.144.238.254", "ubuntu", "xxx", 22)
	c.Run("rm -f a.txt")
}
