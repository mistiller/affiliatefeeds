// +build !unit
// +build !integration

package feedservice

import (
	"log"
	"strings"
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/config"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
	"stillgrove.com/gofeedyourself/pkg/sftp"
)

func TestPipeline(t *testing.T) {
	if true {
		t.Skip("skipping test")
	}

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	p, err := New(cfg, "woocoemmerce", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	p.Run(false, false)
}

func TestPurge(t *testing.T) {
	if true {
		t.Skip("skipping testing in short mode")
	}
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	const path = "/home/stillgrove/stillgrove.com/wp-content/uploads"
	var exceptions = [...]string{
		"/assets/",
		"SGMA-",
		"STILLGROVE-",
	}

	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	host, port, user, password, err := cfg.GetFTP()
	if err != nil {
		t.Fatalf("Clearing image assets - %v", err)
	}
	sess, err := sftp.NewSession(host, user, password, port)
	if err != nil {
		t.Fatalf("Cleaning image assets - %v", err)
	}
	defer sess.Close()

	files, err := sess.ReadDir("/home/stillgrove/stillgrove.com/wp-content/uploads")
	if err != nil {
		t.Fatalf("Cleaning image assets - %v", err)
	}

	var matched bool
	var name string
	for i := range files {
		matched = false
		if files[i].IsDir() == true {
			continue
		}
		name = files[i].Name()
		for j := range exceptions {
			matched = strings.Contains(name, exceptions[j])
			if matched == true {
				break
			}
		}
		if matched == true {
			log.Println(path + "/" + name)
			continue
		}
	}
}
