package fs

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	FILE_APPEND = os.O_APPEND
)

func FileGetContents(file_path string) (data []byte, err error) {
	f, err := os.Open(file_path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	bf, err1 := ioutil.ReadAll(f)
	if err1 != nil {
		return nil, err1
	}
	return bf, nil
}

func FilePutContents(file_path string, data []byte, def ...int) error {
	flags := os.O_RDWR | os.O_CREATE
	is_append := false
	if len(def) > 0 && def[0] == FILE_APPEND {
		is_append = true
		flags = flags | os.O_APPEND
	}
	f, err := os.OpenFile(file_path, flags, 0644)
	defer f.Close()
	if err != nil {
		return err
	}
	if is_append {
		stat, _ := f.Stat()
		f.WriteAt(data, stat.Size())
	} else {
		f.Truncate(0)
		f.Write(data)
	}
	return nil
}

func FileExists(file_path string) bool {
	_, err := os.Stat(file_path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func File_Md5(file_path string) (string, error) {
	file, err := os.Open(file_path)
	if err == nil {
		h := md5.New()
		io.Copy(h, file)
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	}
	return "", err
}

// DirCheck create dir if not exists
func DirCheck(filePath string) error {
	dir := filepath.Dir(filePath)
	if !FileExists(dir) {
		return os.MkdirAll(dir, 0777)
	}
	return nil
}
