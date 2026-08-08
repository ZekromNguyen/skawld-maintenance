import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { usePrincipal } from "../usePrincipal";
import { useSite } from "../state/SiteContext";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import type { Evidence } from "../../types";

/** Route an evidence result to its owning surface, when the kind allows. */
function routeFor(item: Evidence): string | null {
  const kind = item.kind.toLowerCase();
  if (kind.includes("document") || kind.includes("knowledge")) {
    // For chunk evidence the source_id is the chunk, not the document.
    return `/knowledge/${item.document_id ?? item.source_id}`;
  }
  if (kind.includes("incident")) {
    return `/incidents/${item.source_id}`;
  }
  return null;
}

/** Mark query terms inside result text. */
function highlight(text: string, query: string): ReactNode {
  const q = query.trim();
  if (!q) return text;
  const index = text.toLowerCase().indexOf(q.toLowerCase());
  if (index === -1) return text;
  return (
    <>
      {text.slice(0, index)}
      <mark>{text.slice(index, index + q.length)}</mark>
      {text.slice(index + q.length)}
    </>
  );
}

/**
 * SearchPage: hybrid knowledge search. Query is URL-synced, results navigate
 * to their owning surface, terms are highlighted, recent queries persist in
 * localStorage, and the input autofocuses.
 */
const RECENT_KEY = "skawld.search.recent";
const RECENT_LIMIT = 5;

function readRecent(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string") : [];
  } catch {
    return [];
  }
}

function writeRecent(query: string) {
  const next = [query, ...readRecent().filter((item) => item !== query)].slice(0, RECENT_LIMIT);
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    // storage unavailable (private mode); recent queries are best-effort
  }
}

type Scope = "ALL" | "DOCUMENT" | "INCIDENT";

const SCOPES: Scope[] = ["ALL", "DOCUMENT", "INCIDENT"];

export function SearchPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const query = (params.get("q") ?? "").trim();
  const { data: principal } = usePrincipal();
  const { siteId } = useSite();
  const [input, setInput] = useState(query);
  const [recent, setRecent] = useState<string[]>(() => readRecent());
  const [scope, setScope] = useState<Scope>("ALL");

  useEffect(() => {
    setInput(query);
  }, [query]);

  const results = useQuery(
    () =>
      query
        ? api.searchKnowledge(siteId ?? "", query, undefined, 20)
        : Promise.resolve({ retrieval_run_id: "", items: [] as Evidence[] }),
    [query, siteId],
  );

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const value = input.trim();
    if (value) writeRecent(value);
    setRecent(readRecent());
    navigate(value ? `/search?q=${encodeURIComponent(value)}` : "/search");
  };

  const scopedItems = useMemo(() => {
    const items = results.data?.items ?? [];
    if (scope === "ALL") return items;
    const wanted = scope.toLowerCase();
    return items.filter((item) => item.kind.toLowerCase().includes(wanted));
  }, [results.data, scope]);

  const groups = useMemo(() => {
    const map = new Map<string, Evidence[]>();
    for (const item of scopedItems) {
      const key = item.authority || t("search.unknownAuthority");
      const bucket = map.get(key) ?? [];
      bucket.push(item);
      map.set(key, bucket);
    }
    return [...map.entries()];
  }, [scopedItems, t]);

  const hasData = scopedItems.length > 0;

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("nav.search")} principal={principal} />
        <form onSubmit={submit} className="search-row" style={{ marginBottom: 18 }}>
          <input
            type="search"
            autoFocus
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder={t("search.placeholder")}
            aria-label={t("search.placeholder")}
          />
          <button className="primary-button" type="submit">{t("search.submit")}</button>
        </form>
        {query ? (
          <div className="incident-tabs" role="tablist" aria-label={t("search.scope")}>
            {SCOPES.map((option) => (
              <button
                key={option}
                role="tab"
                aria-selected={scope === option}
                className={`tab${scope === option ? " active" : ""}`}
                onClick={() => setScope(option)}
              >
                {t(`search.scope.${option.toLowerCase()}` as MessageKey)}
              </button>
            ))}
          </div>
        ) : null}
        {!query && recent.length > 0 ? (
          <div className="panel" style={{ marginBottom: 14 }}>
            <div className="panel-heading"><h2>{t("search.recent")}</h2></div>
            <div className="search-result">
              {recent.map((item) => (
                <Link key={item} to={`/search?q=${encodeURIComponent(item)}`} className="strong">
                  {item}
                </Link>
              ))}
            </div>
          </div>
        ) : null}
        {results.error ? (
          <ErrorState message={results.error} onRetry={() => void results.refetch()} />
        ) : null}
        {!query ? (
          <EmptyState title={t("search.empty")} />
        ) : results.loading && !hasData ? (
          <Skeleton height={200} />
        ) : !hasData ? (
          <EmptyState title={t("search.noResults", { query })} />
        ) : (
          groups.map(([authority, items]) => (
            <div className="panel" key={authority} style={{ marginBottom: 14 }}>
              <div className="panel-heading">
                <h2 className="mono">{authority}</h2>
                <span className="count">{items.length}</span>
              </div>
              <div>
                {items.map((item) => {
                  const to = routeFor(item);
                  const title = to ? (
                    <Link to={to} className="strong">{highlight(item.title, query)}</Link>
                  ) : (
                    <strong>{highlight(item.title, query)}</strong>
                  );
                  return (
                    <div key={item.id} className="search-result">
                      {title}
                      <small className="search-snippet">{highlight(item.content.slice(0, 140), query)}</small>
                      <small className="search-meta">
                        {item.locator} · {item.score.rrf_score.toFixed(2)}
                      </small>
                    </div>
                  );
                })}
              </div>
            </div>
          ))
        )}
      </section>
    </PageTrailProvider>
  );
}
