package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/lentzi90/cloud-cost-tracker/internal/cct/azure"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
)

var subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")

func main() {
	fmt.Println("The coolest DB Client V1.0.5")

	dbConfig := dbclient.DBClientConfig{
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
	client := azure.NewRestClient(*subscriptionID)
	usageExplorer := azure.NewUsageExplorer(client)

	db := dbclient.NewDBClient(dbConfig)
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
