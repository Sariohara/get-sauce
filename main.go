package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/gan-of-culture/get-sauce/config"
	"github.com/gan-of-culture/get-sauce/downloader"
	"github.com/gan-of-culture/get-sauce/extractors"
	"github.com/gan-of-culture/get-sauce/static"
)

func init() {
	flag.IntVar(&config.Amount, "a", 0, "Amount of files to download")
	flag.IntVar(&config.Caption, "c", -1, "Download caption to a extra file")
	flag.BoolVar(&config.NoMerge, "n", false, "No automatic merging of video, audio and subtitles using ffmpeg")
	flag.StringVar(&config.OutputName, "o", "", "Output name")
	flag.StringVar(&config.OutputPath, "O", "", "Output path (include ending delimiter)")
	flag.StringVar(&config.Pages, "p", "", "Enter pages like 1,2,3-4,6,7,8-9 for doujins")
	flag.BoolVar(&config.Quiet, "q", false, "Quiet mode - show minimal information")
	flag.StringVar(&config.SelectStream, "s", "0", "Select a stream")
	flag.BoolVar(&config.ShowExtractedData, "j", false, "Show extracted data as json")
	flag.BoolVar(&config.ShowInfo, "i", false, "Show info")
	flag.IntVar(&config.Workers, "w", 1, "Number of workers used for downloading")
	flag.StringVar(&config.Username, "un", "", "Username for exhentai")
	flag.StringVar(&config.UserPassword, "up", "", "User password for exhentai")
}

func download(URL string) {
	if !config.NoMerge {
		_, err := exec.LookPath("ffmpeg")
		if err != nil {
			log.Println("No merging possible, because ffmpeg is not installed or not found in PATH")
			config.NoMerge = true
		}
	}

	data, err := extractors.Extract(URL)
	if err != nil {
		log.Fatal(err)
	}

	if config.ShowExtractedData {
		for _, singleData := range data {
			jsonData, _ := json.MarshalIndent(singleData, "", "    ")
			fmt.Printf("%s\n", jsonData)
		}
		return
	}

	if config.SelectStream == "" {
		config.SelectStream = "0"
	}

	lenOfData := len(data)
	/*
		We have 3 main types of data that has to be downloaded concurrently
		1. lenOfData = 3000 e.g. mass scraping image boards
		2. lenOfData = 1 URLs = 200 e.g. doujin
		3. lenOfData = 1-10 but big file size e.g.hentai video
		here in main we will deal with the first type
	*/
	workers := config.Workers
	if workers > lenOfData {
		workers = lenOfData
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	datachan := make(chan *static.Data, lenOfData)

	downloader := downloader.New(true)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				d, ok := <-datachan
				if !ok {
					return
				}
				err := downloader.Download(d)
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}

	for _, d := range data {
		datachan <- d
	}
	close(datachan)
	wg.Wait()
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Too few arguments")
		fmt.Println("Usage: go-hentai-scraper [args] URLs...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	for _, a := range args {
		download(a)
	}
}
