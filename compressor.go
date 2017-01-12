package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type walker struct {
	writer *tar.Writer
}

func (w *walker) walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	link, _ := os.Readlink(path)
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}

	hdr.Name = path
	if hdr.Name[0] == '/' { // How to support windows?
		hdr.Name = hdr.Name[1:]
	}

	if err := w.writer.WriteHeader(hdr); err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return nil
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if _, err := w.writer.Write(content); err != nil {
		return err
	}

	if err := w.writer.Flush(); err != nil {
		return err
	}
	return nil
}

func CreateTar(entries []string) (*[]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	w := walker{writer: tw}

	for _, entry := range entries {
		info, err := os.Lstat(entry)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			err = filepath.Walk(entry, w.walkFunc)
			if err != nil {
				return nil, err
			}
		} else {
			err = w.walkFunc(entry, info, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	rv := buf.Bytes()

	return &rv, nil
}

func ListTar(reader *tar.Reader) error {
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		if hdr.Linkname != "" {
			fmt.Printf("%s -> %s\n", hdr.Name, hdr.Linkname)
			continue
		}

		fmt.Println(hdr.Name)
	}
	return nil
}

func ExtractTar(reader *tar.Reader) error {
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(hdr.Name, 0777); err != nil {
				return err
			}
			continue
		}

		if hdr.Linkname != "" {
			if _, err := os.Stat(hdr.Name); err != nil {
				continue
			}
			if err := os.Symlink(hdr.Linkname, hdr.Name); err != nil {
				return err
			}
			continue
		}

		f, err := os.Create(hdr.Name)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(f, reader); err != nil {
			return err
		}
	}
	return nil
}

func CompressGzip(input *[]byte) (*[]byte, error) {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)

	if _, err := gw.Write(*input); err != nil {
		return nil, err
	}

	if err := gw.Close(); err != nil {
		return nil, err
	}

	rv := buf.Bytes()

	return &rv, nil
}

func UncompressGzip(input *[]byte) (*[]byte, error) {
	buf := bytes.NewReader(*input)
	gr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	rv, err := ioutil.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	return &rv, nil
}

func Compress(entries []string) (*string, error) {
	t, err := CreateTar(entries)
	if err != nil {
		return nil, err
	}

	x, err := CompressGzip(t)
	if err != nil {
		return nil, err
	}

	b := pem.Block{Type: "TAR-GIST", Bytes: *x}
	rv := string(pem.EncodeToMemory(&b))

	return &rv, nil
}

func Uncompress(content *string) (*tar.Reader, error) {
	b, _ := pem.Decode([]byte(*content))
	if b == nil {
		return nil, fmt.Errorf("error: pem: failed to extract gzipped content from PEM")
	}

	if b.Type != "TAR-GIST" {
		return nil, fmt.Errorf("error: pem: invalid PEM type: %s", b.Type)
	}

	t, err := UncompressGzip(&b.Bytes)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(*t)

	return tar.NewReader(buf), nil
}
