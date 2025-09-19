CREATE TABLE IF NOT EXISTS "instances" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "uuid" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "last_seen" INTEGER NOT NULL
);
