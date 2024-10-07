package manganelapiclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"sort"
	"time"

	"github.com/ivan-penchev/manga-updates/pkg/types"
	"github.com/machinebox/graphql"
)

type MangaNelAPIClient struct {
	addr   string
	apiKey string
	client *graphql.Client
}

func NewMangaNelAPIClient(addr string, apiKey string) *MangaNelAPIClient {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	graphqlClientWithOptions := graphql.WithHTTPClient(client)
	graphqlClient := graphql.NewClient(addr, graphqlClientWithOptions)
	//graphqlClient.Log = func(s string) { slog.Error(s) }

	return &MangaNelAPIClient{
		addr:   addr,
		client: graphqlClient,
		apiKey: apiKey,
	}
}

func (m *MangaNelAPIClient) GetMangaSeriesFull(slug string) (*types.MangaEntity, error) {
	return m.getMangaSeries(slug, true)
}

func (m *MangaNelAPIClient) GetMangaSeriesShort(slug string) (*types.MangaEntity, error) {
	return m.getMangaSeries(slug, false)
}

func (m *MangaNelAPIClient) getMangaSeries(slug string, shouldIncludeChapters bool) (*types.MangaEntity, error) {
	const maxAttempts = 3
	var graphqlResponse any
	var err error
	graphqlRequest := graphql.NewRequest(getQueryForSlug(slug, shouldIncludeChapters))
	graphqlRequest.Header.Add("Origin", "https://manganel.me")
	graphqlRequest.Header.Add("Referer", "https://manganel.me/")
	graphqlRequest.Header.Add("X-Mhub-Access", m.apiKey)
	graphqlRequest.Header.Add("user-agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36")
	graphqlRequest.Header.Add("Sec-Ch-Ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = m.client.Run(context.Background(), graphqlRequest, &graphqlResponse)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	mapResponse, ok := graphqlResponse.(map[string]any)
	if !ok {
		return nil, errors.New("cant cast graphQL response from server to a map")
	}
	mapManga, ok := mapResponse["manga"].(map[string]any)
	if !ok {
		return nil, errors.New("cant cast manga query from graphQL response to a map")
	}

	manga := types.MangaEntity{}
	manga.Name = mapManga["title"].(string)
	manga.Slug = mapManga["slug"].(string)
	manga.Status = types.MangaStatus(mapManga["status"].(string))
	manga.Source = types.MangaSourceMangaNel
	updateLastString := mapManga["updatedDate"].(string)
	timeUpdate, _ := time.Parse(time.RFC3339, updateLastString)
	manga.LastUpdate = timeUpdate
	manga.Chapters = make([]types.ChapterEntity, 0)
	if !shouldIncludeChapters {
		return &manga, nil
	}

	mapChapters, ok := mapManga["chapters"].([]any)
	if !ok {
		return nil, errors.New("cant cast manga chapters query from graphQL response to a map")
	}

	for _, v := range mapChapters {
		s := reflect.ValueOf(v)
		if s.Kind() != reflect.Map {
			panic("InterfaceSlice() given a non-slice type")
		}
		ss, _ := v.(map[string]any)

		number, ok := ss["number"].(float64)
		if !ok {
			slog.Error("cant find chapter number of type float64", "value", ss)
		}
		chapterUpdateTime, ok := ss["date"].(string)
		timeUpdate, _ := time.Parse(time.RFC3339, chapterUpdateTime)

		if !ok {
			slog.Error("cant find chapter slug of type string", "value", ss)
		}
		chapter := types.ChapterEntity{
			Number: &number,
			Date:   &timeUpdate,
			URI:    fmt.Sprintf("https://manganel.me/chapter/%s/chapter-%v", manga.Slug, number),
		}
		manga.Chapters = append(manga.Chapters, chapter)
	}

	sort.Slice(manga.Chapters, func(i, j int) bool { return *manga.Chapters[i].Number > *manga.Chapters[j].Number })

	return &manga, nil
}

// manga(x:mn05,slug:"my-wife-is-a-demon-queen"){id,rank,title,slug,status,image,latestChapter,author,artist,genres,description,alternativeTitle,mainSlug,isYaoi,isPorn,isSoftPorn,unauthFile,noCoverAd,isLicensed,createdDate,updatedDate,chapters{id,number,title,slug,date}}}
func getQueryForSlug(slug string, includeChapters bool) string {
	if includeChapters {
		return fmt.Sprintf(`
	{
		latestPopular(x:mn05) {
			id,
			rank,
			title,
			slug,
			image,
			latestChapter,
			unauthFile,
			updatedDate
		}
		manga(x: mn05, slug: "%s") {
			id,
			rank,
			title,
			slug,
			status,
			image,
			latestChapter,
			author,
			artist,
			genres,
			description,
			alternativeTitle,
			mainSlug,
			isYaoi,
			isPorn,
			isSoftPorn,
			unauthFile,
			noCoverAd,
			isLicensed,
			createdDate,
			updatedDate,
			chapters {
				id,
				number,
				title,
				slug,
				date
			}
		}
	}	
`, slug)
	}
	return fmt.Sprintf(`
{
	manga(x: mn05, slug: "%s") {
		title
		slug
		status
		image
		genres
		latestChapter
		createdDate
		updatedDate
	}
}	
`, slug)
}
