// parseDsn turns a pasted connection string into the fields the connection
// dialog collects. It is deliberately client-side and pure: the API shape and
// the stored connection are unchanged, and the server still validates on probe.

export interface ParsedDsn {
  driver: string;
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
  params: Record<string, string>;
}

export type DsnResult = { ok: true; value: ParsedDsn } | { ok: false; error: string };

// scheme maps to a driver and its default port, mirroring the dialog's
// DRIVER_PORT table and the server's driver vocabulary.
const SCHEMES: Record<string, { driver: string; port: number }> = {
  postgres: { driver: "postgres", port: 5432 },
  postgresql: { driver: "postgres", port: 5432 },
  mysql: { driver: "mysql", port: 3306 },
  mariadb: { driver: "mysql", port: 3306 },
  sqlite: { driver: "sqlite", port: 0 },
  file: { driver: "sqlite", port: 0 },
  mongodb: { driver: "mongo", port: 27017 },
  "mongodb+srv": { driver: "mongo", port: 27017 },
  redis: { driver: "redis", port: 6379 },
  rediss: { driver: "redis", port: 6379 },
  valkey: { driver: "valkey", port: 6379 },
};

const fail = (error: string): DsnResult => ({ ok: false, error });

// decode tolerates a value the URL parser already decoded (or did not encode).
function decode(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function parseDsn(input: string): DsnResult {
  const dsn = input.trim();
  if (!dsn) return fail("paste a connection string");

  // sqlite/file are file paths, not URLs with an authority.
  if (/^(sqlite|file):/i.test(dsn)) {
    const rest = dsn.replace(/^(sqlite|file):\/{0,2}/i, "");
    const [path, query = ""] = rest.split("?");
    if (!path) return fail("a sqlite connection string needs a file path");
    return {
      ok: true,
      value: {
        driver: "sqlite",
        host: "",
        port: 0,
        user: "",
        password: "",
        database: decode(path),
        params: Object.fromEntries(new URLSearchParams(query)),
      },
    };
  }

  let url: URL;
  try {
    url = new URL(dsn);
  } catch {
    return fail("that is not a valid connection string");
  }
  const scheme = url.protocol.replace(/:$/, "").toLowerCase();
  const meta = SCHEMES[scheme];
  if (!meta) return fail(`unsupported scheme "${scheme}"`);

  return {
    ok: true,
    value: {
      driver: meta.driver,
      host: url.hostname,
      port: url.port ? Number(url.port) : meta.port,
      user: decode(url.username),
      password: decode(url.password),
      database: decode(url.pathname.replace(/^\//, "")),
      params: Object.fromEntries(url.searchParams),
    },
  };
}
