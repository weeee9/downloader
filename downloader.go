package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Downloader struct {
	concurrency int
}

func NewDownloader(concurrency int) *Downloader {
	return &Downloader{concurrency: concurrency}
}

func (d *Downloader) Download(url, filename string) error {
	if filename == "" {
		filename = filepath.Base(url)
	}

	resp, err := http.Head(url)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		// concurrency download
		return d.multiDownload(url, filename, int(resp.ContentLength))
	}

	// normal download
	return d.singleDownload(url, filename)
}

func (d *Downloader) multiDownload(url, filename string, contentLen int) error {

	partSize := contentLen / d.concurrency

	partDir := getDir(filename)
	os.Mkdir(partDir, 0777)
	defer os.RemoveAll(partDir)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)

	rangeStart := 0

	for i := 0; i < d.concurrency; i++ {
		go func(i, rangeStart int) {
			defer wg.Done()

			rangeEnd := rangeStart + partSize
			if i == d.concurrency-1 {
				rangeEnd = contentLen
			}

			d.partialDownload(url, filename, rangeStart, rangeEnd, i)

		}(i, rangeStart)

		rangeStart += partSize + 1
	}

	wg.Wait()

	if err := d.merge(filename); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) partialDownload(url, filename string, start, end, i int) {
	if start >= end {
		return
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY

	file, err := os.OpenFile(getPartialFilename(filename, i), flags, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		if err == io.EOF {
			return
		}
		log.Fatal(err)
	}

}

func (d *Downloader) merge(filename string) error {
	dst, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer dst.Close()

	for i := 0; i < d.concurrency; i++ {
		partFilename := getPartialFilename(filename, i)
		file, err := os.Open(partFilename)
		if err != nil {
			return err
		}
		io.Copy(dst, file)
		file.Close()
		os.Remove(partFilename)
	}

	return nil
}

func (d *Downloader) singleDownload(url, filename string) error {
	return nil
}
