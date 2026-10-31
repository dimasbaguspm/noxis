import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type SearchRequest, type SearchTarget } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";
import { useUrlState } from "@/hooks/use-url-state";

// SEARCH_DEFAULTS keeps the editor stable across renders: an empty target means "the first one the platform reports".
const SEARCH_DEFAULTS = { target: "", query: "" };

// useSearchUrlParams keeps the picked target and the query document in the URL, so a search survives a reload and a
// reviewer can be sent the exact query that was run.
export function useSearchUrlParams() {
  const [state, patch] = useUrlState(SEARCH_DEFAULTS);
  return {
    target: state.target,
    query: state.query,
    setTarget: (value: string) => patch({ target: value }),
    setQuery: (value: string) => patch({ query: value }),
  };
}

// useSearchTargetsQuery lists what the platform indexes: the page's left column, cached per connection.
export function useSearchTargetsQuery(driver?: string, connection?: string) {
  const query = useQuery({
    queryKey: queryKeys.search.targets(driver, connection),
    queryFn: () => api.search(driver as string).targets(connection as string),
    enabled: !!driver && !!connection,
  });
  return { ...query, targets: (query.data?.targets ?? []) as SearchTarget[] };
}

// useSearchMutation runs one query. It is a mutation rather than a query on purpose: the query document is an editor's
// draft, so caching every keystroke of it would be noise rather than a cache.
export function useSearchMutation(driver?: string, connection?: string) {
  return useMutation({
    mutationFn: (request: SearchRequest) =>
      api.search(driver as string).run(connection as string, request),
  });
}
