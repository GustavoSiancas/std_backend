package main

import (
	"flag"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"std/internal/app"
	"std/internal/model"
)

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply pending SQL migrations and exit")
	flag.Parse()
	dsn := env("DATABASE_URL", "postgres://postgres:12345678@localhost:5432/std?sslmode=disable")
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if e != nil {
		log.Fatal(e)
	}
	if e = model.Migrate(db); e != nil {
		log.Fatal(e)
	}
	if *migrateOnly {
		log.Print("migrations applied")
		return
	}
	a := &app.App{DB: db}
	log.Fatal(a.Router().Run(":" + env("APP_PORT", "8080")))
}
