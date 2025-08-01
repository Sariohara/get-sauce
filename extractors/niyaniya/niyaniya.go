package niyaniya

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/gan-of-culture/get-sauce/request"
	"github.com/gan-of-culture/get-sauce/static"
	"github.com/gan-of-culture/get-sauce/utils"
)

const origin = "https://niyaniya.moe"
const site = origin + "/"
const api = "https://api.schale.network/books"
const detailAPI = api + "/detail"

type apiSearchResponse struct {
	Entries []struct {
		ID        int    `json:"id"`
		Key       string `json:"key"`
		Title     string `json:"title"`
		Language  int    `json:"language"`
		Pages     int    `json:"pages"`
		Thumbnail struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"thumbnail"`
	} `json:"entries"`
	Limit   int `json:"limit"`
	Page    int `json:"page"`
	Total   int `json:"total"`
	Matches []struct {
		Query     string `json:"query"`
		Namespace string `json:"namespace"`
		Entries   []struct {
			Value      string `json:"value"`
			StartsWith bool   `json:"starts_with"`
			EndsWith   bool   `json:"ends_with"`
		} `json:"entries"`
	} `json:"matches"`
}

type apiEntryMetaData struct {
	ID         int    `json:"id"`
	Key        string `json:"key"`
	CreatedAt  int64  `json:"created_at"`
	Title      string `json:"title"`
	Thumbnails struct {
		Base string `json:"base"`
		Main struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"main"`
		Entries []struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"entries"`
	} `json:"thumbnails"`
	Tags []struct {
		Name      string `json:"name"`
		Count     int    `json:"count"`
		Namespace int    `json:"namespace,omitempty"`
	} `json:"tags"`
}

type apiEntryData struct {
	Data map[string]struct {
		ID   int    `json:"id"`
		Key  string `json:"key"`
		Size int    `json:"size"`
	} `json:"data"`
	Similar []struct {
		ID        int    `json:"id"`
		Key       string `json:"key"`
		Title     string `json:"title"`
		Language  int    `json:"language"`
		Pages     int    `json:"pages"`
		Thumbnail struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"thumbnail"`
	} `json:"similar"`
}

type apiEntryResponse struct {
	ID        int    `json:"id"`
	PublicKey string `json:"public_key"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	Data      map[string]struct {
		ID        int    `json:"id"`
		PublicKey string `json:"public_key"`
		Size      int    `json:"size"`
	} `json:"data"`
	Thumbnails struct {
		Base string `json:"base"`
		Main struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"main"`
		Entries []struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"entries"`
	} `json:"thumbnails"`
	Tags []struct {
		Namespace int    `json:"namespace"`
		Name      string `json:"name"`
		Count     int    `json:"count"`
	} `json:"tags"`
	Rels []struct {
		ID        int    `json:"id"`
		PublicKey string `json:"public_key"`
		CreatedAt int64  `json:"created_at"`
		Title     string `json:"title"`
		Language  string `json:"language"`
		Pages     int    `json:"pages"`
		Thumbnail struct {
			Path       string `json:"path"`
			Dimensions []int  `json:"dimensions"`
		} `json:"thumbnail"`
		Tags []struct {
			Namespace int    `json:"namespace,omitempty"`
			Name      string `json:"name"`
		} `json:"tags"`
		Subtitle string `json:"subtitle,omitempty"`
	} `json:"rels"`
}

type apiEntryDataResponse struct {
	Base    string `json:"base"`
	Entries []struct {
		Path       string `json:"path"`
		Dimensions []int  `json:"dimensions"`
	} `json:"entries"`
}

type extractor struct{}

// New returns a niyaniya extractor
func New() static.Extractor {
	return &extractor{}
}

func (e *extractor) Extract(URL string) ([]*static.Data, error) {
	postIDs := parseURL(URL)
	if len(postIDs) == 0 {
		return nil, static.ErrURLParseFailed
	}

	data := []*static.Data{}
	for _, pID := range postIDs {
		d, err := extractData(pID)
		if err != nil {
			return nil, utils.Wrap(err, pID)
		}
		data = append(data, d...)
	}

	return data, nil
}

func parseURL(URL string) []string {
	if strings.HasPrefix(URL, site+"g/") {
		splitUrl := strings.Split(URL, "/")
		splitUrlLen := len(splitUrl)
		return []string{fmt.Sprintf("%s/%s/%s", detailAPI, splitUrl[splitUrlLen-2], splitUrl[splitUrlLen-1])}
	}

	u, err := url.Parse(URL)
	if err != nil {
		return nil
	}

	apiUrl, err := url.Parse(api)
	if err != nil {
		return nil
	}
	q := apiUrl.Query()
	ss := strings.Split(path.Base(u.Path), ":")
	q.Set("s", fmt.Sprintf("%s:^%s$", ss[0], ss[1]))
	apiUrl.RawQuery = q.Encode()

	res, err := request.GetAsBytesWithHeaders(apiUrl.String(), map[string]string{
		"Origin":  origin,
		"Referer": site,
	})
	if err != nil {
		return nil
	}

	apiResponse := apiSearchResponse{}
	err = json.Unmarshal(res, &apiResponse)
	if err != nil {
		return nil
	}

	out := []string{}
	for _, entry := range apiResponse.Entries {
		out = append(out, fmt.Sprintf("%s/%d/%s", detailAPI, entry.ID, entry.Key))
	}

	return out
}

func extractData(URL string) ([]*static.Data, error) {

	res, err := request.GetAsBytesWithHeaders(URL, map[string]string{
		"Origin":  origin,
		"Referer": site,
	})
	if err != nil {
		return nil, err
	}

	entryMetadata := &apiEntryMetaData{}
	err = json.Unmarshal(res, entryMetadata)
	if err != nil {
		return nil, err
	}

	if entryMetadata == nil {
		return nil, static.ErrDataSourceParseFailed
	}

	// this request needs a crt which is aquired through a cf challenge
	res, err = request.PostAsBytesWithHeaders(URL, map[string]string{
		"Origin":  origin,
		"Referer": site,
	}, nil)

	if err != nil {
		return nil, err
	}

	entryData := &apiEntryData{}
	err = json.Unmarshal(res, entryData)
	if err != nil {
		return nil, err
	}

	streams := make(map[string]*static.Stream)
	var stream *static.Stream
	streamIdx := -1
	for k, v := range entryData.Data {
		streamIdx++
		stream = &static.Stream{
			Type: static.DataTypeImage,
			Size: int64(v.Size),
			Headers: map[string]string{
				"Referer": site,
				"Origin":  origin,
			},
		}

		dataURL := strings.Replace(URL, "detail", "data", 1)
		dataURL = fmt.Sprintf("%s/%d/%s", dataURL, v.ID, v.Key)
		dURL, err := url.Parse(dataURL)
		if err != nil {
			return nil, err
		}
		q := url.Values{}
		q.Set("v", fmt.Sprint(entryMetadata.CreatedAt))
		q.Set("w", k)
		dURL.RawQuery = q.Encode()
		res, err = request.GetAsBytesWithHeaders(dURL.String(), map[string]string{
			"Origin":  origin,
			"Referer": site,
		})
		if err != nil {
			return nil, err
		}

		dataRes := apiEntryDataResponse{}
		err = json.Unmarshal(res, &dataRes)
		if err != nil {
			return nil, err
		}

		numOfEntries := len(dataRes.Entries)
		stream.Info = fmt.Sprintf("%d pages", numOfEntries)
		middleEntry := dataRes.Entries[numOfEntries/2]
		if len(middleEntry.Dimensions) > 1 {
			stream.Quality = fmt.Sprintf("%dx%d", middleEntry.Dimensions[0], middleEntry.Dimensions[1])
		}

		cdnUrl, err := url.Parse(dataRes.Base)
		if err != nil {
			return nil, err
		}

		urls := make([]*static.URL, numOfEntries)
		for idx, entry := range dataRes.Entries {
			strDataURL, err := url.JoinPath(dataRes.Base, entry.Path)
			if err != nil {
				return nil, err
			}
			dataURL, err := cdnUrl.Parse(strDataURL)
			if err != nil {
				return nil, err
			}
			q := url.Values{}
			q.Set("w", k)
			dataURL.RawQuery = q.Encode()
			strDataURL = dataURL.String()
			urls[idx] = &static.URL{
				URL: strDataURL,
				Ext: utils.GetFileExt(entry.Path),
			}
		}
		stream.URLs = urls
		streams[fmt.Sprint(streamIdx)] = stream
	}

	return []*static.Data{
		{
			Site:    site,
			Title:   entryMetadata.Title,
			Type:    static.DataTypeImage,
			Streams: streams,
			URL:     URL,
		},
	}, nil
}
