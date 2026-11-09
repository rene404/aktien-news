// Source icons: the favicon of each publisher, downloaded once and served from
// public/sources/<publisher>.png (64 px), so visitors' browsers never ask the
// publisher or Google for it. Several feeds share a publisher: the publisher is
// the part of the source key before the first "_" (fn_adhoc, fn_dax → fn).
// A new publisher: add its PNG and its prefix here. The FCA NSM has none: its
// terms forbid any use of the FCA's name or logo.
export const ICON_PUBLISHERS = ["afm", "amf", "cnmv", "fn", "nasdaq", "oekb", "ots", "prn", "sec", "telekom", "voestalpine"] as const;

const known = new Set<string>(ICON_PUBLISHERS);

/** The icon path for a source key, or null when the publisher has none. */
export function sourceIcon(key: string): string | null {
  const publisher = key.split("_")[0];
  return known.has(publisher) ? `/sources/${publisher}.png` : null;
}
