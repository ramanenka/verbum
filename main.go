package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var esURL string

func main() {
	esURL = os.Getenv("ELASTICSEARCH_URL")
	if len(esURL) == 0 {
		log.Fatalln("ELASTICSEARCH_URL env var is not set")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		q := request.FormValue("q")
		var hits []map[string]interface{}
		if len(q) > 0 {
			query := struct {
				Query struct {
					MultiMatch struct {
						Query  string   `json:"query"`
						Fields []string `json:"fields"`
					} `json:"multi_match"`
				} `json:"query"`
			}{}
			query.Query.MultiMatch.Query = q
			query.Query.MultiMatch.Fields = []string{"keywords", "translations.value"}

			pipeReader, pipeWriter := io.Pipe()
			go func() {
				defer pipeWriter.Close()
				json.NewEncoder(pipeWriter).Encode(query)
			}()

			esResp, err := http.Post(esURL+"/dict-*/_search?pretty", "application/json", pipeReader)
			if err != nil {
				// TODO: handler error
			}

			data := struct {
				Hits struct {
					Hits []map[string]interface{} `json:"hits"`
				} `json:"hits"`
			}{}

			json.NewDecoder(esResp.Body).Decode(&data)
			hits = data.Hits.Hits
			esResp.Body.Close()
		}

		t, err := template.ParseFiles("index.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, struct {
			Hits []map[string]interface{}
			Q    string
		}{
			hits,
			q,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/_suggest", func(w http.ResponseWriter, request *http.Request) {
		request.ParseForm()
		q := request.Form.Get("q")
		// TODO: handle case when q is empty
		mq, err := json.Marshal(q)
		// TODO: handle error
		_ = err

		resp, err := http.Post(esURL+"/dict-*/_search", "application/json", strings.NewReader(`{
				"_source": false,
				"suggest": {
					"my-suggest": {
						"prefix": `+string(mq)+`,
						"completion": {
					    "field": "typeaheads",
					    "size": 10,
							"fuzzy": {
								"unicode_aware": true
							}
					  }
					}
				}
			}`))
		_ = err
		defer resp.Body.Close()

		m := struct {
			Suggest struct {
				MySuggest []struct {
					Options []struct {
						Text string `json:"text"`
					} `json:"options"`
				} `json:"my-suggest"`
			} `json:"suggest"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&m)
		_ = err
		result := make([]string, 0, len(m.Suggest.MySuggest[0].Options))
		for _, options := range m.Suggest.MySuggest[0].Options {
			result = append(result, options.Text)
		}

		json.NewEncoder(w).Encode(result)
	})

	http.ListenAndServe(":8080", nil)
}
