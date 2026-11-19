package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/migrations"
	"context"
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	adopt := flag.Bool("adopt-existing", false, "record the baseline for an audited compatible existing schema; never recreate its tables")
	role := flag.String("grant-role", "", "explicitly grant runtime access to an existing non-administrative API role")
	flag.Parse()
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("invalid optional .env file")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := appCore.InitDBContext(ctx, appCore.DatabaseSourceFromEnv())
	if err != nil {
		log.Fatal("database unavailable")
	}
	defer db.Close()
	if err := migrations.Apply(ctx, db, *adopt); err != nil {
		log.Fatal(err)
	}
	if *role != "" {
		if err := migrations.GrantRuntimeAccess(ctx, db, *role); err != nil {
			log.Fatal(err)
		}
	}
	log.Print("database migrations are current")
}
