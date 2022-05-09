package main

import (
    "archive/zip"
    "fmt"
    "io"
    "os"
)


func ExtractAppZip() error {
    err := os.Chdir("/app/")
    if err != nil { panic(err) }

    r, err := zip.OpenReader("/app/app.zip")
    if err != nil { panic(err) }

	for _, f := range r.File {

		fmt.Println("extracting", f.Name)

        fi := f.FileInfo()
        if fi.IsDir() {
            os.MkdirAll(f.Name, os.ModePerm)
            continue;
        }

		fr, err := f.Open()
		if err != nil { panic(err) }

        fw, err := os.Create(f.Name);
		if err != nil { panic(err) }

		_, err = io.Copy(fw, fr)
		if err != nil { panic(err) }

		fr.Close()
		fw.Close()
	}

    return nil
}
