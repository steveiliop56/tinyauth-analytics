package main

import (
	"database/sql"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var versions = []struct {
	name   string
	weight int
}{
	{"v5.1.0", 40},
	{"v5.0.2", 22},
	{"v5.0.1", 12},
	{"v5.0.0", 9},
	{"v4.3.1", 6},
	{"v4.2.0", 4},
	{"v4.0.0", 3},
	{"v3.6.2", 2},
	{"v3.5.0", 1},
	{"v2.9.1", 1},
}

var versionPool = func() []string {
	var pool []string
	for _, v := range versions {
		for range v.weight {
			pool = append(pool, v.name)
		}
	}
	return pool
}()

func main() {
	db, err := sql.Open("sqlite", "analytics.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS "instances" (
		"uuid" TEXT NOT NULL PRIMARY KEY,
		"version" TEXT NOT NULL,
		"last_seen" INTEGER NOT NULL
	);`); err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO instances (uuid, version, last_seen) VALUES (?, ?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	now := time.Now()

	for range 3000 {
		lastSeen := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

		if _, err := stmt.Exec(
			uuid.NewString(),
			versionPool[rand.Intn(len(versionPool))],
			lastSeen.Unix(),
		); err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	log.Println("seeded 3000 instances")
}
