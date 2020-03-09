package zip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
)

// Zip returns a compressed byte slice of payload
func Zip(payload []byte) (compressed []byte, err error) {
	var handle bytes.Buffer

	zipWriter, err := gzip.NewWriterLevel(&handle, 9)
	if err != nil {
		return nil, err
	}

	_, err = zipWriter.Write(payload)
	if err != nil {
		return nil, err
	}

	if err = zipWriter.Close(); err != nil {
		return nil, err
	}

	compressed = handle.Bytes()

	return compressed, nil
}

// Unzip returns an uncompressed byte slice of compressed
func Unzip(compressed []byte) (unzipped []byte, err error) {
	handle := bytes.NewReader(compressed)
	zipReader, err := gzip.NewReader(handle)
	if err != nil {
		fmt.Println("[ERROR] New gzip reader:", err)
	}
	defer zipReader.Close()

	unzipped, err = ioutil.ReadAll(zipReader)

	return unzipped, err
}
