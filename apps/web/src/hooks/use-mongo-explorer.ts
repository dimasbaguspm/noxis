import { useEffect, useState } from "react";
import {
  useMongoCollectionsQuery,
  useMongoDatabasesQuery,
  useMongoRunMutation,
} from "@/hooks/use-mongodb";

// useMongoExplorer keeps the instance find console: the database and collection fall back to the first listed one, and
// the filter resets whenever the target moves. The driver is the one the page was opened with.
export function useMongoExplorer(driver: string, connection: string) {
  const databases = useMongoDatabasesQuery(driver, connection);
  const [database, setDatabase] = useState("");
  const db = databases.databases.some((d) => d.name === database)
    ? database
    : databases.databases[0]?.name;

  const collections = useMongoCollectionsQuery(driver, connection, db);
  const [collection, setCollection] = useState("");
  const coll = collections.collections.some((c) => c.name === collection)
    ? collection
    : collections.collections[0]?.name;

  const [filter, setFilter] = useState("{}");
  const run = useMongoRunMutation();

  // a stale filter from another collection must never run against the new target.
  useEffect(() => {
    setFilter("{}");
  }, [db, coll]);

  // the last result only counts when it was run against the current target and filter.
  const target = run.variables;
  const matches =
    !!target &&
    target.driver === driver &&
    target.connection === connection &&
    target.body.database === db &&
    target.body.collection === coll &&
    target.body.filter === filter;
  const result = matches ? run.data : undefined;
  const documents = (result?.documents ?? []).map((doc) => {
    try {
      return JSON.parse(doc) as unknown;
    } catch {
      return doc;
    }
  });

  function query() {
    if (!db || !coll || run.isPending) return;
    run.mutate({
      driver,
      connection,
      body: { op: "find", database: db, collection: coll, filter, limit: 50 },
    });
  }

  return {
    databases,
    db,
    setDatabase,
    collections,
    coll,
    setCollection,
    filter,
    setFilter,
    run,
    result,
    documents,
    query,
  };
}
