// +build unit
// +build !integration

package sftp

import (
	"os"
	"strconv"
	"testing"
)

func TestSFTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	port, err := strconv.Atoi(os.Getenv("FTP_PORT"))
	if err != nil {
		t.Fatalf("Var not found -%v", err)
	}
	sess, err := NewSession(
		os.Getenv("FTP_HOST"),
		os.Getenv("FTP_USER"),
		os.Getenv("FTP_PASS"),
		port,
	)
	if err != nil {
		t.Fatalf("Connect to FTP -%v", err)
	}
	sess.Close()
}
