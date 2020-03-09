// +build unit
// +build !integration

package zip

import "testing"

func TestZip(t *testing.T) {
	message := "Hello!"

	cypher, err := Zip([]byte(message))
	if err != nil {
		t.Fatalf("%v", err)
	}
	decypher, err := Unzip(cypher)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if string(decypher) != message {
		t.Fatalf("Message was scrambled! %s", string(decypher))
	}
}
