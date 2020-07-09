package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if channel, ok := request.PathParameters["channel"]; ok {
		return newRedirectResponse(
			fmt.Sprintf("https://offen.s3.eu-central-1.amazonaws.com/binaries/offen-%s.tar.gz", channel),
		), nil
	}

	latest, err := getLatestReleaseInfo(os.Getenv("GITHUB_REPO"))
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("error getting latest release: %w", err)
	}

	return newRedirectResponse(latest.Assets[0].BrowserDownloadURL), nil
}

type releaseInfo struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
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
