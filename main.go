package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spring-media/multiple-man/mlt"
	"github.com/spring-media/rbbt-video-recoman/contentapi"
	"github.com/spring-media/rbbt-video-recoman/elastic"
	"github.com/spring-media/weltapi"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

const (
	port   = ":5000"
	index  = "content-search-default"
	testID = "181291926"
)

func router(s elastic.Service, api contentapi.Service, timeout time.Duration) http.Handler {
	router := mux.NewRouter().StrictSlash(true)

	router.Handle("/mlt/{id:[0-9]+}", mltHandler(s, api, timeout)).Methods(http.MethodGet)

	router.HandleFunc("/info", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json;charset=UTF-8")
		rw.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	return router
}

func mltHandler(s elastic.Service, api contentapi.Service, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		c, err := api.GetContent(id)
		if err != nil {
			panic(err)
		} else {
			fmt.Printf("Got Content with url: %s\n", c.WebURL)
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		resp, err := query(ctx, s, c)

		enc := json.NewEncoder(w)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusBadGateway)
			enc.Encode(mlt.Response{Took: 0, Total: 0, Status: "error"})
			return
		}

		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(resp)
	})

}

func main() {
	log.Printf("conntected to '%s@%s/%s'", config.GetString("elastic.user"), config.GetString("elastic.cluster"), index)

	s, err := elastic.New(config.GetString("elastic.cluster"), config.GetString("elastic.user"), config.GetString("elastic.password"), index)
	if err != nil {
		panic(err)
	}

	api := contentapi.New(
		config.GetString("api.host"),
		config.GetString("api.user"),
		config.GetString("api.pass"))

	timeout := time.Duration(config.GetInt("timeout")) * time.Millisecond
	log.Printf("listening on port %s", port)
	log.Fatal(http.ListenAndServe(port, router(s, api, timeout)))

}

func query(ctx context.Context, service elastic.Service, content *weltapi.Article) (*elastic.Response, error) {
	r, err := service.Query(ctx, content)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return r, nil
	// fmt.Println("\n#####")
	// for index, element := range r.Results {
	// 	fmt.Printf("Result %d (%s) with score(%v):\n", index, element.Content.WebURL, element.Score)
	// 	fmt.Print("Tags: ")
	// 	for _, tag := range element.Content.Tags {
	// 		fmt.Printf("%s,", tag.ID)
	// 	}
	// 	fmt.Println("")
	// 	fmt.Print("Keywords: ")
	// 	for _, keyword := range element.Content.Keywords {
	// 		fmt.Printf("%s,", keyword.Label)
	// 	}
	// 	fmt.Println("\n#####")
	// }
}
func init() {
	// defaults
	config.SetDefault("elastic.cluster", "http://localhost:9200")
	config.SetDefault("elastic.user", "elastic")
	config.SetDefault("elastic.password", "changeme")
	config.SetDefault("timeout", 500)

	config.SetDefault("api.host", "https://frank-ecs-production.up.welt.de/content/")
	config.SetDefault("api.user", "user")
	config.SetDefault("api.pass", "thisisnotapassword")

	// enable ENV overrides
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// optional config file
	config.AddConfigPath(".")
	config.SetConfigName("config")
	if err := config.ReadInConfig(); err == nil {
		log.Info("Using config file:", config.ConfigFileUsed())
	}

}
