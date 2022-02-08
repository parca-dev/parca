package debuginfo

import (
	"io"
	"net/http"
	"path"
)

var publicServer = "https://debuginfod.systemtap.org"

type DebugInfodClient interface {
	GetDebugInfo(buildid string) (io.ReadCloser, error)
}

type HttpDebuginfodClient struct {
	UpstreamServer string //url
}
type ObjectStorageDebugInfodClientCache struct {
	client DebugInfodClient
}

func NewHttpDebugInfoClient(serverUrl string) HttpDebuginfodClient {
	return HttpDebuginfodClient{UpstreamServer: serverUrl}
}

func NewObjectStorageDebugInfodClientCache(h HttpDebuginfodClient) *ObjectStorageDebugInfodClientCache {
	return &ObjectStorageDebugInfodClientCache{client: &h}
}

func (c *ObjectStorageDebugInfodClientCache) GetDebugInfo(buildid string) (io.ReadCloser, error) {
	return c.client.GetDebugInfo(buildid)
}

func (c *HttpDebuginfodClient) GetDebugInfo(buildID string) (io.ReadCloser, error) {
	serverUrl := path.Join(c.UpstreamServer, buildID, "debuginfo")

	resp, err := http.Get(serverUrl)
	if err != nil {
		//level.Debug(logger).Log("msg", "object not found in public server", "object", buildID, "err", err)
		return nil, ErrDebugInfoNotFound
	}

	return resp.Body, nil
}
