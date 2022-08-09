//go:build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var models = []string{}

const (
	modelsPath = "./gen/models/"
)

func main() {
	for _, model := range models {
		f := strings.Split(model, "/")
		path := filepath.Join(modelsPath, f[len(f)-1])
		if _, err := os.Stat(path); err != nil {
			err := downloadModel(path, model)
			if err != nil {
				panic(err)
			}
			fmt.Println("Downloaded model: " + path)
		}
	}
}

func downloadModel(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
