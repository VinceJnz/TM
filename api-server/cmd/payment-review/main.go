// payment-review is an explicit operator tool, never an application endpoint.
package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	session := flag.String("session", "", "session to resolve after an externally confirmed refund; omit to list open cases")
	actor := flag.String("actor", "", "operator identity")
	evidence := flag.String("evidence", "", "verified provider refund reference and case note")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := appCore.InitDBContext(ctx, appCore.DatabaseSourceFromEnv())
	if err != nil {
		log.Fatal("database unavailable")
	}
	defer db.Close()
	if *session != "" {
		if err := paymentLedger.ResolveReview(ctx, db, *session, *actor, *evidence); err != nil {
			log.Fatal(err)
		}
		return
	}
	rows, err := db.QueryxContext(ctx, `SELECT session_id,booking_id,amount_minor,currency,reason,first_seen,last_seen,failures FROM at_payment_reviews WHERE resolved_at IS NULL ORDER BY first_seen LIMIT 100`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		row := map[string]any{}
		if err := rows.MapScan(row); err != nil {
			log.Fatal(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(row); err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
