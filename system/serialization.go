package system

import (
	"encoding/gob"
	"os"
)

// DumpGob : serialize and dump to disk given object to give filePath path
func DumpGob(filePath string, object interface{}) error {
	file, err := os.Create(filePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// FetchGob : load previously dumped object from filePath to given object
func FetchGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}
