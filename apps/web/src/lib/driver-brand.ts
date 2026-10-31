// Every driver the console can draw is named here with its monogram and the CSS token that carries its brand colour in
// the current theme. A driver the console has never seen still gets a mark, derived from its name.

export interface DriverBrand {
  // monogram is the two letters a driver tile shows, because eight engines cannot be told apart by one glyph.
  monogram: string;
  // token is the custom property the stylesheet defines twice, once per theme.
  token: string;
}

const BRANDS: Record<string, { monogram: string; token: string }> = {
  postgres: { monogram: "Pg", token: "--driver-postgres" },
  mysql: { monogram: "My", token: "--driver-mysql" },
  mariadb: { monogram: "Ma", token: "--driver-mariadb" },
  sqlite: { monogram: "Sq", token: "--driver-sqlite" },
  cockroach: { monogram: "Cr", token: "--driver-cockroach" },
  redis: { monogram: "Rd", token: "--driver-redis" },
  valkey: { monogram: "Vk", token: "--driver-valkey" },
  dragonfly: { monogram: "Df", token: "--driver-dragonfly" },
  mongo: { monogram: "Mo", token: "--driver-mongo" },
  couchdb: { monogram: "Cb", token: "--driver-couchdb" },
  elasticsearch: { monogram: "Es", token: "--driver-elasticsearch" },
  meilisearch: { monogram: "Me", token: "--driver-meilisearch" },
  splunk: { monogram: "Sp", token: "--driver-splunk" },
};

const UNKNOWN = "--driver-unknown";

// brandOf resolves a driver to its monogram and colour token; an unknown driver keeps its first two letters.
export function brandOf(driver?: string): DriverBrand {
  const key = (driver ?? "").trim().toLowerCase();
  const known = BRANDS[key];
  if (known) return known;
  const monogram = key ? key.charAt(0).toUpperCase() + key.charAt(1) : "??";
  return { monogram, token: UNKNOWN };
}
