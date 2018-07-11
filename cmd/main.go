package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/lentzi90/cct-azure/internal/cct/cct_azure"
	"github.com/lentzi90/cct-azure/internal/cct/db_client"
)

var subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")

func main() {
	fmt.Println("Welcome to Cloud Cost Tracker V1.0.5")

	dbConfig := db_client.DBClientConfig{
		DBName:   "prometheus",
		Username: "prom",
		Password: "prom",
		Address:  "http://localhost:8086",
	}

	flag.Parse()
	if *subscriptionID == "" {
		log.Fatal("You must provide a subscription id by using the --subscription-id flag.")
	}

	log.Println("Initializing client...")
	client := cct_azure.NewRestClient(*subscriptionID)
	usageExplorer := cct_azure.NewUsageExplorer(client)

	db := db_client.NewDBClient(dbConfig)
	now := time.Date(2017, time.October, 5, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		fetchTime := now.AddDate(0, 0, -i)
		fmt.Println("Getting for period", fetchTime)
		test, err := usageExplorer.GetCloudCost(fetchTime)
		if err == nil {
			db.AddUsageData(test)
		} else {
			log.Println("Got error, skipping usage data:", err)
		}
	}

	log.Println("DONE!!!")
}
