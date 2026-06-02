import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

function App() {
  return (
    <main className="shell">
      <p className="eyebrow">Local vision search</p>
      <h1>Pokémon Binder Finder</h1>
      <p>Trova una carta specifica nelle foto di binder e lotti pubblicati su eBay.</p>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

