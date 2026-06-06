import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import axios, { AxiosError } from "axios";
import type { AxiosAdapter, AxiosResponse, InternalAxiosRequestConfig } from "axios";
import { api, tokenStore } from "./client";

function ok(config: InternalAxiosRequestConfig, data: unknown = {}): AxiosResponse {
  return {
    data,
    status: 200,
    statusText: "OK",
    headers: {},
    config,
  };
}

// A custom adapter is responsible for honouring validateStatus itself, so a
// 401 must be surfaced as a rejected AxiosError (with .response/.config set)
// the way the real HTTP adapter would.
function fail401(config: InternalAxiosRequestConfig): Promise<never> {
  const response: AxiosResponse = {
    data: {},
    status: 401,
    statusText: "Unauthorized",
    headers: {},
    config,
  };
  return Promise.reject(
    new AxiosError("Unauthorized", "ERR_BAD_REQUEST", config, null, response),
  );
}

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("tokenStore", () => {
  it("stores and reads access + refresh tokens", () => {
    tokenStore.set("acc", "ref");
    expect(tokenStore.access).toBe("acc");
    expect(tokenStore.refresh).toBe("ref");
  });

  it("set without refresh leaves the existing refresh untouched", () => {
    tokenStore.set("acc1", "ref1");
    tokenStore.set("acc2");
    expect(tokenStore.access).toBe("acc2");
    expect(tokenStore.refresh).toBe("ref1");
  });

  it("clear removes both tokens", () => {
    tokenStore.set("acc", "ref");
    tokenStore.clear();
    expect(tokenStore.access).toBeNull();
    expect(tokenStore.refresh).toBeNull();
  });
});

describe("request interceptor", () => {
  it("attaches a bearer header when a token is present", async () => {
    tokenStore.set("tok123");
    let seen: string | undefined;
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      seen = config.headers.Authorization as string;
      return ok(config);
    }) as AxiosAdapter;

    await api.get("/anything");
    expect(seen).toBe("Bearer tok123");
  });

  it("sends no bearer header when there is no token", async () => {
    let seen: unknown;
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      seen = config.headers.Authorization;
      return ok(config);
    }) as AxiosAdapter;

    await api.get("/anything");
    expect(seen).toBeUndefined();
  });
});

describe("401 -> refresh -> replay", () => {
  it("refreshes once and replays the original request with the new token", async () => {
    tokenStore.set("stale", "refresh-tok");
    const post = vi
      .spyOn(axios, "post")
      .mockResolvedValue({ data: { access_token: "fresh" } });

    const seenAuth: (string | undefined)[] = [];
    let calls = 0;
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      calls += 1;
      seenAuth.push(config.headers.Authorization as string | undefined);
      if (calls === 1) return fail401(config);
      return ok(config, { ok: true });
    }) as AxiosAdapter;

    const resp = await api.get("/protected");

    expect(resp.status).toBe(200);
    expect(calls).toBe(2); // original + one replay
    expect(post).toHaveBeenCalledOnce(); // one refresh attempt
    expect(seenAuth[0]).toBe("Bearer stale");
    expect(seenAuth[1]).toBe("Bearer fresh"); // replay uses the new token
    expect(tokenStore.access).toBe("fresh"); // store updated
  });

  it("rejects without retrying when there is no refresh token", async () => {
    tokenStore.set("stale"); // no refresh token
    let calls = 0;
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
      calls += 1;
      return fail401(config);
    }) as AxiosAdapter;

    await expect(api.get("/protected")).rejects.toBeDefined();
    expect(calls).toBe(1); // no replay attempted
  });

  it("clears tokens when the refresh request itself fails", async () => {
    tokenStore.set("stale", "refresh-tok");
    vi.spyOn(axios, "post").mockRejectedValue(new Error("refresh boom"));
    api.defaults.adapter = (async (config: InternalAxiosRequestConfig) =>
      fail401(config)) as AxiosAdapter;

    await expect(api.get("/protected")).rejects.toBeDefined();
    expect(tokenStore.access).toBeNull();
    expect(tokenStore.refresh).toBeNull();
  });
});
