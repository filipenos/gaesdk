package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	URL_VERSION = "https://storage.googleapis.com/appengine-sdks/featured/VERSION"
	URL_SDK     = "https://storage.googleapis.com/appengine-sdks/featured/go_appengine_sdk_linux_amd64-%s.zip"
	TEMP_FILE   = "/tmp/go_appengine.zip"
)

func main() {

	fmt.Println("Google Appengine SDK Download")
	v := getVersion()
	fmt.Printf("Found version %s\n", v)
	fmt.Println("Downloading...")
	download(v)
	fmt.Println("Extracting new version")
	unzip(TEMP_FILE)
	fmt.Println("Done")
}

func getVersion() string {
	resp, err := http.Get(URL_VERSION)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		s := strings.Split(l, ":")
		if s[0] == "release" {
			reg := regexp.MustCompile("[^0-9//.]+")
			return reg.ReplaceAllString(s[1], "")
		}
	}
	return ""
}

func download(version string) {
	resp, err := http.Get(fmt.Sprintf(URL_SDK, version))
	if err != nil {
		panic(err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(TEMP_FILE)
	if err != nil {
		panic(err)
		return
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
		return
	}
	fmt.Printf("Total of bytes %v\n", n)

	err = unzip(TEMP_FILE)
	if err != nil {
		panic(err)
		return
	}
}

func unzip(zipfile string) error {
	reader, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.Reader.File {
		zipped, err := f.Open()
		if err != nil {
			return err
		}
		defer zipped.Close()

		// get the individual file name and extract the current directory
		path := filepath.Join("./", f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			//fmt.Println("Creating directory", path)
		} else {
			writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, f.Mode())
			if err != nil {
				return err
			}

			defer writer.Close()

			if _, err = io.Copy(writer, zipped); err != nil {
				return err
			}
			//fmt.Println("Decompressing : ", path)
		}
	}
	return nil
}
