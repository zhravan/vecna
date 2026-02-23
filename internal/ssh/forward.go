package ssh

import (
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// StartPortForward listens on localAddr and forwards each connection to remoteAddr via the SSH client.
// Returns a stop function that closes the listener (and thus all accepted connections).
func StartPortForward(client *ssh.Client, localAddr, remoteAddr string) (stop func(), err error) {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}

	var once sync.Once
	stop = func() {
		once.Do(func() {
			listener.Close()
		})
	}

	go func() {
		defer listener.Close()
		for {
			localConn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer localConn.Close()
				remoteConn, err := client.Dial("tcp", remoteAddr)
				if err != nil {
					return
				}
				defer remoteConn.Close()
				go io.Copy(localConn, remoteConn)
				io.Copy(remoteConn, localConn)
			}()
		}
	}()

	return stop, nil
}
