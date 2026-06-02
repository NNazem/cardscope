export type Card = { id: string; name: string; setId: string; setName: string; number: string; imageUrl: string };
export type SearchJob = {
  id: string; listingQuery: string; status: "queued" | "fetching" | "analyzing" | "completed" | "failed";
  listingsFound: number; imagesAnalyzed: number; imagesTotal: number; confirmedMatches: number;
  possibleMatches: number; visionDegraded: boolean; error?: string;
};
export type SearchResult = {
  id: string; listingTitle: string; listingUrl: string; cachedImageUrl: string; confidence: number;
  reason: string; polygon: { x: number; y: number }[];
};

const baseURL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, init);
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(body.error ?? response.statusText);
  }
  return response.json();
}

export const api = {
  cards: (query: string) => request<Card[]>(`/api/cards?query=${encodeURIComponent(query)}&limit=12`),
  syncCatalog: () => request<{ status: string }>("/api/catalog/sync", { method: "POST" }),
  createJob: (cardId: string, listingQuery: string) => request<SearchJob>("/api/search-jobs", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ cardId, listingQuery }) }),
  job: (id: string) => request<SearchJob>(`/api/search-jobs/${id}`),
  results: (id: string, bucket: "confirmed" | "possible") => request<SearchResult[]>(`/api/search-jobs/${id}/results?bucket=${bucket}`),
  asset: (path: string) => `${baseURL}${path}`,
};

