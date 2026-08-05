import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Topbar } from "../layout/Topbar";

/**
 * SearchPage: hybrid knowledge search. Results are grouped by evidence
 * authority. The API /search returns knowledge Evidence only; the tabs
 * filter the already-loaded lists client-side where available.
 */
export function SearchPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const query = (params.get("q") ?? "").trim();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0] ?? "";
  const [input, setInput] = useState(query);

  const results = useApi(() =>
    query
      ? api.searchKnowledge(siteID, query)
      : Promise.resolve({ retrieval_run_id: "", items: [] as import("../../types").Evidence[] }),
  );

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const value = input.trim();
    navigate(value ? `/search?q=${encodeURIComponent(value)}` : "/search");
  };

  const groups = new Map<string, import("../../types").Evidence[]>();
  for (const item of results.data?.items ?? []) {
    const key = item.authority || t("search.unknownAuthority");
    const bucket = groups.get(key) ?? [];
    bucket.push(item);
    groups.set(key, bucket);
  }

  return (
    <section>
      <Topbar title={t("nav.search")} principal={principal} />
      <form onSubmit={submit} className="search-row" style={{ marginBottom: 18 }}>
        <input
          type="search"
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder={t("search.placeholder")}
          aria-label={t("search.placeholder")}
        />
        <button className="primary-button" type="submit">{t("search.submit")}</button>
      </form>
      {results.error && (
        <div className="toast-error" role="alert">{results.error}</div>
      )}
      {results.loading && query && (
        <div className="skeleton" style={{ height: 200 }} />
      )}
      {!query && (
        <div className="empty" style={{ padding: 40 }}>{t("search.empty")}</div>
      )}
      {query && !results.loading && results.data?.items.length === 0 && (
        <div className="empty" style={{ padding: 40 }}>{t("search.noResults")}</div>
      )}
      {query &&
        !results.loading &&
        groups.size > 0 &&
        [...groups.entries()].map(([authority, items]) => (
          <div className="panel" key={authority} style={{ marginBottom: 14 }}>
            <div className="panel-heading">
              <h2 className="mono">{authority}</h2>
              <span className="count">{items.length}</span>
            </div>
            <div>
              {items.map((item) => (
                <div
                  key={item.id}
                  style={{
                    padding: "12px 16px",
                    borderBottom: "1px solid var(--line)",
                    fontSize: 13
                  }}
                >
                  <strong>{item.title}</strong>
                  <small style={{ display: "block", color: "var(--ink-muted)", marginTop: 3 }}>
                    {item.content.slice(0, 140)}
                  </small>
                  <small style={{ display: "block", color: "var(--ink-muted)", marginTop: 3 }}>
                    {t("search.score")} {item.score.rrf_score.toFixed(2)} · {item.locator}
                  </small>
                </div>
              ))}
            </div>
          </div>
        ))}
    </section>
  );
}
