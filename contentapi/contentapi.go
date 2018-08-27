package contentapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/spring-media/weltapi"
)

// Service is the Service interface
type Service interface {
	GetContent(id string) (*weltapi.Article, error)
}

// Service holds necessary config values
type config struct {
	h string // host with endpoint
	u string // basic username
	p string // basic password
}

// New initializes a Service connected to the provided Elastic cluster using basic authentication
func New(host, user, pass string) Service {
	return config{h: host, u: user, p: pass}
}

// GetContent gets the Article for its escenic id from frank
func (c config) GetContent(id string) (*weltapi.Article, error) {
	url := c.h + id
	fmt.Printf("Requesting url: %s\n", url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("GET request seems to be invalid.")
		return nil, err
	}

	req.SetBasicAuth(c.u, c.p)
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("http GET failed. returning empty content")
		return nil, err
	}

	temp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ioutil read response body failed. returning empty content")
		return nil, err
	}
	defer resp.Body.Close()

	var a weltapi.APIResponse
	if err := json.Unmarshal(temp, &a); err != nil {
		fmt.Println("Json Unmarshalling failed")
		return nil, err
	}
	return &a.Content, nil
}
