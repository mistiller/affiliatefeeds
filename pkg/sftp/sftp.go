package sftp

import (
	"fmt"
	"net"
	"os"
	"regexp"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	SIZE = 1 << 15
)

// SFTP contains the ssh connection and the sftp client object
type SFTP struct {
	isOpen     bool
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

type callback func(string)

// NewSession initializes an SFTP session object
func NewSession(host, user, password string, port int) (*SFTP, error) {
	var session SFTP
	var err error

	var auths []ssh.AuthMethod
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))

	}
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	config := ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	session.sshClient, err = ssh.Dial("tcp", addr, &config)

	if err != nil {
		return &session, err
	}

	// open an SFTP session over an existing ssh connection.
	session.sftpClient, err = sftp.NewClient(session.sshClient, sftp.MaxPacket(SIZE))
	if err != nil {
		return &session, err
	}

	session.isOpen = true

	return &session, nil
}

// ReadDir returns a list of files in a directory
func (s *SFTP) ReadDir(directory string) ([]os.FileInfo, error) {
	return s.sftpClient.ReadDir(directory)
}

// Walk traverses a directory tree starting from the specified directory
// and applies a callback function to entries that match a regex selector
func (s *SFTP) Walk(root, regex string, cb callback) (err error) {
	if s.isOpen == false {
		return fmt.Errorf("Failed to walk %s - Session not initialized", root)
	}
	w := s.sftpClient.Walk(root)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		matched, err := regexp.Match(regex, []byte(w.Path()))
		if err != nil {
			continue
		}
		if matched != true {
			continue
		}
		cb(w.Path())
	}

	return nil
}

// Remove removes the object specified in path
func (s *SFTP) Remove(path string) error {
	err := s.sftpClient.Remove(path)
	return err
}

// Close closes the ssh and sftp connections
func (s *SFTP) Close() {
	s.sftpClient.Close()
	s.sshClient.Close()

	s.isOpen = false
}
