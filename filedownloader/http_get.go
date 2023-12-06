package filedownloader

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type HTTPGetFile struct {
	URL string
}

func NewHTTPGetFile(url string) *HTTPGetFile {
	return &HTTPGetFile{
		URL: url,
	}
}

func (h *HTTPGetFile) Download(to string) error {
	res, err := http.Get(h.URL)
	if err != nil {
		return fmt.Errorf("failed to download support packet: %w", err)
	}
	defer res.Body.Close()

	// create all dirs required for the file
	err = os.MkdirAll(filepath.Dir(to), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.ReadFrom(res.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
