package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Configuration is a type that defines global application configuration
type Configuration struct {
	ElasticsearchURL string
	RealHTTPSPort    uint16
	HTTPSEnabled     bool
	HTTPSCertFile    string
	HTTPSKeyFile     string
}

// Config is a global variable holding configuration of the application
var Config = Configuration{
	RealHTTPSPort: 10443,
	HTTPSEnabled:  false,
}

// DefaultHTTPSPort browser default https port
const DefaultHTTPSPort = 443

// AccentRune is the code of accent rune
const AccentRune = 0x301

func initConfig() {
	Config.ElasticsearchURL = os.Getenv("ELASTICSEARCH_URL")
	if len(Config.ElasticsearchURL) == 0 {
		log.Fatalln("ELASTICSEARCH_URL env var is not set")
	}

	httpsPortStr := os.Getenv("REAL_HTTPS_PORT")
	if len(httpsPortStr) > 0 {
		httpsPort, err := strconv.ParseUint(httpsPortStr, 10, 16)
		if err != nil {
			log.Fatalln("REAL_HTTPS_PORT value is invalid: ", err)
		}
		Config.RealHTTPSPort = uint16(httpsPort)
	}

	_, Config.HTTPSEnabled = os.LookupEnv("HTTPS_ENABLED")
	if Config.HTTPSEnabled {
		var ok bool
		if Config.HTTPSCertFile, ok = os.LookupEnv("HTTPS_CERT_FILE"); !ok {
			log.Fatalln("HTTPS_CERT_FILE is not specified")
		}

		if Config.HTTPSKeyFile, ok = os.LookupEnv("HTTPS_KEY_FILE"); !ok {
			log.Fatalln("HTTPS_KEY_FILE is not specified")
		}
	}

}

func redirect(w http.ResponseWriter, req *http.Request) {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		log.Println("Could not split host and port of ", req.Host)
		host = req.Host
	}

	if Config.RealHTTPSPort != DefaultHTTPSPort {
		host = host + ":" + strconv.FormatUint(uint64(Config.RealHTTPSPort), 10)
	}

	target := "https://" + host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}

	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func highlightAccents(s string) template.HTML {
	var result []byte
	s = norm.NFD.String(s)
	for len(s) > 0 {
		d := norm.NFC.NextBoundaryInString(s, true)

		hasAccentRune := false
		for _, c := range s[:d] {
			if c == AccentRune {
				hasAccentRune = true
				break
			}
		}

		if hasAccentRune {
			result = norm.NFC.AppendString(result, `<span class="accent">`)
		}
		result = norm.NFC.AppendString(result, s[:d])
		if hasAccentRune {
			result = norm.NFC.AppendString(result, `</span>`)
		}

		s = s[d:]
	}
	return template.HTML(result)
}

func removeAccents(s string) string {
	t := transform.RemoveFunc(func(r rune) bool {
		return r == AccentRune
	})

	result, _, _ := transform.String(t, s)
	return result
}

func main() {
	initConfig()
	serveMux := http.NewServeMux()
	rootRouter := mux.NewRouter()
	rootRouter.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
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

			esResp, err := http.Post(Config.ElasticsearchURL+"/dict-*/_search?pretty", "application/json", pipeReader)
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

		funcMap := template.FuncMap{
			"highlightAccents": highlightAccents,
		}

		t, err := template.New("main").Funcs(funcMap).ParseFiles("index.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.ExecuteTemplate(w, "index.gohtml", struct {
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
	rootRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("./statics/")))
	serveMux.Handle("/", rootRouter)

	serveMux.HandleFunc("/_suggest", func(w http.ResponseWriter, request *http.Request) {
		request.ParseForm()
		q := request.Form.Get("q")
		// TODO: handle case when q is empty
		mq, err := json.Marshal(q)
		// TODO: handle error
		_ = err

		resp, err := http.Post(Config.ElasticsearchURL+"/dict-*/_search", "application/json", strings.NewReader(`{
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
			result = append(result, removeAccents(options.Text))
		}

		json.NewEncoder(w).Encode(result)
	})

	if Config.HTTPSEnabled {
		go func() {
			log.Fatalln(http.ListenAndServe(":8080", http.HandlerFunc(redirect)))
		}()
		log.Fatalln(http.ListenAndServeTLS(":10443", Config.HTTPSCertFile, Config.HTTPSKeyFile, serveMux))
	} else {
		log.Fatalln(http.ListenAndServe(":8080", serveMux))
	}
}
