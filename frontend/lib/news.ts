import type { NewsItem } from "./api";

/** Items of `incoming` that are not in `current` yet. */
export function unseen(current: NewsItem[], incoming: NewsItem[]): NewsItem[] {
  const seen = new Set(current.map((i) => i.id));
  return incoming.filter((i) => !seen.has(i.id));
}

/** Both lists in one, without duplicates, in API order: newest first, then higher id. */
export function mergeNews(current: NewsItem[], incoming: NewsItem[]): NewsItem[] {
  return [...current, ...unseen(current, incoming)].sort(
    (a, b) => Date.parse(b.published_at) - Date.parse(a.published_at) || b.id - a.id,
  );
}
