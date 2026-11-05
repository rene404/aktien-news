import { NextResponse, type NextRequest } from "next/server";
import { getNews } from "@/lib/api";
import { feedParams } from "@/lib/feed";

// Further pages for infinite scrolling. The browser never talks to the Go API
// directly; this route forwards the same public fields the pages render.
export async function GET(req: NextRequest) {
  const params = feedParams(req.nextUrl.searchParams, new Date());
  if (!params) return NextResponse.json({ error: "invalid ticker" }, { status: 400 });
  try {
    const page = await getNews(params);
    return NextResponse.json(page, { headers: { "Cache-Control": "public, max-age=60" } });
  } catch {
    return NextResponse.json({ error: "upstream error" }, { status: 502 });
  }
}
