package main

import (
	"flag"
	"fmt"
	"log"
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

// Struct to be able to use the interface from dbclient with Azure
type azureCloudCost struct {
	*azure.UsageExplorer
}

// Struct to be able to use the interface from dbclient with AWS
type awsCloudCost struct {
	*aws.Client
}

func main() {
	fmt.Println("Welcome to Cloud Cost Tracker V1.0.5")

	flag.Parse()

	db := dbclient.NewDBClient(dbclient.Config{
		DBName:   *dbName,
		Username: *dbUsername,
		Password: *dbPassword,
		Address:  *dbAddress,
	})

	cloudCost := getCloudCostClient()
	fetchDataForDate(db, cloudCost, time.Now())
}

// Fetches data from a CloudCostClient for an interval and adding it to the database
func fetchDataForInterval(db dbclient.DBClient, cloudCost dbclient.CloudCostClient, startDate time.Time, stopDate time.Time) {
	if startDate.After(stopDate) {
		log.Fatalf("Fetch for interval: start date can't be after stop date")
	}

	currentTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	// Make sure stop date will be included
	stopDate = time.Date(stopDate.Year(), stopDate.Month(), stopDate.Day(), 0, 0, 0, 1, stopDate.Location())

	for currentTime.Before(stopDate) {
		fetchDataForDate(db, cloudCost, currentTime)
		currentTime = currentTime.AddDate(0, 0, 1)
	}
}

// Fetches data from a CloudCostClient and adding it to the database
func fetchDataForDate(db dbclient.DBClient, cloudCost dbclient.CloudCostClient, time time.Time) {
	fmt.Println("Getting for period", time)
	test, err := cloudCost.GetCloudCost(time)
	if err == nil {
		if err = db.AddUsageData(test); err != nil {
			log.Fatalf("DB Error: %v", err.Error())
		}

	} else {
		log.Println("Got error, skipping usage data:", err)
	}
}

// Retrieves the correct CloudCostClient depending on the cloud flag
func getCloudCostClient() dbclient.CloudCostClient {
	var cloudCost dbclient.CloudCostClient

	if strings.EqualFold(*cloud, "azure") {
		log.Println("Initializing Azure client...")
		azureClient := initAzureClient()
		cloudCost = &azureCloudCost{UsageExplorer: &azureClient}
	} else if strings.EqualFold(*cloud, "aws") {
		log.Println("Initializing AWS client...")
		awsClient := initAwsClient()
		cloudCost = &awsCloudCost{Client: &awsClient}
	} else {
		log.Fatalf("Cloud provider \"%v\" is not supported", *cloud)
	}
	return cloudCost
}

// Initializes the Azure client
func initAzureClient() azure.UsageExplorer {
	client := azure.NewRestClient()
	explorer := azure.NewUsageExplorer(client)
	return explorer
}

// Initializes the AWS client
func initAwsClient() aws.Client {
	return aws.NewClient("elastisys-billing-data")
}
