// operations-check reports aggregate queue health without changing application data.
package main

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/operations"
	store "api-server/v2/modelMethods/operations"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	review := flag.Duration("max-review-age", 5*time.Minute, "maximum unresolved paid-review age")
	refund := flag.Duration("max-refund-age", 24*time.Hour, "maximum pending provider-refund age")
	checkout := flag.Duration("max-checkout-overdue", 15*time.Minute, "maximum checkout age after expiry grace")
	flag.Parse()
	if *review <= 0 || *refund <= 0 || *checkout <= 0 {
		log.Fatal("positive thresholds are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := appCore.InitDBContext(ctx, appCore.DatabaseSourceFromEnv())
	if err != nil {
		log.Fatal("operations database unavailable")
	}
	defer db.Close()
	snapshot, err := (store.Store{DB: db}).Snapshot(ctx)
	if err != nil {
		log.Fatal("operations snapshot unavailable")
	}
	alerts := service.Alerts(snapshot, *review, *refund, *checkout)
	if err = json.NewEncoder(os.Stdout).Encode(struct {
		store.Snapshot
		Alerts []string `json:"alerts"`
	}{snapshot, alerts}); err != nil {
		log.Fatal("could not encode operations snapshot")
	}
	if len(alerts) > 0 {
		os.Exit(1)
	}
}
