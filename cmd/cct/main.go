package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/lentzi90/cloud-cost-tracker/internal/cct/azure"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
)

var (
	subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")
	dbName         = flag.String("db-name", "cloudCostTracker", "The name of the database to use.")
	dbUsername     = flag.String("db-username", "cctUser", "The username to the database.")
	dbPassword     = flag.String("db-password", "cctPassword", "The password to the database.")
	dbAddress      = flag.String("db-address", "http://localhost:8086", "The address to the database.")
)

func main() {
	fmt.Println("Welcome to Cloud Cost Tracker V1.0.5")

	flag.Parse()
	if *subscriptionID == "" {
		log.Fatal("You must provide a subscription id by using the --subscription-id flag.")
	}

	dbConfig := dbclient.DBClientConfig{
		DBName:   *dbName,
		Username: *dbUsername,
		Password: *dbPassword,
		Address:  *dbAddress,
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
			if err = db.AddUsageData(test); err != nil {
				log.Fatalf("DB Error: %v", err.Error())
			}

		} else {
			log.Println("Got error, skipping usage data:", err)
		}
	}

	log.Println("DONE!!!")
}
