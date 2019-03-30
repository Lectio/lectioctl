package main

import "os"

// CreateDirIfNotExist creates a path if it does not exist. It is similar to mkdir -p in shell command,
// which also creates parent directory if not exists.
func CreateDirIfNotExist(dir string) (bool, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		return true, err
	}
	return false, nil
}
