# get

This is a simple AWS Lambda handler for serving <get.offen.dev>. It looks up the releases for the `offen/offen` repo, picks the latest release (including pre-releases) and returns a redirect to the download location.

---

Changes are deployed manually (this requires Go 1.13+ and `serverless` to be installed):

```sh
make build
sls deploy
```

