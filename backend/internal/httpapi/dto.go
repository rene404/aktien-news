package httpapi

import (
	"time"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// Public response contracts. Legal invariant: no body text, snippets,
// sentiment or event types — enforced by TestPublicDTOsCarryNoTextOrScores.
// Every new public DTO must be added to publicDTOs.
var publicDTOs = []any{newsPageDTO{}, instrumentListDTO{}, instrumentDetailDTO{}, sourceListDTO{}, errorDTO{}}

type newsPageDTO struct {
	Items      []newsItemDTO `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

type newsItemDTO struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      sourceRef `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Tickers     []string  `json:"tickers"`
}

type sourceRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type instrumentDTO struct {
	Ticker string `json:"ticker"`
	MIC    string `json:"mic"`
	Name   string `json:"name"`
	ISIN   string `json:"isin,omitempty"`
}

type instrumentListDTO struct {
	Items []instrumentDTO `json:"items"`
}

type instrumentDetailDTO struct {
	instrumentDTO
	Country string   `json:"country"`
	Indices []string `json:"indices"`
}

type sourceDTO struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Market string `json:"market"`
	Kind   string `json:"kind"`
}

type sourceListDTO struct {
	Items []sourceDTO `json:"items"`
}

type errorDTO struct {
	Error string `json:"error"`
}

func toNewsPage(p service.NewsPage) newsPageDTO {
	out := newsPageDTO{Items: make([]newsItemDTO, len(p.Items))}
	for i, it := range p.Items {
		tickers := it.Tickers
		if tickers == nil {
			tickers = []string{}
		}
		out.Items[i] = newsItemDTO{
			ID: it.ID, Title: it.Title, URL: it.URL,
			Source:      sourceRef{Key: it.SourceKey, Name: it.SourceName, Kind: it.SourceKind},
			PublishedAt: it.PublishedAt, Tickers: tickers,
		}
	}
	if p.NextCursor != "" {
		out.NextCursor = &p.NextCursor
	}
	return out
}

func toInstrument(i service.Instrument) instrumentDTO {
	return instrumentDTO{Ticker: i.Ticker, MIC: i.MIC, Name: i.Name, ISIN: i.ISIN}
}
