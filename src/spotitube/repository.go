package spotitube

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"../system"
)

const (
	// RepositoryURI indicates repository URI
	RepositoryURI = "https://github.com/streambinder/spotitube"
	// RepositoryUpstreamURI indicates repository latest version URI
	RepositoryUpstreamURI = RepositoryURI + "/releases/latest"
	// RepositoryUpstreamAPI indicates repository latest version API URI
	RepositoryUpstreamAPI = "https://api.github.com/repos/streambinder/spotitube/releases/latest"
)

var (
	// UserUpstreamCheckGob is the path in which to read/write last time
	// the application checked for an upstream version
	UserUpstreamCheckGob = fmt.Sprintf("%s/upstream.gob", UserPath())
	// UserBinaryTemporary is the path in which the upstrem version gets downloaded to
	UserBinaryTemporary = fmt.Sprintf("%s/spotitube.tmp", UserPath())
)

// UpstreamCheck contains last time an upstream version check has been done
type UpstreamCheck struct {
	Version int
	Time    time.Time
}

// GitHubRelease maps essential GitHub release API objects fields needed in the application flow
type GitHubRelease struct {
	Name string `json:"name"`
}

// UpstreamVersion returns latest Spotitube version by interacting with GitHub release APIs
func UpstreamVersion() (int, error) {
	last := new(UpstreamCheck)
	if err := system.FetchGob(UserUpstreamCheckGob, last); err == nil && time.Since(last.Time).Hours() < 24 {
		return last.Version, nil
	}

	req, err := http.NewRequest(http.MethodGet, RepositoryUpstreamAPI, nil)
	if err != nil {
		return 0, err
	}

	res, err := system.Client.Do(req)
	if err != nil {
		return 0, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	rel := new(GitHubRelease)
	if err := json.Unmarshal(body, rel); err != nil {
		return 0, err
	}

	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		return 0, err
	}

	v, err := strconv.Atoi(reg.ReplaceAllString(rel.Name, ""))
	if err != nil {
		return 0, err
	}

	system.DumpGob(UserUpstreamCheckGob, UpstreamCheck{Version: v, Time: time.Now()})
	return v, nil
}

// UpstreamDownload downloads latest Spotitube version binary into given path
func UpstreamDownload(path string) error {
	req, err := http.NewRequest(http.MethodGet, RepositoryUpstreamAPI, nil)
	if err != nil {
		return err
	}

	res, err := system.Client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	rel := make(map[string]interface{})
	if err := json.Unmarshal([]byte(body), &rel); err != nil {
		return err
	}

	binURL := ""
	for _, a := range rel["assets"].([]interface{}) {
		binName := a.(map[string]interface{})["name"].(string)
		if (runtime.GOOS == "windows" && filepath.Ext(binName) == ".exe") ||
			filepath.Ext(binName) == ".bin" {
			binURL = a.(map[string]interface{})["browser_download_url"].(string)
			break
		}
	}
	if len(binURL) == 0 {
		return fmt.Errorf("Upstream binary asset not found")
	}

	if system.FileExists(UserBinaryTemporary) {
		os.Remove(UserBinaryTemporary)
	}

	f, err := os.Create(UserBinaryTemporary)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := http.Get(binURL)
	if err != nil {
		return err
	}
	defer b.Body.Close()

	if _, err := io.Copy(f, b.Body); err != nil {
		return err
	}

	os.Remove(UserBinary)
	if err := system.FileCopy(UserBinaryTemporary, UserBinary); err != nil {
		return err
	}
	os.Remove(UserBinaryTemporary)

	if err := os.Chmod(UserBinary, 0755); err != nil {
		return err
	}

	return nil
}
