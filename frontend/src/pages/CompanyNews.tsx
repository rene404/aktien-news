import { useQuery, useMutation } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import type { NewsList, Watchlist } from "../api/types";

export default function CompanyNews() {
  const { stockId } = useParams<{ stockId: string }>();
  const { user } = useAuth();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["stock-news", stockId],
    queryFn: async () => {
      const resp = await api.get<NewsList>(`/stocks/${stockId}/news`);
      return resp.data;
    },
    enabled: !!stockId,
  });

  const addToWatchlist = useMutation({
    mutationFn: async () => {
      const wl = await api.get<Watchlist[]>("/watchlists");
      const target = wl.data[0];
      await api.post(`/watchlists/${target.id}/stocks`, { stock_id: stockId });
    },
  });

  if (isLoading) return <p>Loading…</p>;
  if (isError) return <p className="error">Company not found.</p>;

  return (
    <div>
      <div className="row">
        <h2>News ({data?.total ?? 0})</h2>
        {user && (
          <button
            onClick={() => addToWatchlist.mutate()}
            disabled={addToWatchlist.isPending || addToWatchlist.isSuccess}
          >
            {addToWatchlist.isSuccess ? "Added ✓" : "+ Add to watchlist"}
          </button>
        )}
      </div>
      {data && data.items.length === 0 && <p>No news linked yet.</p>}
      <ul className="news">
        {data?.items.map((item) => (
          <li key={item.id}>
            <a href={item.url} target="_blank" rel="noreferrer">
              {item.title}
            </a>
            <div className="meta">
              <span className="badge">{item.source_type}</span>
              {item.published_at && (
                <span>{new Date(item.published_at).toLocaleString()}</span>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
