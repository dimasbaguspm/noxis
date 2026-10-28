import { useState } from "react";
import { useMongoCollectionsQuery, useMongoDatabasesQuery } from "@/hooks/use-mongodb";
import type { Change } from "@/lib/api";

// useMongoRequestForm walks the request wizard: the database and collection fall back to the first listed one, and the last rejected validation stays visible.
export function useMongoRequestForm(connection: string) {
  const { databases } = useMongoDatabasesQuery(connection);
  const [selectedDatabase, setSelectedDatabase] = useState("");
  const database = databases.some((item) => item.name === selectedDatabase)
    ? selectedDatabase
    : (databases[0]?.name ?? "");

  const { collections } = useMongoCollectionsQuery(connection, database);
  const [selectedCollection, setSelectedCollection] = useState("");
  const collection = collections.some((item) => item.name === selectedCollection)
    ? selectedCollection
    : (collections[0]?.name ?? "");

  const [step, setStep] = useState(1);
  const [failure, setFailure] = useState<Change>();

  // a new connection or database invalidates every pick below it.
  function chooseConnection() {
    setSelectedDatabase("");
    setSelectedCollection("");
  }

  function chooseDatabase(name: string) {
    setSelectedDatabase(name);
    setSelectedCollection("");
  }

  function chooseCollection(name: string) {
    setSelectedCollection(name);
  }

  return {
    databases,
    collections,
    database,
    collection,
    step,
    setStep,
    failure,
    setFailure,
    chooseConnection,
    chooseDatabase,
    chooseCollection,
  };
}
