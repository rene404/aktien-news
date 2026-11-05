package httpapi

import (
	"time"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// Public response contracts. Legal invariant: no body text, snippets,
// sentiment or event types — enforced by TestPublicDTOsCarryNoTextOrScores.
// Every new public DTO must be added to publicDTOs.
var publicDTOs = []any{newsPageDTO{}, instrumentListDTO{}, instrumentDetailDTO{}, sourceListDTO{}, tickerCountListDTO{}, errorDTO{}}

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
	// The same news at other sources: title, link, source and time only.
	Also []alsoDTO `json:"also"`
}

type alsoDTO struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      sourceRef `json:"source"`
	PublishedAt time.Time `json:"published_at"`
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
	// Last successful fetch (UTC); null before the first.
	LastOKAt *time.Time `json:"last_ok_at"`
}

type sourceListDTO struct {
	Items []sourceDTO `json:"items"`
}

// A ticker with the number of news items (clusters) it appears in.
type tickerCountDTO struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

type tickerCountListDTO struct {
	Items []tickerCountDTO `json:"items"`
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
			PublishedAt: it.PublishedAt, Tickers: tickers, Also: make([]alsoDTO, len(it.Also)),
		}
		for j, a := range it.Also {
			out.Items[i].Also[j] = alsoDTO{
				Title: a.Title, URL: a.URL,
				Source:      sourceRef{Key: a.SourceKey, Name: a.SourceName, Kind: a.SourceKind},
				PublishedAt: a.PublishedAt,
			}
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
