// Copyright 2020 - Offen Authors <hioffen@posteo.de>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const (
	githubAssetTypeTarball = "tarball"
	versionDevelopment     = "development"
	versionStable          = "stable"
	githubRepo             = "offen/offen"
	storageServer          = "storage.offen.dev"
)

var errNotFound = errors.New("not found")

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/healthz", healthHandler)
	r.HandleFunc("/", redirectHandler)
	r.HandleFunc("/{param1}", redirectHandler)
	r.HandleFunc("/{param1}/{param2}", redirectHandler)
	withRecovery := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(r)

	port := os.Getenv("PORT")
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: withRecovery,
	}

	go srv.ListenAndServe()
	log.Printf("Server now listening on port %s", port)

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	log.Print("Gracefully shut down server")
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	redirect, err := getRedirect(mux.Vars(r))
	if err != nil {
		if err == errNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", redirect)
	w.WriteHeader(http.StatusFound)
}

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
				return fmt.Sprintf("https://%s/deb/offen_%s_amd64.deb", storageServer, version), nil
			}
			return fmt.Sprintf("https://%s/deb/offen_latest_amd64.deb", storageServer), nil
		default:
			// The default behavior is to return the tarball containing binaries
			return fmt.Sprintf("https://%s/binaries/offen-%s.tar.gz", storageServer, param1), nil
		}

	}

	latest, err := getLatestReleaseInfo()
	if err != nil {
		return "", fmt.Errorf("error getting latest release: %w", err)
	}

	asset, assetErr := latest.match(githubAssetTypeTarball)
	if assetErr != nil {
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

func getLatestReleaseInfo() (*releaseInfo, error) {
	res, err := http.Get(
		fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo),
	)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
