package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasttemplate"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const MetaTags = `
<meta name="og:description" content="%s">
<meta name="og:image" content="%s">
<meta name="og:image:type" content="%s">
<meta name="theme-color" content="#0288D1">
<link type="application/json+oembed" href="%s">
`

type Emote struct {
	URL        string `json:"url"`
	FrameCount int    `json:"frame_count"`
	Format     string `json:"format"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type GQLEmoteResponse struct {
	Data struct {
		Emote struct {
			Name      string    `json:"name"`
			Animated  bool      `json:"animated"`
			CreatedAt time.Time `json:"created_at"`
			Images    []Emote   `json:"images"`
			Owner     struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"owner"`
			Channels struct {
				Total int `json:"total"`
			} `json:"channels"`
		} `json:"emote"`
	} `json:"data"`
}

type OEmbedData struct {
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Type         string `json:"type"`
	URL          string `json:"url"`
}

func main() {
	root := os.Getenv("FS_ROOT")
	if root == "" {
		root = "./public"
	}

	gqlApiURL := os.Getenv("GQL_API_URL")
	websiteURL := os.Getenv("WEBSITE_URL")

	// Setup FS handler
	fs := &fasthttp.FS{
		Root: root,
	}

	index, err := os.ReadFile(path.Join(root, "index.html"))
	if err != nil {
		panic(err)
	}

	favicon, err := os.ReadFile(path.Join(root, "favicon.ico"))
	if err != nil {
		panic(err)
	}

	//replace-me
	template := fasttemplate.New(string(index), "<!-- {{", "}} -->")

	addr := os.Getenv("WEBSITE_BIND")
	if addr == "" {
		addr = "0.0.0.0:3000"
	}

	// Start HTTP server.
	log.Printf("Starting HTTP server on %q\n", addr)
	go func() {
		handler := fs.NewRequestHandler()
		if err := fasthttp.ListenAndServe(addr, func(ctx *fasthttp.RequestCtx) {
			pth := string(ctx.Path())
			if strings.HasPrefix(pth, "/assets/") {
				handler(ctx)
			} else {
				if strings.HasPrefix(pth, "/emotes/") {
					id := strings.Split(strings.TrimPrefix(pth, "/emotes/"), "?")[0]
					emoteID, err := primitive.ObjectIDFromHex(id)
					if err != nil {
						goto end
					}
					// handle emote route
					query := url.Values{}
					query.Set("query", fmt.Sprintf(`
{
    emote(id: "%s") {
        name
		animated
		created_at
		images {
			url
			frame_count
			width
			height
		}
        owner {
			id
            display_name
        }
        channels {
            total
        }
    }
}`, emoteID.Hex()))

					req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?%s", gqlApiURL, query.Encode()), nil)
					if err != nil {
						log.Println("Failed to make request: ", err)
						goto end
					}
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Println("Failed to do request: ", err)
						goto end
					}
					defer resp.Body.Close()
					data, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Println("Failed to read response: ", err)
						goto end
					}
					gqlResp := GQLEmoteResponse{}
					if err := json.Unmarshal(data, &gqlResp); err != nil {
						log.Println("Failed to parse response: ", err)
						goto end
					}

					emote := gqlResp.Data.Emote
					if len(emote.Images) == 0 {
						goto end
					}

					imageType := ""
					var images []Emote

					if emote.Animated {
						imageType = "image/gif"
						for _, v := range emote.Images {
							if v.FrameCount != 1 && v.Format == "GIF" {
								images = append(images, v)
							}
						}
					} else {
						imageType = "image/webp"
						for _, v := range emote.Images {
							if v.Format == "WEBP" {
								images = append(images, v)
							}
						}
					}

					sort.Slice(images, func(i, j int) bool {
						return images[i].Width > images[j].Width
					})

					if len(images) == 0 {
						goto end
					}

					oembed, _ := json.Marshal(OEmbedData{
						AuthorName:   fmt.Sprintf("%s by %s (%d Channels)", emote.Name, emote.Owner.DisplayName, emote.Channels.Total),
						AuthorURL:    fmt.Sprintf("%s/emotes/%s", websiteURL, emoteID.Hex()),
						ProviderName: "7TV.APP - It's like a third party thing",
						ProviderURL:  websiteURL,
						Type:         "image",
						URL:          images[0].URL,
					})

					obj := base64.StdEncoding.EncodeToString(oembed)

					ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
					ctx.SetBodyString(template.ExecuteString(map[string]interface{}{
						"META": fmt.Sprintf(MetaTags,
							fmt.Sprintf("uploaded by %s", emote.Owner.DisplayName), // og:description
							images[0].URL, // og:image
							imageType,     // og:image:type
							fmt.Sprintf("%s/services/oembed/%s.json", websiteURL, obj), // oembed url
						),
					}))
				} else if strings.HasPrefix(pth, "/services/oembed/") && strings.HasSuffix(pth, ".json") {
					out, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(pth, "/services/oembed/"), ".json"))
					if err != nil {
						ctx.SetStatusCode(404)
						return
					}

					data := OEmbedData{}
					if err = json.Unmarshal(out, &data); err == nil {
						out, _ = json.Marshal(data)
						ctx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
						ctx.Response.Header.Set("Cache-Control", "max-age=3600")
						ctx.SetBody(out)
						return
					}

					ctx.SetStatusCode(404)
					return
				} else if pth == "/favicon.ico" {
					ctx.Response.Header.Set("Content-Type", "image/ico")
					ctx.SetBody(favicon)
					return
				}

			end:
				ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
				ctx.SetBodyString(template.ExecuteString(map[string]interface{}{
					"META": "",
				}))
			}
		}); err != nil {
			log.Fatalf("error in ListenAndServe: %s", err)
		}
	}()

	log.Printf("Serving files from directory %s\n", root)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Wait forever.
	<-sig
}
