package hentaifox

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gan-of-culture/go-hentai-scraper/request"
	"github.com/gan-of-culture/go-hentai-scraper/static"
	"github.com/gan-of-culture/go-hentai-scraper/utils"
)

const site = "https://hentaifox.com/"
const cdn = "https://i.hentaifox.com/"

var reTitle *regexp.Regexp = regexp.MustCompile(`<title>(.+)</title>`)
var reJSONStr *regexp.Regexp = regexp.MustCompile(`parseJSON\('[^']+`)
var reImgDir *regexp.Regexp = regexp.MustCompile(`image_dir" value="([^"]*)`)
var reGalleryID *regexp.Regexp = regexp.MustCompile(`gallery_id" value="([^"]*)`)

type extractor struct{}

// New returns a hentaifox extractor.
func New() static.Extractor {
	return &extractor{}
}

func (e *extractor) Extract(URL string) ([]*static.Data, error) {
	IDs := parseURL(URL)
	if len(IDs) == 0 {
		return nil, static.ErrURLParseFailed
	}

	data := []*static.Data{}
	for _, u := range IDs {
		d, err := extractData(u)
		if err != nil {
			return nil, utils.Wrap(err, u)
		}
		data = append(data, d)
	}

	return data, nil
}

func parseURL(URL string) []string {
	re := regexp.MustCompile(`/gallery/(\d+)/`)
	matchedID := re.FindStringSubmatch(URL)
	if len(matchedID) == 2 {
		return []string{matchedID[1]}
	}

	htmlString, err := request.Get(URL)
	if err != nil {
		return []string{}
	}

	IDs := []string{}
	for _, v := range re.FindAllStringSubmatch(htmlString, -1) {
		IDs = append(IDs, v[1])
	}
	return utils.RemoveAdjDuplicates(IDs)
}

func extractData(ID string) (*static.Data, error) {
	htmlString, err := request.Get(fmt.Sprintf("%sg/%s/1/", site, ID))
	if err != nil {
		return &static.Data{}, err
	}

	title := strings.TrimSuffix(reTitle.FindStringSubmatch(htmlString)[1], " - Page 1 - HentaiFox")

	jsonStr := reJSONStr.FindString(htmlString)
	if jsonStr == "" {
		return &static.Data{}, errors.New("JSON string not found for")
	}
	//cut of the beginning
	jsonStr = jsonStr[11:]

	imageData := map[string]string{}
	err = json.Unmarshal([]byte(jsonStr), &imageData)
	if err != nil {
		return &static.Data{}, err
	}

	imageDir := reImgDir.FindStringSubmatch(htmlString)
	if len(imageDir) < 1 {
		return &static.Data{}, errors.New("cannot find image_dir for")
	}

	gID := reGalleryID.FindStringSubmatch(htmlString)
	if len(gID) < 1 {
		return &static.Data{}, errors.New("cannot find gallery_id for")
	}

	noOfPages := len(imageData)
	pages := utils.NeedDownloadList(noOfPages)

	URLs := []*static.URL{}
	for _, i := range pages {
		params := strings.Split(imageData[fmt.Sprint(i)], ",") //type, width, height
		switch params[0] {
		case "j":
			params[0] = "jpg"
		case "p":
			params[0] = "png"
		case "b":
			params[0] = "bmp"
		case "g":
			params[0] = "gif"
		}
		URLs = append(URLs, &static.URL{
			URL: fmt.Sprintf("%s%s/%s/%d.%s", cdn, imageDir[1], gID[1], i, params[0]),
			Ext: params[0],
		})
	}

	return &static.Data{
		Site:  site,
		Title: title,
		Type:  "image",
		Streams: map[string]*static.Stream{
			"0": {
				URLs: URLs,
			},
		},
		Url: fmt.Sprintf("%sgallery/%s/", site, ID),
	}, nil

}