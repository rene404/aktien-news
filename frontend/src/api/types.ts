export interface UserOut {
  id: string;
  email: string;
  role: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
}

export interface StockResult {
  stock_id: string;
  symbol: string;
  exchange: string;
  company_name: string;
}

export interface NewsItem {
  id: string;
  title: string;
  url: string;
  source_type: string;
  published_at: string | null;
}

export interface NewsList {
  items: NewsItem[];
  total: number;
  limit: number;
  offset: number;
}

export interface WatchlistStockItem {
  stock_id: string;
  symbol: string;
  company_name: string;
}

export interface Watchlist {
  id: string;
  name: string;
  stocks: WatchlistStockItem[];
}

export interface Feed {
  id: string;
  url: string;
  name: string;
  active: boolean;
  last_fetched_at: string | null;
}

export interface ReviewItem {
  news_stock_id: string;
  confidence: number;
  matched_alias: string | null;
  news: { id: string; title: string; url: string };
  stock: { symbol: string; company_name: string };
}

export interface ReviewList {
  items: ReviewItem[];
  total: number;
  limit: number;
  offset: number;
}
