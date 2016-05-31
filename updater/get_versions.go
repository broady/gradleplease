package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/blang/semver"
	"github.com/cheggaaa/pb"
)

const (
	baseURL = "http://dl-ssl.google.com/android/repository/"
)

type addons struct {
	AddOns []struct {
		Archives struct {
			Archive struct {
				Size     int    `xml:"size"`
				Checksum string `xml:"checksum"`
				URL      string `xml:"url"`
			} `xml:"archive"`
		} `xml:"archives"`
	} `xml:"extra"`
}

func getVersions() (*VersionInfo, error) {
	f, err := getFile("addon.xml", false)
	if err != nil {
		return nil, err
	}
	a := &addons{}
	if err := xml.Unmarshal(f, a); err != nil {
		return nil, err
	}

	var supportURL, googleURL string
	for _, addon := range a.AddOns {
		url := addon.Archives.Archive.URL
		if strings.HasPrefix(url, "android_m2repository") {
			supportURL = url
		}
		if strings.HasPrefix(url, "google_m2repository") {
			googleURL = url
		}
	}

	if supportURL == "" {
		return nil, errors.New("couldn't find support repo URL")
	}
	if googleURL == "" {
		return nil, errors.New("couldn't find google repo URL")
	}

	support, err := getFile(supportURL, true)
	if err != nil {
		return nil, err
	}
	google, err := getFile(googleURL, true)
	if err != nil {
		return nil, err
	}

	v := &VersionInfo{}
	v.PlayServicesVersion, err = getLatestVersion(google, "play-services-base", "aar")
	if err != nil {
		return nil, err
	}
	v.AndroidSupportVersion, err = getLatestVersion(support, "support-v13", "aar")
	return v, err
}

var versionsToIgnore = map[string]bool{
	"alpha1": true,
	"alpha2": true,
	"beta1":  true,
}

func getLatestVersion(zipFile []byte, prefix, suffix string) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(zipFile), int64(len(zipFile)))
	if err != nil {
		return "", err
	}
	latest := ""
	for _, f := range r.File {
		artifact := path.Base(f.Name)
		if strings.HasPrefix(artifact, prefix) && strings.HasSuffix(artifact, suffix) {
			v := extractVersion(artifact)
			if _, ok := versionsToIgnore[v]; ok {
				continue
			}
			if latest == "" {
				latest = v
			}
			v1, err := semver.Parse(v)
			if err != nil {
				return "", fmt.Errorf("%q %v", v, err)
			}
			l := semver.MustParse(latest)
			if v1.GT(l) {
				latest = v
			}
		}
	}
	return latest, nil
}

func extractVersion(f string) string {
	i := strings.LastIndex(f, "-") + 1
	return f[i : len(f)-len(path.Ext(f))]
}

func getFile(filename string, shouldCache bool) ([]byte, error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err == nil {
		return ioutil.ReadAll(f)
	}
	log.Printf("fetching %s", filename)
	resp, err := http.Get(baseURL + filename)
	if err != nil {
		return nil, err
	}
	bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES)
	bar.ShowSpeed = true
	bar.ShowTimeLeft = true
	bar.Start()
	b, err := ioutil.ReadAll(bar.NewProxyReader(resp.Body))
	if err != nil {
		return nil, err
	}
	if shouldCache {
		log.Printf("creating file: %s", filename)
		f, err = os.Create(filename)
		defer f.Close()
		if err != nil {
			return nil, err
		}
		log.Printf("writing cached file: %s", filename)
		_, err = f.Write(b)
		if err != nil {
			// Try to remove failed write.
			os.Remove(filename)
			return nil, err
		}
	}
	return b, err
}
