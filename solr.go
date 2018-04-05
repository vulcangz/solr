package solr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Solr struct {
	CoreUrl string
	Verbose bool
}

func New(coreUrl string, verbose bool) Solr {
	return Solr{CoreUrl: coreUrl, Verbose: verbose}
}

func (s Solr) Get(params GetParams) (Document, error) {
	url := s.CoreUrl + "/select?" + params.toSolrQueryString()
	raw, err := s.httpGet(url)
	if err != nil {
		return Document{}, err
	}

	count := len(raw.Data.Documents)
	if count == 0 {
		return Document{}, nil
	} else if count > 1 {
		msg := fmt.Sprintf("More than one document was found (Q=%s)", params.Q)
		return Document{}, errors.New(msg)
	}
	return raw.Data.Documents[0], err
}

// Issues a search with the values indicated in the paramers
func (s Solr) Search(params SearchParams) (SearchResponse, error) {
	url := s.CoreUrl + "/select?" + params.toSolrQueryString()
	raw, err := s.httpGet(url)
	if err != nil {
		return SearchResponse{}, err
	}
	return NewSearchResponse(params, raw), err
}

// Issues a search for the text indicated and using only
// Solr default values
func (s Solr) SearchText(text string) (SearchResponse, error) {
	options := map[string]string{}
	facets := map[string]string{}
	params := NewSearchParams(text, options, facets)
	return s.Search(params)
}

func (s Solr) httpGet(url string) (responseRaw, error) {
	if s.Verbose {
		log.Printf("Solr URL: %s", url)
	}
	r, err := http.Get(url)
	if err != nil {
		return responseRaw{}, err
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return responseRaw{}, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		msg := fmt.Sprintf("HTTP Status: %s. ", r.Status)
		if len(body) > 0 {
			msg += fmt.Sprintf("Body: %s", body)
		}
		return responseRaw{}, errors.New(msg)
	}

	// log.Printf("Body: %s", body)

	var response responseRaw
	err = json.Unmarshal([]byte(body), &response)
	if err == nil {
		// HTTP request was successful but Solr reported an error.
		if response.Error.Trace != "" {
			msg := fmt.Sprintf("Solr Error. %#v", response.Error)
			err = errors.New(msg)
		}
	} else {
		if len(r.Header["Content-Type"]) > 0 {
			// Perhaps the response was not in JSON
			// (e.g. if Solr returns XML by default)
			msg := fmt.Sprintf("%s. Solr's response Content-Type: %s", err, r.Header["Content-Type"])
			err = errors.New(msg)
		}
	}
	return response, err
}