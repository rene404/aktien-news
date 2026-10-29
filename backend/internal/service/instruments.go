package service

import (
	"context"
	"strings"
)

type Instrument struct {
	Ticker string
	MIC    string
	Name   string
	ISIN   string
}

type InstrumentDetail struct {
	Instrument
	Country string
	Indices []string
}

type Source struct {
	Key    string
	Name   string
	Market string
	Kind   string
}

// SearchInstruments matches ticker prefix, name substring or exact ISIN.
func (s *Service) SearchInstruments(ctx context.Context, q string) ([]Instrument, error) {
	q = strings.TrimSpace(q)
	if q == "" || len(q) > 64 {
		return nil, invalid("q must be 1-64 characters")
	}
	rows, err := s.q.SearchInstruments(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]Instrument, len(rows))
	for i, r := range rows {
		out[i] = Instrument{Ticker: r.Ticker, MIC: r.Mic, Name: r.Name, ISIN: deref(r.Isin)}
	}
	return out, nil
}

// Instrument resolves a ticker (case-insensitive) for the stock page header.
func (s *Service) Instrument(ctx context.Context, ticker string) (InstrumentDetail, error) {
	if !tickerPattern.MatchString(ticker) {
		return InstrumentDetail{}, invalid("invalid ticker")
	}
	r, err := s.q.InstrumentByTicker(ctx, ticker)
	if err != nil {
		return InstrumentDetail{}, notFound(err)
	}
	indices, err := s.q.ListingIndices(ctx, r.ID)
	if err != nil {
		return InstrumentDetail{}, err
	}
	if indices == nil {
		indices = []string{}
	}
	return InstrumentDetail{
		Instrument: Instrument{Ticker: r.Ticker, MIC: r.Mic, Name: r.Name, ISIN: deref(r.Isin)},
		Country:    r.Country,
		Indices:    indices,
	}, nil
}

func (s *Service) Sources(ctx context.Context) ([]Source, error) {
	rows, err := s.q.ListEnabledSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Source, len(rows))
	for i, r := range rows {
		out[i] = Source{Key: r.Key, Name: r.Name, Market: r.Market, Kind: string(r.Kind)}
	}
	return out, nil
}
