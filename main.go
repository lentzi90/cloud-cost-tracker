package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"

	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")

func main() {
	flag.Parse()
	if *subscriptionID == "" {
		log.Fatal("You must provide a subscription id by using the --subscription-id flag.")
	}
	iter, err := getUsageIterator(*subscriptionID)
	if err == nil {
		extractUsage(iter)
	}
}

func getUsageIterator(subscriptionID string) (consumption.UsageDetailsListResultIterator, error) {
	var result consumption.UsageDetailsListResultIterator

	usageClient := consumption.NewUsageDetailsClient(subscriptionID)
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		usageClient.Authorizer = authorizer
	} else {
		log.Fatal(err)
	}

	expand := ""
	filter := ""
	skiptoken := ""
	var top *int32
	apply := ""
	log.Println("Trying to get list...")
	result, err = usageClient.ListComplete(context.Background(), expand, filter, skiptoken, top, apply)

	if err == nil {
		log.Println("Got a list!")
		return result, nil
	}
	log.Fatal(err)
	return result, err
}

func extractUsage(usageIterator consumption.UsageDetailsListResultIterator) {
	fmt.Println("Pretax cost Currency, Usage start - Usage end, Instance ID")
	fmt.Println("----------------------------------------------------------")
	// For all values, print some information
	for usageIterator.NotDone() {
		usageDetails := usageIterator.Value()
		instanceID := *usageDetails.InstanceID
		pretaxCost := *usageDetails.PretaxCost
		currency := *usageDetails.Currency
		usageStart := *usageDetails.UsageStart
		usageEnd := *usageDetails.UsageEnd
		// isEstimated := *usageDetails.IsEstimated
		fmt.Printf("%s %s, %s - %s, %s\n", pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), usageEnd.Format("2006-01-02 15:04"), instanceID)
		usageIterator.Next()
	}
}
