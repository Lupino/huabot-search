package main

import (
	"flag"
	"github.com/Lupino/tokenizer"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/char/html"
	"github.com/blevesearch/bleve/index/store/goleveldb"
	"github.com/blevesearch/bleve/mapping"
)

var tokenizerHost = flag.String("tokenizer", "localhost:3000", "tokenizer server host.")

func createMapping() mapping.IndexMapping {
	_mapping, err := newIndexMapping()
	if err != nil {
		panic(err)
	}
	return _mapping
}

func openIndex(path string) (index bleve.Index, err error) {
	if index, err = bleve.Open(path); err != nil {
		_mapping := createMapping()
		if index, err = bleve.NewUsing(path, _mapping, bleve.Config.DefaultIndexType, goleveldb.Name, nil); err != nil {
			return
		}
	}
	return
}

func newIndexMapping() (mapping.IndexMapping, error) {
	var (
		err error
	)
	_mapping := bleve.NewIndexMapping()

	if err = _mapping.AddCustomTokenizer("sego",
		map[string]interface{}{
			"host": *tokenizerHost,
			"type": tokenizer.Name,
		},
	); err != nil {
		return nil, err
	}

	// create a custom analyzer
	if err = _mapping.AddCustomAnalyzer("sego",
		map[string]interface{}{
			"type": custom.Name,
			"char_filters": []string{
				html.Name,
			},
			"tokenizer": "sego",
			"token_filters": []string{
				"possessive_en",
				"to_lower",
				"stop_en",
			},
		},
	); err != nil {
		return nil, err
	}

	_mapping.DefaultAnalyzer = tokenizer.Name
	return _mapping, nil
}
