import { Database } from "bun:sqlite";
import { randomUUID } from "crypto";

const db = new Database("analytics.db");

// Create table if it doesn't exist
db.exec(`
  CREATE TABLE IF NOT EXISTS instances (
    uuid TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    last_seen INTEGER NOT NULL
  )
`);

const versions = [
  { version: "v4.1.0", count: 1988 },
  { version: "v4.0.1", count: 267 },
  { version: "v4.0.0", count: 21 },
  { version: "v4.1.0-beta.1", count: 4 },
  { version: "development", count: 8 },
  { version: "v4.0.1-beta.1", count: 1 },
  { version: "4.1.0", count: 1 },
  { version: "v4.1.0-custom-2", count: 3 },
  { version: "nightly", count: 2 },
  { version: "v5.0.0-alpha.1", count: 1 },
  { version: "v5.0.0-beta.3", count: 1 },
  { version: "v5.0.0", count: 43 },
  { version: "v5.0.1", count: 88 },
  { version: "main", count: 1 },
  { version: "v5.0.2", count: 77 },
  { version: "v5.0.3", count: 20 },
  { version: "v5.0.4", count: 410 },
  { version: "v5.0.5", count: 13 },
  { version: "v5.0.6-beta.1", count: 1 },
  { version: "v5.0.6", count: 284 },
];

// Clear existing data
db.exec("DELETE FROM instances");

// Insert data
const insert = db.prepare(
  `INSERT INTO instances (uuid, version, last_seen) VALUES (?, ?, ?)`,
);

let totalInstances = 0;
const now = Math.floor(Date.now() / 1000);

versions.forEach(({ version, count }) => {
  for (let i = 0; i < count; i++) {
    const uuid = randomUUID();
    const lastSeen = now - Math.floor(Math.random() * 86400 * 30); // Random time in last 30 days
    insert.run(uuid, version, lastSeen);
    totalInstances++;
  }
});

console.log(`✓ Seeded ${totalInstances} instances`);
console.log(`✓ Across ${versions.length} versions`);

db.close();
