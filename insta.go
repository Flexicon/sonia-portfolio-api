package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gocolly/colly"
)

type pageInfo struct {
	EndCursor string `json:"end_cursor"`
	NextPage  bool   `json:"has_next_page"`
}

type postNode struct {
	ShortCode    string `json:"shortcode"`
	ImageURL     string `json:"display_url"`
	ThumbnailURL string `json:"thumbnail_src"`
	IsVideo      bool   `json:"is_video"`
	Date         int    `json:"date"`
	Dimensions   struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	CaptionEdges struct {
		Edges []struct {
			Node struct {
				Caption string `json:"text"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"edge_media_to_caption"`
	LikesEdge struct {
		Count int `json:"count"`
	} `json:"edge_media_preview_like"`
	CommentsEdge struct {
		Count int `json:"count"`
	} `json:"edge_media_to_comment"`
}

type postEdge struct {
	Node postNode `json:"node"`
}

type mainPageData struct {
	Rhxgis    string `json:"rhx_gis"`
	EntryData struct {
		ProfilePage []struct {
			Graphql struct {
				User struct {
					ID    string `json:"id"`
					Media struct {
						Edges    []postEdge `json:"edges"`
						PageInfo pageInfo   `json:"page_info"`
					} `json:"edge_owner_to_timeline_media"`
				} `json:"user"`
			} `json:"graphql"`
		} `json:"ProfilePage"`
	} `json:"entry_data"`
}

type nextPageData struct {
	Data struct {
		User struct {
			Container struct {
				Count    int        `json:"count"`
				PageInfo pageInfo   `json:"page_info"`
				Edges    []postEdge `json:"edges"`
			} `json:"edge_user_to_photos_of_you"`
		} `json:"user"`
	} `json:"data"`
}

type postResponse struct {
	Caption      string `json:"caption"`
	ImageURL     string `json:"image_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Likes        int    `json:"likes"`
	Comments     int    `json:"comments"`
	Link         string `json:"link"`
}

type instaResponse struct {
	Posts []postResponse `json:"posts"`
	Count int            `json:"count"`
}

// appendPost structures the postEdge data into a postResponse
// then appends it to the given postResponse slice and returns it
func appendPost(posts []postResponse, edge postEdge) []postResponse {
	if edge.Node.IsVideo {
		return posts
	}

	var caption string
	if len(edge.Node.CaptionEdges.Edges) > 0 {
		caption = edge.Node.CaptionEdges.Edges[0].Node.Caption
	}

	link := fmt.Sprintf("https://instagram.com/p/%s", edge.Node.ShortCode)

	post := postResponse{
		Caption:      caption,
		ImageURL:     edge.Node.ImageURL,
		ThumbnailURL: edge.Node.ThumbnailURL,
		Likes:        edge.Node.LikesEdge.Count,
		Comments:     edge.Node.CommentsEdge.Count,
		Link:         link,
	}

	return append(posts, post)
}

// InstaHandler fetches instagram posts and returns them in JSON format
func InstaHandler(w http.ResponseWriter, r *http.Request) {
	const nextPageURL string = `https://www.instagram.com/graphql/query/?query_hash=%s&variables=%s`
	const nextPagePayload string = `{"id":"%s","first":50,"after":"%s"}`
	const instagramAccount string = "sonia_ehm"

	var requestID string
	var queryIDPattern = regexp.MustCompile(`queryId:".{32}"`)
	var actualUserID string
	var posts []postResponse

	c := colly.NewCollector(
		colly.CacheDir("./_instagram_cache/"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"),
	)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("X-Requested-With", "XMLHttpRequest")
		r.Headers.Set("Referrer", "https://www.instagram.com/"+instagramAccount)
		if r.Ctx.Get("gis") != "" {
			gis := fmt.Sprintf("%s:%s", r.Ctx.Get("gis"), r.Ctx.Get("variables"))
			h := md5.New()
			h.Write([]byte(gis))
			gisHash := fmt.Sprintf("%x", h.Sum(nil))
			r.Headers.Set("X-Instagram-GIS", gisHash)
		}
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		d := c.Clone()
		d.OnResponse(func(r *colly.Response) {
			requestIds := queryIDPattern.FindAll(r.Body, -1)
			requestID = string(requestIds[1][9:41])
		})
		requestIDURL := e.Request.AbsoluteURL(e.ChildAttr(`link[as="script"]`, "href"))
		d.Visit(requestIDURL)

		dat := e.ChildText("body > script:first-of-type")
		jsonData := dat[strings.Index(dat, "{") : len(dat)-1]
		data := &mainPageData{}
		err := json.Unmarshal([]byte(jsonData), data)
		if err != nil {
			log.Fatal(err)
		}

		page := data.EntryData.ProfilePage[0]
		actualUserID = page.Graphql.User.ID

		for _, obj := range page.Graphql.User.Media.Edges {
			posts = appendPost(posts, obj)
		}

		nextPageVars := fmt.Sprintf(nextPagePayload, actualUserID, page.Graphql.User.Media.PageInfo.EndCursor)
		e.Request.Ctx.Put("variables", nextPageVars)

		if page.Graphql.User.Media.PageInfo.NextPage {
			u := fmt.Sprintf(
				nextPageURL,
				requestID,
				url.QueryEscape(nextPageVars),
			)
			log.Println("Next page found", u)
			e.Request.Ctx.Put("gis", data.Rhxgis)
			e.Request.Visit(u)
		}
	})

	c.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
	})

	c.OnResponse(func(r *colly.Response) {
		if strings.Index(r.Headers.Get("Content-Type"), "json") == -1 {
			return
		}

		data := &nextPageData{}
		err := json.Unmarshal(r.Body, data)
		if err != nil {
			log.Fatal(err)
		}

		for _, obj := range data.Data.User.Container.Edges {
			posts = appendPost(posts, obj)
		}
		if data.Data.User.Container.PageInfo.NextPage {
			nextPageVars := fmt.Sprintf(nextPagePayload, actualUserID, data.Data.User.Container.PageInfo.EndCursor)
			r.Request.Ctx.Put("variables", nextPageVars)
			u := fmt.Sprintf(
				nextPageURL,
				requestID,
				url.QueryEscape(nextPageVars),
			)
			log.Println("Next page found", u)
			r.Request.Visit(u)
		}
	})

	c.Visit("https://instagram.com/" + instagramAccount)

	json.NewEncoder(w).Encode(instaResponse{Posts: posts, Count: len(posts)})
}
