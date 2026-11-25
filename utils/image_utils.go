package utils

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"os"

	"golang.org/x/image/webp"
)

func NormalizePage(src io.Reader, destPath string) error {
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	rest := io.MultiReader(bytes.NewReader(head[:n]), src)

	if http.DetectContentType(head) == "image/webp" {
		img, err := webp.Decode(rest)
		if err != nil {
			return err
		}

		file, err := os.Create(destPath)
		if err != nil {
			return err
		}

		defer HandleClose(file.Close)

		return jpeg.Encode(file, img, &jpeg.Options{Quality: 100})
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}

	defer HandleClose(file.Close)

	_, err = io.Copy(file, rest)
	return err
}
