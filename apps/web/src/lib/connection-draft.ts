import { z } from "zod";
import type { Connection } from "@/lib/api";

// A driver's draft and the fields it needs live here, so the create page and the instance picker agree on them.

const FILE_DRIVERS = new Set(["sqlite"]);
const CACHE_DRIVERS = new Set(["redis", "valkey"]);

// isFileDriver reports a driver whose "host" is a file path and which has no user.
export const isFileDriver = (driver?: string) => FILE_DRIVERS.has(driver ?? "");

// DRIVER_PORT is the port each engine listens on by default; the catalog still decides which drivers exist.
const DRIVER_PORT: Record<string, number> = {
  postgres: 5432,
  mysql: 3306,
  mariadb: 3306,
  redis: 6379,
  valkey: 6379,
  mongo: 27017,
};

// draftFor is the empty draft of one driver: its port, and the database it usually listens on.
export function draftFor(driver: string): Connection {
  return {
    name: "",
    driver,
    host: "",
    port: isFileDriver(driver) ? 0 : (DRIVER_PORT[driver] ?? 5432),
    user: "",
    password: "",
    database: CACHE_DRIVERS.has(driver) ? "0" : driver === "mongo" ? "admin" : "",
  };
}

// stepsFor is how many steps a driver's form needs: a file-backed database has no credentials to enter.
export const stepsFor = (driver: string) => (isFileDriver(driver) ? 2 : 3);

// connectionSchema validates shared fields; the server probe enforces driver-specific host or file-path requirements.
export const connectionSchema = z
  .object({
    name: z.string().trim().min(1, "Name is required"),
    host: z.string().trim().optional(),
    port: z.coerce.number().int().min(0).max(65535, "Port must be between 1 and 65535").optional(),
    user: z.string().trim().optional(),
    password: z.string().optional(),
    database: z.string().trim().min(1, "Database is required"),
    params: z.record(z.string(), z.string()).optional(),
    driver: z.string().optional(),
  })
  .superRefine((v, ctx) => {
    if (isFileDriver(v.driver)) return;
    if (!v.host) ctx.addIssue({ code: "custom", path: ["host"], message: "host is required" });
    if (!v.port)
      ctx.addIssue({ code: "custom", path: ["port"], message: "Port must be between 1 and 65535" });
    if (!v.user) ctx.addIssue({ code: "custom", path: ["user"], message: "user is required" });
  });

export type ConnectionForm = z.infer<typeof connectionSchema>;
