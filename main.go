package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type releaseInfo struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var redirect string
	if channel, ok := request.PathParameters["channel"]; ok {
		switch channel {
		case "latest", "stable":
			redirect = fmt.Sprintf("https://offen.s3.eu-central-1.amazonaws.com/binaries/offen-%s.tar.gz", channel)
		default:
			return events.APIGatewayProxyResponse{}, fmt.Errorf("channel %s is not supported", channel)
		}
	} else {
		repo := os.Getenv("GITHUB_REPO")
		endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

		res, err := http.Get(endpoint)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		defer res.Body.Close()

		var data []releaseInfo
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		redirect = data[0].Assets[0].BrowserDownloadURL
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location": redirect,
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
