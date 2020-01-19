package upstream

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

	"github.com/streambinder/spotitube/system"
)

const (
	cacheGob      = "/tmp/upstream.gob"
	cacheDuration = 24 * time.Hour

	repositoryURI  = "https://github.com/streambinder/spotitube"
	upstreamAPI    = "https://api.github.com/repos/streambinder/spotitube/releases/latest"
	upstreamAPIURI = repositoryURI + "/releases/latest"
)

// Check contains last time an upstream version check has been done
type Check struct {
	Version int
	Time    time.Time
}

type gitHubRelease struct {
	Name string `json:"name"`
}

// Version returns latest Spotitube version by interacting with GitHub release APIs
func Version() (int, error) {
	last := new(Check)
	if err := system.FetchGob(cacheGob, last); err == nil && time.Since(last.Time) < cacheDuration {
		return last.Version, nil
	}

	req, err := http.NewRequest(http.MethodGet, upstreamAPI, nil)
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

	rel := new(gitHubRelease)
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

	system.DumpGob(cacheGob, Check{Version: v, Time: time.Now()})
	return v, nil
}

// Download downloads latest Spotitube version binary into given path
func Download(path string) error {
	req, err := http.NewRequest(http.MethodGet, upstreamAPI, nil)
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

	if system.FileExists(usrBinaryTemporary(path)) {
		os.Remove(usrBinaryTemporary(path))
	}

	f, err := os.Create(usrBinaryTemporary(path))
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

	os.Remove(path)
	if err := system.FileCopy(usrBinaryTemporary(path), path); err != nil {
		return err
	}
	os.Remove(usrBinaryTemporary(path))

	if err := os.Chmod(path, 0755); err != nil {
		return err
	}

	return nil
}

func usrBinaryTemporary(path string) string {
	return fmt.Sprintf("%s.tmp", path)
}
