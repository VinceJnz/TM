package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/gateways/stripe"
	"api-server/v2/app/services/checkout"
	"context"
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	payment := flag.Int("payment", 0, "local provider-backed payment ID")
	refund := flag.String("refund", "", "provider refund reference")
	actor := flag.String("actor", "", "operator identity")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := appCore.InitDBContext(ctx, appCore.DatabaseSourceFromEnv())
	if err != nil {
		log.Fatal("database unavailable")
	}
	defer db.Close()
	key := os.Getenv("PAYMENT_KEY")
	if key == "" {
		log.Fatal("PAYMENT_KEY is required")
	}
	if err = checkout.ConfirmRefund(ctx, db, stripe.NewFromKey(key, "", false), *payment, *refund, *actor); err != nil {
		log.Fatal("refund verification failed; inspect pending record and provider status")
	}
	log.Print("provider refund status recorded; only succeeded refunds affect booking balance")
}
