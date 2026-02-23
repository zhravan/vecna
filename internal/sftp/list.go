package sftp

import (
	"os"

	sftppkg "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ListRemote lists the given path on the remote server over the SSH connection.
// Caller must ensure conn is valid; it is not closed by this function.
func ListRemote(conn *ssh.Client, path string) ([]os.FileInfo, error) {
	client, err := sftppkg.NewClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	return client.ReadDir(path)
}
