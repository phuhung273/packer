// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package publiczip

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	plugingetter "github.com/hashicorp/packer/packer/plugin-getter"
	"github.com/hashicorp/packer/packer/plugin-getter/github"
)

const (
	defaultUserAgent = "packer-public-zip-plugin-getter"
)

type Getter struct {
	Client    *http.Client
	UserAgent string
}

var _ plugingetter.Getter = &Getter{}

func (g *Getter) Get(what string, opts plugingetter.GetOptions) (io.ReadCloser, error) {

	if g.Client == nil {
		g.Client = &http.Client{}
	}

	var req *http.Request
	var err error
	transform := func(in io.ReadCloser) (io.ReadCloser, error) {
		return in, nil
	}

	userAgent := defaultUserAgent
	if g.UserAgent != "" {
		userAgent = g.UserAgent
	}

	switch what {
	case "releases":
		u := filepath.ToSlash("https://api.github.com/repos/" + opts.PluginRequirement.Identifier.RealRelativePath() + "/git/matching-refs/tags")
		req, err = http.NewRequest("GET", u, nil)
		transform = github.TransformVersionStream
	case "sha256":
		// something like https://github.com/sylviamoss/packer-plugin-comment/releases/download/v0.2.11/packer-plugin-comment_v0.2.11_x5_SHA256SUMS
		u := filepath.ToSlash("https://github.com/" + opts.PluginRequirement.Identifier.RealRelativePath() + "/releases/download/" + opts.Version() + "/" + opts.PluginRequirement.FilenamePrefix() + opts.Version() + "_SHA256SUMS")
		req, err = http.NewRequest("GET", u, nil)
		transform = github.TransformChecksumStream()
	case "zip":
		u := filepath.ToSlash("https://" + opts.PluginRequirement.Identifier.Hostname + "/" + opts.PluginRequirement.Identifier.RealRelativePath() + "/releases/download/" + opts.Version() + "/" + opts.ExpectedZipFilename())
		req, err = http.NewRequest("GET", u, nil)
	default:
		return nil, fmt.Errorf("%q not implemented", what)
	}

	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] public-zip-getter: getting %q", req.URL)

	req.Header = http.Header{
		"User-Agent": {userAgent},
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		log.Printf("[TRACE] failed requesting: %T. %v", err, err)
		return nil, err
	}

	return transform(resp.Body)
}
