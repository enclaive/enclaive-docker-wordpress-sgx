package main

import (
	"archive/zip"
	"io"
	"os"
	"path"
)

func ExtractAppZip() {
	err := os.Mkdir(basePath, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	err = os.Chdir(basePath)
	check(err)

	r, err := zip.OpenReader("/app/app.zip")
	check(err)

	for _, f := range r.File {

		//fmt.Println("extracting", f.Name)

		fi := f.FileInfo()
		path.Dir(f.Name)
		if fi.IsDir() {
			check(os.MkdirAll(f.Name, os.ModePerm))
			continue
		}

		fr, err := f.Open()
		check(err)

		fw, err := os.Create(f.Name)
		if os.IsNotExist(err) {
			check(os.MkdirAll(path.Dir(f.Name), os.ModePerm))
			fw, err = os.Create(f.Name)
		}
		check(err)

		_, err = io.Copy(fw, fr)
		check(err)

		check(fr.Close())
		check(fw.Close())
	}
}
