package downloader

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/twatzl/webdav-downloader/webdav"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const DELTA_FLAG_SIZE = "SIZE"
const DELTA_FLAG_DATE = "DATE"

type webdavDownloader struct {
	logger    *logrus.Logger
	client    *http.Client
	cfg       *Config
	remoteDir string
}

// DownloadDir will download the files and folders at remoteDir recursively.
// remoteDir is a relative path to the remote folder
func DownloadDir(conf *Config, remoteDir string) {
	w := webdavDownloader{
		logger:    logrus.StandardLogger(),
		client:    &http.Client{},
		cfg:       conf,
		remoteDir: remoteDir,
	}

	//w.logger.SetLevel(logrus.DebugLevel)
	//testConnection(logger, server, user, password, client)

	var localDirs []string
	dirsToSearch := []string{remoteDir}
	var filesFound []string

	// list remote directories
	for len(dirsToSearch) > 0 {
		currentDir := dirsToSearch[0]
		dirsToSearch = dirsToSearch[1:]

		dirsFound, files := w.crawlFilesInDir(currentDir)
		if dirsFound == nil && files == nil {
			// TODO: maybe log error here
			continue
		}

		dirsToSearch = append(dirsToSearch, dirsFound...)
		localDirs = append(localDirs, dirsFound...)
		filesFound = append(filesFound, files...)
	}

	w.logger.WithField("filesFound", len(filesFound)).Infoln()

	localDirs = append(localDirs, ".")

	// create dirs
	for _, dir := range localDirs {
		dir = w.remotePathToLocalPath(dir)
		if dir == "" || dir == w.cfg.LocalDir {
			continue
		}

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			if os.IsExist(err) {
				w.logger.WithError(err).Info("directory exists")
				continue
			}
			w.logger.WithError(err).Fatal("could not create local dirs")
		}
	}

	for _, file := range filesFound {
		w.downloadResource(file)
	}

}

func (w *webdavDownloader) authenticateRequest(req *http.Request) {
	req.SetBasicAuth(w.cfg.User, w.cfg.Pass)
}

func (w *webdavDownloader) crawlFilesInDir(directory string) (dirs, files []string) {
	directoryUrl := w.getDirectoryUrl(directory)
	req, err := http.NewRequest("PROPFIND", directoryUrl, bytes.NewReader([]byte{}))
	if err != nil {
		w.logger.Fatal(err)
	}

	req.Header.Add("Content-type", "application/xml")
	req.Header.Add("Depth", "1")
	w.authenticateRequest(req)

	w.logRequest(req)
	res, err := w.client.Do(req)
	if err != nil {
		w.logger.WithError(err).Errorln("error while sending request")
	}

	if res.StatusCode != 207 {
		w.logger.WithField("code", res.StatusCode).WithField("status", res.Status).Errorln("got wrong response code. expected 207")
		return nil, nil
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.logger.WithError(err).Errorln("error while reading response body")
		return nil, nil
	}

	// print raw xml response
	w.logger.Debugln(string(data))

	// parse xml
	var webdavMultistatus webdav.Multistatus
	dec := xml.NewDecoder(bytes.NewBuffer(data))
	err = dec.Decode(&webdavMultistatus)
	if err != nil {
		w.logger.WithError(err).Errorln("error decoding xml response from server")
	}

	dirs = []string{}
	files = []string{}

	// process responses
	for _, resp := range webdavMultistatus.Responses {
		resourcePath, err := url.PathUnescape(resp.Href)
		if err != nil {
			w.logger.WithField("path", resp.Href).WithError(err).Errorln("could not decode path")
			continue
		}
		resourcePath = strings.Trim(resourcePath, "/")
		baseDir := strings.Trim(w.cfg.BaseDir, "/")
		resourcePath = strings.TrimPrefix(resourcePath, baseDir)
		resourcePath = strings.Trim(resourcePath, "/")

		// for some reason the response also gives us the directory itself
		// we have to filter it out
		if strings.HasSuffix(resourcePath, directory) {
			continue
		}

		if resp.Props.Prop.ResourceType.Collection == nil {
			localPath := w.remotePathToLocalPath(resourcePath)
			skip := w.shouldSkipFileInDeltaMode(localPath, resp)
			if skip {
				continue
			}

			files = append(files, resourcePath)
		} else {
			dirs = append(dirs, resourcePath)
		}
	}
	w.logger.Infoln(dirs)

	return dirs, files
}

func (w *webdavDownloader) shouldSkipFileInDeltaMode(localPath string, resp webdav.Response) bool {
	info, err := os.Stat(localPath)

	if os.IsNotExist(err) {
		return false
	}

	if w.cfg.DeltaMode {
		var reasons []string
		if len(w.cfg.DeltaFlags) == 0 {
			w.logger.WithField("localPath", localPath).Infoln("delta mode: skipped file. reason: file exists.")
			return true
		}
		if w.cfg.DeltaFlags[DELTA_FLAG_SIZE] && info.Size() == resp.Props.Prop.ContentLength {
			reasons = append(reasons, DELTA_FLAG_SIZE)
		}
		if w.cfg.DeltaFlags[DELTA_FLAG_DATE] && info.ModTime() == resp.Props.Prop.GetLastModifiedTime() {
			reasons = append(reasons, DELTA_FLAG_DATE)
		}

		if len(reasons) > 0 {
			message := fmt.Sprintf("delta mode: skipped file. reasons: [%s]\n", strings.Join(reasons, ","))
			w.logger.WithField("localPath", localPath).Infoln(message)
			return true
		}
	}
	return false
}

func (w *webdavDownloader) downloadResource(resourcePath string) {
	url := w.getDirectoryUrl(resourcePath)
	localPath := w.remotePathToLocalPath(resourcePath)

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		w.logger.Fatal(err)
	}

	req.Header.Add("Content-type", "application/xml")
	//req.Header.Add("Depth", "1")
	w.authenticateRequest(req)

	w.logRequest(req)
	res, err := w.client.Do(req)
	if err != nil {
		w.logger.WithError(err).WithField("url", req.URL).Errorln("error sending request")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.logger.WithError(err).WithField("resource", resourcePath).Errorln("error while reading response body for resource")
	}

	err = ioutil.WriteFile(localPath, data, 0755)
	if err != nil {
		w.logger.WithError(err).WithField("resource", resourcePath).Errorln("error while writing downloaded file")
	}

	w.logger.WithField("resource", resourcePath).Infoln("resource downloaded")
}

func (w *webdavDownloader) getDirectoryUrl(directory string) string {
	return w.cfg.Protocol + "://" + path.Join(w.cfg.Host, w.cfg.BaseDir, directory)
}

func (w *webdavDownloader) logRequest(request *http.Request) {
	w.logger.WithField("url", request.URL).Infoln("sending request")
}

func (w *webdavDownloader) getLocalDir(dir string) string {
	return path.Join(w.cfg.LocalDir, dir)
}

func (w *webdavDownloader) remotePathToLocalPath(dir string) string {
	dir = strings.TrimPrefix(dir, w.remoteDir) // remove base path
	dir = strings.Trim(dir, "/") // remove remaining slashes
	dir = path.Join(w.cfg.LocalDir, dir) // add local dir
	return dir
}

func testConnection(logger *logrus.Logger, server string, user string, password string, client *http.Client) {
	logger.Infoln("testing connection to server")
	req, err := http.NewRequest(http.MethodGet, server, bytes.NewReader([]byte{}))
	if err != nil {
		logger.Fatal(err)
	}
	req.SetBasicAuth(user, password)
	response, err := client.Do(req)
	if err != nil {
		logger.Fatal(err)
	}
	if response.StatusCode == 200 {
		logger.Infoln("connection ok")
	} else {
		logger.WithField("status", response.Status).WithField("code", response.StatusCode).Errorln("connection failed")
	}
}
