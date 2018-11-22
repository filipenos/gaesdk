package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

const (
	URL_VERSION = "https://storage.googleapis.com/appengine-sdks/featured/VERSION"
	URL_STORAGE = "https://storage.googleapis.com/appengine-sdks/"
	FILENAME    = "go_appengine_sdk_linux_amd64-%s.zip"
)

var (
	version, install string
	listRemote       bool
)

type RemoteSDK struct {
	XMLName     xml.Name
	Text        string
	Xmlns       string
	Name        string
	Prefix      string
	Marker      string
	NextMarker  string
	IsTruncated string
	Contents    []struct {
		Text           string
		Key            string
		Generation     string
		MetaGeneration string
		LastModified   string
		ETag           string
		Size           string
	}
}

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&version, "version", "latest", "Version of App Engine SDK")
	flag.StringVar(&install, "install", pwd, "Directory to install sdk")
	flag.BoolVar(&listRemote, "list-remote", false, "List remote versions")
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
		if err := remoteVersion(); err != nil {
			log.Fatal(err)
		}
		log.Println("Found version:", version)
	} else {
		log.Println("Using:", version)
	}

	remoteVersions, err := getRemoveVersions()
	if err != nil {
		log.Panic("Error on get remove versions: %v", err)
	}

	if listRemote {
		log.Println("Listing remote versions")
		for _, v := range remoteVersions {
			log.Println(v)
		}
	}

	found := false
	for _, v := range remoteVersions {
		if v == version {
			found = true
		}
	}
	if !found {
		log.Fatalf("Version %s not found on server", version)
	}

	local, err := localVersion()
	if err != nil {
		log.Fatal(err)
	}
	if local == "" {
		log.Printf("No versions found in %s/\n", install)
	} else if local == version {
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
	tempFile, err := download(version)
	if err != nil {
		log.Fatal(err)
	}

	err = extract(tempFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}

func remoteVersion() error {
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

func localVersion() (string, error) {
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

func download(version string) (string, error) {
	filename := fmt.Sprintf(FILENAME, version)
	log.Printf("Downloading file %s", filename)

	url := fmt.Sprintf("%sfeatured/%s", URL_STORAGE, filename)
	log.Printf("Downloading file from %s", url)

	tempFile := fmt.Sprintf("%s/%s", os.TempDir(), filename)
	log.Printf("Saving on %s", tempFile)

	cmd := exec.Command("wget", url, "-O", tempFile)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Error get file: %v, %s", err, string(b))
	}

	return tempFile, nil
}

func extract(tempFile string) error {
	log.Printf("Extracting file %s", tempFile)

	cmd := exec.Command("unzip", tempFile, "-d", install)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error on unzip files: %v, %s", err, string(b))
	}

	return nil
}

func getRemoveVersions() ([]string, error) {
	versions := make([]string, 0, 0)

	resp, err := http.Get(URL_STORAGE)
	if err != nil {
		return versions, err
	}
	defer resp.Body.Close()

	var r RemoteSDK
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return versions, err
	}

	re := regexp.MustCompile("amd64-([0-9.]+).zip")
	for _, c := range r.Contents {
		featured := strings.Contains(c.Key, "featured")
		if featured && strings.Contains(c.Key, "go_appengine_sdk_linux_amd64-") {
			subs := re.FindStringSubmatch(c.Key)
			if len(subs) == 2 {
				versions = append(versions, strings.TrimSpace(subs[1]))
			}
		}
	}

	sort.Strings(versions)

	return versions, nil
}
