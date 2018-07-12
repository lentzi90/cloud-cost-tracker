package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lentzi90/cloud-cost-tracker/internal/cct/aws"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/azure"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
)

var (
	cloud      = flag.String("cloud", "", "The cloud provider you want to update.")
	dbName     = flag.String("db-name", "cloudCostTracker", "The name of the database to use.")
	dbUsername = flag.String("db-username", "cctUser", "The username to the database.")
	dbPassword = flag.String("db-password", "cctPassword", "The password to the database.")
	dbAddress  = flag.String("db-address", "http://localhost:8086", "The address to the database.")
)

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	fmt.Println("Welcome to Cloud Cost Tracker V1.0.5")

	flag.Parse()

	dbConfig := dbclient.DBClientConfig{
		DBName:   *dbName,
		Username: *dbUsername,
		Password: *dbPassword,
		Address:  *dbAddress,
	}

	//var usageExplorer azure.UsageExplorer
	var usageExplorer dbclient.CloudCost
	if strings.EqualFold(*cloud, "azure") {
		log.Println("Initializing Azure client...")
		usageExplorer = initAzureClient()
	} else if strings.EqualFold(*cloud, "aws") {
		log.Println("Initializing AWS client...")
		usageExplorer = initAwsClient()
	} else {
		log.Fatalf("Cloud provider %v is not supported", *cloud)
	}

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

func initAzureClient() dbclient.CloudCost {
	client := azure.NewRestClient()
	return azure.NewUsageExplorer(client)
}

func initAwsClient() dbclient.CloudCost {
	return aws.NewClient{}
}
