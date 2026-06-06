import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Feed } from "../../api/types";

export default function Feeds() {
  const qc = useQueryClient();
  const [url, setUrl] = useState("");
  const [name, setName] = useState("");

  const { data } = useQuery({
    queryKey: ["admin-feeds"],
    queryFn: async () => (await api.get<Feed[]>("/admin/feeds")).data,
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-feeds"] });

  const create = useMutation({
    mutationFn: async () => {
      await api.post("/admin/feeds", { url, name });
    },
    onSuccess: () => {
      setUrl("");
      setName("");
      invalidate();
    },
  });

  const toggle = useMutation({
    mutationFn: async (f: Feed) => {
      await api.patch(`/admin/feeds/${f.id}`, { active: !f.active });
    },
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/admin/feeds/${id}`);
    },
    onSuccess: invalidate,
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate();
  }

  return (
    <div>
      <h2>RSS Feeds</h2>
      <form className="card form-inline" onSubmit={onSubmit}>
        <input
          placeholder="Feed name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <input
          placeholder="https://example.com/feed.rss"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          required
        />
        <button type="submit" disabled={create.isPending}>
          Add feed
        </button>
      </form>
      {create.isError && <p className="error">Could not add feed (duplicate URL?)</p>}

      <table className="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>URL</th>
            <th>Active</th>
            <th>Last fetched</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {data?.map((f) => (
            <tr key={f.id}>
              <td>{f.name}</td>
              <td className="url">{f.url}</td>
              <td>
                <button className="link-btn" onClick={() => toggle.mutate(f)}>
                  {f.active ? "on" : "off"}
                </button>
              </td>
              <td>
                {f.last_fetched_at
                  ? new Date(f.last_fetched_at).toLocaleString()
                  : "—"}
              </td>
              <td>
                <button className="link-btn" onClick={() => remove.mutate(f.id)}>
                  delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
