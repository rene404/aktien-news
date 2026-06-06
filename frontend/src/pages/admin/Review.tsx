import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { ReviewList } from "../../api/types";

export default function Review() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["admin-review"],
    queryFn: async () => (await api.get<ReviewList>("/admin/review")).data,
  });

  const decide = useMutation({
    mutationFn: async (vars: { id: string; decision: "approve" | "reject" }) => {
      await api.post(`/admin/review/${vars.id}`, { decision: vars.decision });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-review"] }),
  });

  if (isLoading) return <p>Loading…</p>;

  return (
    <div>
      <h2>Review queue ({data?.total ?? 0})</h2>
      {data && data.items.length === 0 && <p>Nothing to review. 🎉</p>}
      <ul className="review">
        {data?.items.map((item) => (
          <li key={item.news_stock_id} className="card">
            <div className="row">
              <span>
                <strong>{item.stock.symbol}</strong> · {item.stock.company_name}
                <span className="badge">
                  conf {item.confidence.toFixed(2)}
                </span>
                {item.matched_alias && (
                  <span className="alias">matched “{item.matched_alias}”</span>
                )}
              </span>
            </div>
            <a href={item.news.url} target="_blank" rel="noreferrer">
              {item.news.title}
            </a>
            <div className="actions">
              <button
                onClick={() =>
                  decide.mutate({ id: item.news_stock_id, decision: "approve" })
                }
              >
                Approve
              </button>
              <button
                className="secondary"
                onClick={() =>
                  decide.mutate({ id: item.news_stock_id, decision: "reject" })
                }
              >
                Reject
              </button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
