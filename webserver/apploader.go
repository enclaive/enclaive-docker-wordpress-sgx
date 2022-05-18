package main

import (
	"archive/zip"
	"io"
	"os"
)

func ExtractAppZip() error {
	err := os.Mkdir(basePath, 0755)
	if !os.IsExist(err) {
		panic(err)
	}

	err = os.Chdir(basePath)
	check(err)

	r, err := zip.OpenReader("/app/app.zip")
	check(err)

	for _, f := range r.File {

		//fmt.Println("extracting", f.Name)

		fi := f.FileInfo()
		if fi.IsDir() {
			os.MkdirAll(f.Name, os.ModePerm)
			continue
		}

		fr, err := f.Open()
		check(err)

		fw, err := os.Create(f.Name)
		check(err)

		_, err = io.Copy(fw, fr)
		check(err)

		fr.Close()
		fw.Close()
	}

	return nil
}
