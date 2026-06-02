import { StrictMode, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { api, type Card, type SearchJob, type SearchResult } from "./api";
import "./styles.css";

const presets = ["binder pokemon", "lotto carte pokemon", "collezione carte pokemon"];

function App() {
  const jobId = useMemo(() => location.pathname.match(/^\/searches\/([^/]+)$/)?.[1], []);
  return jobId ? <ResultsPage jobId={jobId} /> : <SearchPage />;
}

function SearchPage() {
  const [query, setQuery] = useState("");
  const [cards, setCards] = useState<Card[]>([]);
  const [selected, setSelected] = useState<Card>();
  const [listingQuery, setListingQuery] = useState("binder pokemon");
  const [error, setError] = useState("");
  const [syncing, setSyncing] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (query.trim().length < 2) return setCards([]);
    const timer = setTimeout(() => api.cards(query).then(setCards).catch((err) => setError(err.message)), 220);
    return () => clearTimeout(timer);
  }, [query]);

  async function syncCatalog() {
    setSyncing(true);
    setError("");
    try {
      await api.syncCatalog();
      setError("Catalogo sincronizzato. Cerca una carta per iniziare.");
    } catch (err) {
      setError(message(err));
    } finally {
      setSyncing(false);
    }
  }

  async function submit() {
    if (!selected || !listingQuery.trim()) return;
    setSubmitting(true);
    setError("");
    try {
      const job = await api.createJob(selected.id, listingQuery.trim());
      location.href = `/searches/${job.id}`;
    } catch (err) {
      setError(message(err));
      setSubmitting(false);
    }
  }

  return (
    <main className="shell">
      <Header />
      <section className="hero">
        <div>
          <p className="eyebrow">Local vision search</p>
          <h1>Trova carte nascoste nei binder.</h1>
          <p className="lede">Seleziona una stampa esatta. Il matcher analizza localmente le foto degli annunci eBay.</p>
        </div>
        <div className="hero-card">{selected ? <img src={selected.imageUrl} alt={selected.name} /> : <span>Anteprima carta</span>}</div>
      </section>
      <section className="panel">
        <div className="field">
          <label htmlFor="card">Carta Pokémon</label>
          <input id="card" value={query} onChange={(event) => { setQuery(event.target.value); setSelected(undefined); }} placeholder="Es. Charizard, Base Set, 4" />
          {cards.length > 0 && !selected && (
            <div className="autocomplete">
              {cards.map((card) => (
                <button key={card.id} onClick={() => { setSelected(card); setQuery(`${card.name} · ${card.setName} #${card.number}`); setCards([]); }}>
                  <img src={card.imageUrl} alt="" /><span><strong>{card.name}</strong><small>{card.setName} · #{card.number}</small></span>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="field">
          <label htmlFor="listing-query">Query annunci</label>
          <input id="listing-query" value={listingQuery} onChange={(event) => setListingQuery(event.target.value)} />
          <div className="presets">{presets.map((preset) => <button key={preset} onClick={() => setListingQuery(preset)}>{preset}</button>)}</div>
        </div>
        {error && <p className="notice">{error}</p>}
        <div className="actions">
          <button className="secondary" onClick={syncCatalog} disabled={syncing}>{syncing ? "Sincronizzazione..." : "Sincronizza catalogo"}</button>
          <button className="primary" onClick={submit} disabled={!selected || submitting}>{submitting ? "Avvio..." : "Cerca negli annunci eBay"}</button>
        </div>
      </section>
    </main>
  );
}

function ResultsPage({ jobId }: { jobId: string }) {
  const [job, setJob] = useState<SearchJob>();
  const [confirmed, setConfirmed] = useState<SearchResult[]>([]);
  const [possible, setPossible] = useState<SearchResult[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    async function refresh() {
      try {
        const next = await api.job(jobId);
        const [confirmedMatches, possibleMatches] = await Promise.all([api.results(jobId, "confirmed"), api.results(jobId, "possible")]);
        if (!active) return;
        setJob(next); setConfirmed(confirmedMatches); setPossible(possibleMatches);
        if (next.status !== "completed" && next.status !== "failed") setTimeout(refresh, 1600);
      } catch (err) {
        if (active) setError(message(err));
      }
    }
    refresh();
    return () => { active = false; };
  }, [jobId]);

  const progress = job?.imagesTotal ? Math.round((job.imagesAnalyzed / job.imagesTotal) * 100) : 0;
  return (
    <main className="shell">
      <Header />
      <a className="back" href="/">← Nuova ricerca</a>
      <section className="result-heading">
        <div><p className="eyebrow">Scansione annunci</p><h1>Risultati</h1><p className="lede">{job?.listingQuery ?? "Caricamento ricerca..."}</p></div>
        <div className={`status ${job?.status}`}>{job?.status ?? "loading"}</div>
      </section>
      {error && <p className="notice error">{error}</p>}
      {job && <section className="progress-panel">
        <div className="progress-label"><span>{job.imagesAnalyzed} / {job.imagesTotal} immagini</span><strong>{progress}%</strong></div>
        <div className="progress-track"><i style={{ width: `${progress}%` }} /></div>
        <div className="metrics"><span>{job.listingsFound} annunci</span><span>{job.confirmedMatches} confermati</span><span>{job.possibleMatches} possibili</span>{job.visionDegraded && <span>Ollama offline: modalità locale</span>}</div>
        {job.error && <p className="notice error">{job.error}</p>}
      </section>}
      <ResultSection title="Match confermati" empty="Nessun match confermato." results={confirmed} />
      <ResultSection title="Possibili match" empty="Nessun possibile match." results={possible} />
    </main>
  );
}

function ResultSection({ title, empty, results }: { title: string; empty: string; results: SearchResult[] }) {
  return <section className="result-section"><h2>{title}</h2>{results.length === 0 ? <p className="empty">{empty}</p> : <div className="result-grid">{results.map((result) => <ResultCard key={result.id} result={result} />)}</div>}</section>;
}

function ResultCard({ result }: { result: SearchResult }) {
  const [size, setSize] = useState({ width: 1, height: 1 });
  const points = result.polygon.map(({ x, y }) => `${x},${y}`).join(" ");
  return <article className="result-card">
    <div className="image-wrap">
      <img src={api.asset(result.cachedImageUrl)} alt={result.listingTitle} onLoad={(event) => setSize({ width: event.currentTarget.naturalWidth, height: event.currentTarget.naturalHeight })} />
      <svg viewBox={`0 0 ${size.width} ${size.height}`} preserveAspectRatio="none"><polygon points={points} /></svg>
    </div>
    <div className="result-body"><div className="confidence">{Math.round(result.confidence * 100)}%</div><h3>{result.listingTitle}</h3><p>{result.reason}</p><a href={result.listingUrl} target="_blank">Apri su eBay ↗</a></div>
  </article>;
}

function Header() { return <header><a href="/" className="brand"><b>Binder</b> Finder</a><span>Local-first MVP</span></header>; }
function message(error: unknown) { return error instanceof Error ? error.message : "Errore inatteso"; }

createRoot(document.getElementById("root")!).render(<StrictMode><App /></StrictMode>);

