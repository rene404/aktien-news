import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Watchlist } from "../api/types";

export default function WatchlistPage() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["watchlists"],
    queryFn: async () => {
      const resp = await api.get<Watchlist[]>("/watchlists");
      return resp.data;
    },
  });

  const remove = useMutation({
    mutationFn: async (vars: { watchlistId: string; stockId: string }) => {
      await api.delete(
        `/watchlists/${vars.watchlistId}/stocks/${vars.stockId}`,
      );
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchlists"] }),
  });

  if (isLoading) return <p>Loading…</p>;

  return (
    <div>
      <h2>Watchlists</h2>
      {data?.map((wl) => (
        <div key={wl.id} className="card">
          <h3>{wl.name}</h3>
          {wl.stocks.length === 0 && <p>Empty. Add stocks from a company page.</p>}
          <ul className="results">
            {wl.stocks.map((s) => (
              <li key={s.stock_id}>
                <Link to={`/stocks/${s.stock_id}`}>
                  <strong>{s.symbol}</strong> · {s.company_name}
                </Link>
                <button
                  className="link-btn"
                  onClick={() =>
                    remove.mutate({ watchlistId: wl.id, stockId: s.stock_id })
                  }
                >
                  remove
                </button>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
