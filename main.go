package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

var (
	version, install string
	override         bool
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&version, "version", "latest", "Version of App Engine SDK")
	flag.StringVar(&install, "install", pwd, "Directory to install sdk")
	flag.BoolVar(&override, "override", false, "Force to override installation")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	log.Println("Google Appengine SDK Manager")
	if version == "" || version == "latest" {
		log.Println("Searching latest version of sdk")
		if err := getVersion(); err != nil {
			log.Fatal(err)
		}
		log.Println("Found version:", version)
	} else {
		log.Println("Using:", version)
	}

	local, err := verifyVersion()
	if err != nil {
		log.Fatal(err)
	}
	if local == "" {
		log.Printf("No versions found in %s/\n", install)
	} else if local == version && !override {
		log.Printf("You are already using the latest version %s at %s\n", local, install)
		return
	} else {
		log.Printf("Found version %s installed in %s\n", local, install)
		log.Println("Backing up your old version")
		err = os.Rename(install+"/go_appengine", install+"/go_appengine-"+local)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Downloading...")
	downloadAndExtract()
	log.Println("Done")
}

func getVersion() error {
	resp, err := http.Get(URL_VERSION)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	version, err = readVersion(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func verifyVersion() (string, error) {
	file, err := os.Open(install + "/go_appengine/VERSION")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	v, err := readVersion(file)
	if err != nil {
		return "", err
	}
	return v, nil
}

func readVersion(reader io.ReadCloser) (string, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		s := strings.Split(l, ":")
		if s[0] == "release" {
			reg := regexp.MustCompile("[^0-9//.]+")
			return reg.ReplaceAllString(s[1], ""), nil
		}
	}
	return "", fmt.Errorf("Not found")
}

func downloadAndExtract() error {
	resp, err := http.Get(fmt.Sprintf(URL_SDK, version))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := ioutil.TempFile(os.TempDir(), "go_appengine_")
	if err != nil {
		return err
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	_, err = out.Seek(0, 0)
	if err != nil {
		return err
	}
	return unzip(out, size)
}

func unzip(r io.ReaderAt, size int64) error {
	reader, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}

	for _, f := range reader.File {
		zipped, err := f.Open()
		if err != nil {
			return err
		}
		defer zipped.Close()

		// get the individual file name and extract the current directory
		path := filepath.Join(install, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			fmt.Println("creating:", path)
		} else {
			os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm)
			writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, f.Mode())
			if err != nil {
				return err
			}

			defer writer.Close()

			if _, err = io.Copy(writer, zipped); err != nil {
				return err
			}
			fmt.Println("inflating:", path)
		}
	}
	return nil
}
