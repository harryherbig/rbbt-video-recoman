package elastic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic"
	"github.com/spring-media/weltapi"
)

// Service interface with business functions to find similar content
type Service interface {
	Query(ctx context.Context, content *weltapi.Article) (*Response, error)
}

type service struct {
	client *elastic.Client
	index  string
}

// Response which is serializable as JSON
type Response struct {
	Status  string            `json:"status"`
	Total   int64             `json:"total"`
	Took    int64             `json:"took"`
	Results []weltapi.Article `json:"results"`
}

// ResultWrapper has api content and matching score
type ResultWrapper struct {
	Content *weltapi.Article `json:"content"`
	Score   float64          `json:"score"`
}

// New initializes a Service connected to the provided Elastic cluster using basic authentication
func New(cluster, user, pass, index string) (Service, error) {
	c, err := elastic.NewSimpleClient(
		elastic.SetURL(cluster),
		elastic.SetBasicAuth(user, pass),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)

	if err != nil {
		return nil, err
	}

	return &service{client: c, index: index}, nil
}

// Query delivers similar articles for the given id based on it's tags
func (c *service) Query(ctx context.Context, content *weltapi.Article) (*Response, error) {
	allKeywords := make([]string, 0)
	// for _, tag := range content.Tags {
	// 	allKeywords = appendIfMissing(allKeywords, strings.Replace(tag.ID, ",", "", -1))
	// }

	for _, keyword := range content.Keywords {
		if keyword.Score > 5.0 {
			// allKeywords = appendIfMissing(allKeywords, strings.Replace(keyword.Label, ",", "", -1))
			allKeywords = append(allKeywords, keyword.Label)
		}
	}

	s := make([]interface{}, len(allKeywords))
	for i, v := range allKeywords {
		s[i] = v
	}

	fmt.Println("\nDone collecting Keywords")

	if len(allKeywords) == 0 {
		panic("No keywords provided")
	}

	// joined := strings.Join(allKeywords[:], ";")
	// fmt.Printf("Searching for contents with tags: %s", joined)
	boolQuery := elastic.NewBoolQuery()

	// mlt := elastic.NewMoreLikeThisQuery().
	// 	Field("keywords.label"). //,
	// 	LikeItems(elastic.NewMoreLikeThisQueryItem().
	// 		Id(content.ID).
	// 		Index(c.index)).
	// 	MinDocFreq(1)
	// fmt.Printf("Searching mlt with seoTitle: %s\n", content.Fields["seoTitle"])

	boolQuery.
		// Must(mlt).
		Must(elastic.NewTermsQuery("keywords.label", "RTL", "Lombardi")).
		// Must(elastic.NewBoolQuery().
		// 	Should(elastic.NewTermsQuery("tags.id", s...)).
		// 	Should(elastic.NewTermsQuery("keywords.label", s...))).

		// Should(elastic.NewTermsQuery("keywords.label", "RTL")).

		// Filter(elastic.NewRangeQuery("fields.publicationDate").Gte("now-30d/d")).
		// Filter(elastic.NewRangeQuery("metadata.validFromDate").Lt("now/m")).
		// Filter(elastic.NewRangeQuery("metadata.validToDate").Gt("now/m")).
		// Filter(elastic.NewTermQuery("metadata.state", "published")).
		Must(elastic.NewTermsQuery("type", "video")).
		// MustNot(elastic.NewPrefixQuery("sections.home", "/testgpr/")).
		// MustNot(elastic.NewPrefixQuery("sections.home", "/out-of-home/")).
		MustNot(elastic.NewTermQuery("fields.hiddenArticle", "true"))

	// fmt.Printf("Search:\n%v\n", *boolQuery)

	searchResult, err := c.client.Search().
		Index(c.index).
		Query(boolQuery).
		// Sort("_score", false).
		Sort("fields.publicationDate", false).
		Size(5).
		Pretty(true).
		FetchSourceContext(elastic.NewFetchSourceContext(true).
			Include(
				"fields.headline",
				"fields.intro",
				"fields.lastModifiedDate",
				"fields.publicationDate",
				"fields.topic",
				"fields.urlTitle",
				"id",
				"sections",
				"subType",
				"type",
				"webUrl",
				"tags",
				"keywords",
			)).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	fmt.Printf("\nResults: %d\n", len(searchResult.Hits.Hits))

	response := Response{Took: searchResult.TookInMillis, Total: int64(len(searchResult.Hits.Hits)), Status: "ok", Results: make([]weltapi.Article, 0)}
	for _, hit := range searchResult.Hits.Hits {
		var a weltapi.Article

		if err := json.Unmarshal(*hit.Source, &a); err == nil {
			if len(a.Keywords) < 1 {
				fmt.Printf("%s has no keywords\n", a.ID)
			}
			response.Results = append(response.Results, a)
		}
	}
	return &response, nil
}

func appendIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			fmt.Printf(" not unique: %s |", s)
			return slice
		}
	}
	return append(slice, s)
}
