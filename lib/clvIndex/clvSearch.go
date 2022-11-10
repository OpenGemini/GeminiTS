package clvIndex

import (
	"github.com/openGemini/openGemini/lib/utils"
	"github.com/openGemini/openGemini/lib/vGram/gramTextSearch/gramFuzzyQuery"
	"github.com/openGemini/openGemini/lib/vGram/gramTextSearch/gramMatchQuery"
	"github.com/openGemini/openGemini/lib/vGram/gramTextSearch/gramRegexQuery"
	"github.com/openGemini/openGemini/lib/vToken/tokenTextSearch/tokenFuzzyQuery"
	"github.com/openGemini/openGemini/lib/vToken/tokenTextSearch/tokenMatchQuery"
	"github.com/openGemini/openGemini/lib/vToken/tokenTextSearch/tokenRegexQuery"
)

type QuerySearch int32

const (
	MATCHSEARCH QuerySearch = 0
	FUZZYSEARCH QuerySearch = 1
	REGEXSEARCH QuerySearch = 2
)
const ED = 2

type QueryOption struct {
	measurement string
	fieldKey    string
	querySearch QuerySearch
	queryString string
}

func NewQueryOption(measurement string, fieldKey string, search QuerySearch, queryString string) QueryOption {
	return QueryOption{
		measurement: measurement,
		fieldKey:    fieldKey,
		querySearch: search,
		queryString: queryString,
	}
}

func CLVSearchIndex(clvType CLVIndexType, dicType CLVDicType, queryOption QueryOption, dictionary *CLVDictionary, index *CLVIndexNode) []utils.SeriesId {
	var res []utils.SeriesId
	if queryOption.querySearch == MATCHSEARCH {
		if clvType == VGRAM {
			res = MatchSearchVGramIndex(dicType, queryOption.queryString, dictionary, index)
		}
		if clvType == VTOKEN {
			res = MatchSearchVTokenIndex(dicType, queryOption.queryString, dictionary, index)
		}
	}
	if queryOption.querySearch == FUZZYSEARCH {
		if clvType == VGRAM {
			res = FuzzySearchVGramIndex(dicType, queryOption.queryString, dictionary, index)
		}
		if clvType == VTOKEN {
			res = FuzzySearchVTokenIndex(dicType, queryOption.queryString, index)
		}
	}
	if queryOption.querySearch == REGEXSEARCH {
		if clvType == VGRAM {
			res = RegexSearchVGramIndex(dicType, queryOption.queryString, dictionary, index)
		}
		if clvType == VTOKEN {
			res = RegexSearchVTokenIndex(dicType, queryOption.queryString, index)
		}
	}
	return res
}

func MatchSearchVGramIndex(dicType CLVDicType, queryStr string, dictionary *CLVDictionary, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = gramMatchQuery.MatchSearch(queryStr, dictionary.VgramClvcDicRoot.Root(), index.VgramClvcIndexRoot.Root(), QMINGRAM)
	}
	if dicType == CLVL {
		res = gramMatchQuery.MatchSearch(queryStr, dictionary.VgramClvlDicRoot.Root(), index.VgramClvlIndexRoot.Root(), QMINGRAM)
	}
	return res
}

func MatchSearchVTokenIndex(dicType CLVDicType, queryStr string, dictionary *CLVDictionary, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = tokenMatchQuery.MatchSearch(queryStr, dictionary.VtokenClvcDicRoot.Root(), index.VtokenClvcIndexRoot.Root(), QMINTOKEN)
	}
	if dicType == CLVL {
		res = tokenMatchQuery.MatchSearch(queryStr, dictionary.VtokenClvlDicRoot.Root(), index.VtokenClvlIndexRoot.Root(), QMINTOKEN)
	}
	return res
}

func FuzzySearchVGramIndex(dicType CLVDicType, queryStr string, dictionary *CLVDictionary, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = gramFuzzyQuery.FuzzyQueryGramQmaxTrie(index.LogTreeRoot.Root(), queryStr, dictionary.VgramClvcDicRoot.Root(), index.VgramClvcIndexRoot.Root(), QMINGRAM, LOGTREEMAX, ED)
	}
	if dicType == CLVL {
		res = gramFuzzyQuery.FuzzyQueryGramQmaxTrie(index.LogTreeRoot.Root(), queryStr, dictionary.VgramClvlDicRoot.Root(), index.VgramClvlIndexRoot.Root(), QMINGRAM, LOGTREEMAX, ED)
	}
	return res
}

func FuzzySearchVTokenIndex(dicType CLVDicType, queryStr string, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = tokenFuzzyQuery.FuzzySearchComparedWithES(queryStr, index.VtokenClvcIndexRoot.Root(), ED)
	}
	if dicType == CLVL {
		res = tokenFuzzyQuery.FuzzySearchComparedWithES(queryStr, index.VtokenClvlIndexRoot.Root(), ED)
	}
	return res
}

func RegexSearchVGramIndex(dicType CLVDicType, queryStr string, dictionary *CLVDictionary, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = gramRegexQuery.RegexSearch(queryStr, dictionary.VgramClvcDicRoot, index.VgramClvcIndexRoot)
	}
	if dicType == CLVL {
		res = gramRegexQuery.RegexSearch(queryStr, dictionary.VgramClvlDicRoot, index.VgramClvlIndexRoot)
	}
	return res
}

func RegexSearchVTokenIndex(dicType CLVDicType, queryStr string, index *CLVIndexNode) []utils.SeriesId {
	var res = make([]utils.SeriesId, 0)
	if dicType == CLVC {
		res = tokenRegexQuery.RegexSearch(queryStr, index.VtokenClvcIndexRoot)
	}
	if dicType == CLVL {
		res = tokenRegexQuery.RegexSearch(queryStr, index.VtokenClvlIndexRoot)
	}
	return res
}