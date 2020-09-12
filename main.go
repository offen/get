// Copyright 2020 - Offen Authors <hioffen@posteo.de>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

const (
	githubAssetTypeTarball = "tarball"
	versionDevelopment     = "development"
	versionStable          = "stable"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if param1, ok := request.PathParameters["param1"]; ok {
		switch param1 {
		case "deb":
			// In case the first URL param is `deb` we assume the user wants to
			// download a deb package. No additional parameter will return
			// the asset from the latest GitHub release when a second parameter
			// looks up the specific URL on S3.
			if version, ok := request.PathParameters["param2"]; ok {
				version = strings.TrimPrefix(version, "v")
				if version == versionDevelopment || version == versionStable {
					return events.APIGatewayProxyResponse{
						StatusCode: http.StatusNotFound,
						Body:       "development or stable channels are not available when packaged as deb",
					}, nil
				}
				return newRedirectResponse(
					fmt.Sprintf("https://offen.s3.eu-central-1.amazonaws.com/deb/offen_%s_amd64.deb", version),
				), nil
			}
			return newRedirectResponse(
				"https://offen.s3.eu-central-1.amazonaws.com/deb/offen_latest_amd64.deb",
			), nil
		default:
			// The default behavior is to return the tarball containing binaries
			return newRedirectResponse(
				fmt.Sprintf("https://offen.s3.eu-central-1.amazonaws.com/binaries/offen-%s.tar.gz", param1),
			), nil
		}

	}

	latest, err := getLatestReleaseInfo(os.Getenv("GITHUB_REPO"))
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("error getting latest release: %w", err)
	}

	asset, assetErr := latest.match(githubAssetTypeTarball)
	if assetErr != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
		}, nil
	}
	return newRedirectResponse(asset), nil
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

func newRedirectResponse(location string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location": location,
		},
	}
}
