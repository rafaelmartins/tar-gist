package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Gist struct {
	ID          *string             `json:"id,omitempty"`
	Description *string             `json:"description,omitempty"`
	Public      *bool               `json:"public,omitempty"`
	Files       map[string]GistFile `json:"files,omitempty"`
	URL         *string             `json:"html_url,omitempty"`
}

type GistFile struct {
	Size      *int    `json:"size,omitempty"`
	Type      *string `json:"type,omitempty"`
	RawURL    *string `json:"raw_url,omitempty"`
	Content   *string `json:"content,omitempty"`
	Truncated *bool   `json:"truncated,omitempty"`
}

func GistCreate(content *string) (*Gist, error) {
	desc := "tar-gist file"
	public := false
	files := make(map[string]GistFile)
	files["tar-gist.pem"] = GistFile{Content: content}

	gist := Gist{Description: &desc, Public: &public, Files: files}

	j, err := json.Marshal(gist)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(j)
	resp, err := http.Post("https://api.github.com/gists", "application/json", buf)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &gist); err != nil {
		return nil, err
	}

	return &gist, nil
}

func GistGet(id *string) (*string, error) {
	resp, err := http.Get("https://api.github.com/gists/" + *id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gist Gist

	if err := json.Unmarshal(body, &gist); err != nil {
		return nil, err
	}

	if gist.Files == nil {
		return nil, fmt.Errorf("No files found")
	}

	var rv string

	for file := range gist.Files {
		if file != "tar-gist.pem" {
			continue
		}

		if !*gist.Files[file].Truncated {
			rv = *gist.Files[file].Content
			break
		}

		resp, err := http.Get(*gist.Files[file].RawURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		rv = string(body)
		break
	}

	return &rv, nil
}
