package files

import (
	"io"
	"mime/multipart"
	"os"
)

func SaveToDisk(file multipart.File, filename string) (string, error) {
	path := "uploads/" + filename

	os.MkdirAll("uploads", os.ModePerm)

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return path, nil
}
