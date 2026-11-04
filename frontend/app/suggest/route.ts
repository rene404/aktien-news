import { NextResponse, type NextRequest } from "next/server";
import { searchInstruments } from "@/lib/api";

// Suggestions for the header search, as the user types. The browser never
// talks to the Go API directly. Empty on short input or any error: the plain
// search form still works.
export async function GET(req: NextRequest) {
  const q = req.nextUrl.searchParams.get("q")?.trim() ?? "";
  let items: { ticker: string; mic: string; name: string }[] = [];
  if (q.length >= 2 && q.length <= 64) {
    try {
      items = (await searchInstruments(q)).map(({ ticker, mic, name }) => ({ ticker, mic, name }));
    } catch {
      items = [];
    }
  }
  return NextResponse.json({ items }, { headers: { "Cache-Control": "public, max-age=60" } });
}
