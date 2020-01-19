package system

import (
	"encoding/gob"
	"os"
)

// DumpGob serializes and dumps to disk given object to given path
func DumpGob(path string, object interface{}) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// FetchGob loads dumped object from given file to given object
func FetchGob(path string, object interface{}) error {
	file, err := os.Open(path)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}
