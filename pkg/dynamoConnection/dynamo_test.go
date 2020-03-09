// +build !unit
// +build integration

package dynamoconnection

import (
	"testing"

	"stillgrove.com/gofeedyourself/pkg/feedservice/config"
	"stillgrove.com/gofeedyourself/pkg/feedservice/helpers"
)

func TestDynamoDB(t *testing.T) {
	configPath := helpers.FindFolderDir("gofeedyourself") + "/config/config.se.dev.yaml"
	cfg, err := config.New(configPath)
	if err != nil {
		t.Fatal(err)
	}

	id, secret, table, err := cfg.GetDynamo()
	if err != nil {
		t.Fatal(err)
	}
	_, err = InitDynamoConnection(id, secret, table)
	if err != nil {
		t.Fatal(err)
	}
}
