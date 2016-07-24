package main

import (
	"flag"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/document"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/mholt/binding"
	"github.com/unrolled/render"
	"log"
	"net/http"
	"strconv"
)

var r = render.New()

func sendJSONResponse(w http.ResponseWriter, status int, key string, data interface{}) {
	if key == "" {
		r.JSON(w, status, data)
	} else {
		r.JSON(w, status, map[string]interface{}{key: data})
	}
}

var (
	root string
	host string
)

func init() {
	flag.StringVar(&host, "host", "localhost:3030", "The patent search server host.")
	flag.StringVar(&root, "work_dir", ".", "The patent work dir.")
	flag.Parse()
}

func main() {
	var router = mux.NewRouter()
	var path = root + "/patent-search.db"
	var docIndex, _ = openIndex(path)

	router.HandleFunc("/api/docs/", func(w http.ResponseWriter, req *http.Request) {
		doc := new(Document)
		errs := binding.Bind(req, doc)
		if errs.Handle(w) {
			return
		}
		if err := docIndex.Index(doc.ID, doc); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, "result", "OK")
	}).Methods("POST")

	router.HandleFunc("/api/docs/{id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		id := vars["id"]
		var doc, err = docIndex.Document(id)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if doc == nil {
			sendJSONResponse(w, http.StatusNotFound, "err", "doc["+id+"] not exists.")
			return
		}
		var realDoc Document
		realDoc.ID = doc.ID
		for _, field := range doc.Fields {
			switch field.Name() {
			case "title":
				realDoc.Title = string(field.Value())
			case "summary":
				realDoc.Summary = string(field.Value())
			case "content":
				realDoc.Content = string(field.Value())
			case "created_at":
				v, _ := field.(*document.NumericField).Number()
				realDoc.CreatedAt = int64(v)
			}
		}
		sendJSONResponse(w, http.StatusOK, "", realDoc)
	}).Methods("GET")

	router.HandleFunc("/api/docs/{id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		id := vars["id"]
		if err := docIndex.Delete(id); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, "result", "OK")
	}).Methods("DELETE")

	router.HandleFunc("/api/search/", func(w http.ResponseWriter, req *http.Request) {
		var (
			qs    = req.URL.Query()
			err   error
			from  int
			size  int
			total uint64
			q     = qs.Get("q")
			docs  []ResultID
		)
		if from, err = strconv.Atoi(qs.Get("from")); err != nil {
			from = 0
		}

		if size, err = strconv.Atoi(qs.Get("size")); err != nil {
			size = 10
		}

		if size > 100 {
			size = 100
		}

		if q == "" {
			sendJSONResponse(w, http.StatusBadRequest, "err", "q is required.")
			return
		}

		query, err := bleve.ParseQuery([]byte(q))
		if err != nil {
			query = bleve.NewQueryStringQuery(q)
		}
		searchRequest := bleve.NewSearchRequestOptions(query, size, from, false)
		searchResult, err := docIndex.Search(searchRequest)
		if err != nil {
			log.Printf("bleve.Index.Search() failed(%s)", err)
			sendJSONResponse(w, http.StatusBadRequest, "err", err)
			return
		}

		for _, hit := range searchResult.Hits {
			docs = append(docs, ResultID{ID: hit.ID, Score: hit.Score})
		}

		total = searchResult.Total

		sendJSONResponse(w, http.StatusOK, "", map[string]interface{}{
			"total": total,
			"from":  from,
			"size":  size,
			"q":     q,
			"docs":  docs,
		})
	}).Methods("GET")

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(router)
	n.Run(host)
}
