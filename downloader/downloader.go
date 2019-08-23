package downloader

import (
	"bytes"
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"github.com/twatzl/webdav-downloader/webdav"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type webdavDownloader struct {
	logger *logrus.Logger
	client *http.Client
	cfg    *Config
}

func DownloadDir(conf *Config, directory string) {
	w := webdavDownloader{
		logger: logrus.StandardLogger(),
		client: &http.Client{},
		cfg:    conf,
	}

	//testConnection(logger, server, user, password, client)

	localDirs := []string{directory}
	dirsToSearch := []string{directory}
	var filesFound []string

	for len(dirsToSearch) > 0 {
		currentDir := dirsToSearch[0]
		dirsToSearch = dirsToSearch[1:]

		dirsFound, files := w.listDirectory(currentDir)
		dirsToSearch = append(dirsToSearch, dirsFound...)
		localDirs = append(localDirs, dirsFound...)
		filesFound = append(filesFound, files...)
	}

	w.logger.WithField("filesFound", len(filesFound)).Infoln()

	for _, dir := range localDirs {
		dir = strings.Trim(dir, "/")
		err := os.Mkdir(dir, 0755)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			w.logger.WithError(err).Fatal("could not create local dirs")
		}
	}

	for _, file := range filesFound {
		w.downloadResource(file)
	}

}

func (w *webdavDownloader) getBaseUrl() string {
	return w.cfg.Protocol + "://" + path.Join(w.cfg.Host, w.cfg.BaseDir) + "/"
}

func (w *webdavDownloader) authenticateRequest(req *http.Request) {
	req.SetBasicAuth(w.cfg.User, w.cfg.Pass)
}

func (w *webdavDownloader) listDirectory(directory string) (dirs, files []string) {
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

	// process responses
	for _, resp := range webdavMultistatus.Responses {
		dirPath, err := url.PathUnescape(resp.Href)
		if err != nil {
			w.logger.WithField("path", resp.Href).WithError(err).Errorln("could not decode path")
			continue
		}
		dirPath = strings.Trim(dirPath, "/")
		baseDir := strings.Trim(w.cfg.BaseDir, "/")
		dirPath = strings.TrimPrefix(dirPath, baseDir)
		dirPath = strings.Trim(dirPath, "/")

		// for some reason the response also gives us the directory itself
		// we have to filter it out
		if strings.HasSuffix(dirPath, directory) {
			continue
		}

		if resp.Props.Prop.ResourceType.Collection == nil {
			files = append(files, dirPath)
		} else {
			dirs = append(dirs, dirPath)
		}
	}

	w.logger.Infoln(dirs)

	return dirs, files
}

func (w *webdavDownloader) downloadResource(resourcePath string) {
	url := w.getDirectoryUrl(resourcePath)
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

	err = ioutil.WriteFile(resourcePath, data, 0755)
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
