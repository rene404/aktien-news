import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { StockResult } from "../api/types";

export default function Search() {
  const [input, setInput] = useState("");
  const [query, setQuery] = useState("");

  const { data, isFetching } = useQuery({
    queryKey: ["search", query],
    queryFn: async () => {
      const resp = await api.get<{ results: StockResult[] }>("/search", {
        params: { q: query },
      });
      return resp.data.results;
    },
    enabled: query.length > 0,
  });

  return (
    <div>
      <h2>Find a company</h2>
      <form
        className="searchbar"
        onSubmit={(e) => {
          e.preventDefault();
          setQuery(input.trim());
        }}
      >
        <input
          placeholder="Symbol or company name (e.g. AAPL, Apple)"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
        <button type="submit">Search</button>
      </form>

      {isFetching && <p>Searching…</p>}
      {data && data.length === 0 && <p>No matches.</p>}
      <ul className="results">
        {data?.map((r) => (
          <li key={r.stock_id}>
            <Link to={`/stocks/${r.stock_id}`}>
              <strong>{r.symbol}</strong> · {r.company_name}
              <span className="exchange">{r.exchange}</span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
