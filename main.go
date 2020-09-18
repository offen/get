// Copyright 2020 - Offen Authors <hioffen@posteo.de>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

const githubRepo = "offen/offen"

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.HandleFunc("/{param1}", handler)
	r.HandleFunc("/{param1}/{param2}", handler)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r); err != nil {
		log.Fatalf("error starting server %v", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	redirect, err := getRedirect(vars)
	if err != nil {
		if errors.Is(err, errNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", redirect)
	w.WriteHeader(http.StatusFound)
}

const (
	githubAssetTypeTarball = "tarball"
	versionDevelopment     = "development"
	versionStable          = "stable"
)

var errNotFound = errors.New("not found")

func getRedirect(params map[string]string) (string, error) {
	if param1, ok := params["param1"]; ok {
		switch param1 {
		case "deb":
			// In case the first URL param is `deb` we assume the user wants to
			// download a deb package. No additional parameter will return
			// the asset from the latest GitHub release when a second parameter
			// looks up the specific URL on S3.
			if param2, ok := params["param2"]; ok {
				version := param2
				version = strings.TrimPrefix(version, "v")
				if version == versionDevelopment || version == versionStable {
					return "", errNotFound
				}
				return fmt.Sprintf("https://storage.offen.dev/deb/offen_%s_amd64.deb", version), nil
			}
			return "https://storage.offen.dev/deb/offen_latest_amd64.deb", nil
		default:
			// The default behavior is to return the tarball containing binaries
			return fmt.Sprintf("https://storage.offen.dev/binaries/offen-%s.tar.gz", param1), nil
		}

	}

	latest, err := getLatestReleaseInfo(githubRepo)
	if err != nil {
		return "", fmt.Errorf("error getting latest release: %w", err)
	}

	asset, assetErr := latest.match(githubAssetTypeTarball)
	if assetErr != nil {
		fmt.Println("asset err", assetErr)
		return "", errNotFound
	}
	return asset, nil
}

type releaseInfo struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func (r *releaseInfo) match(pkgType string) (string, error) {
	switch pkgType {
	case "deb":
		for _, asset := range r.Assets {
			if strings.HasSuffix(asset.BrowserDownloadURL, ".deb") {
				return asset.BrowserDownloadURL, nil
			}
		}
	case "tarball":
		for _, asset := range r.Assets {
			if strings.HasSuffix(asset.BrowserDownloadURL, ".tar.gz") {
				return asset.BrowserDownloadURL, nil
			}
		}
	default:
		return "", fmt.Errorf("unknown package type %s", pkgType)
	}
	return "", fmt.Errorf("requested release did not contain an asset for %s", pkgType)
}

func getLatestReleaseInfo(repo string) (*releaseInfo, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	res, err := http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error on HTTP request: %w", err)
	}
	defer res.Body.Close()

	var responseBody releaseInfo
	if err := json.NewDecoder(res.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}
	return &responseBody, nil
}
