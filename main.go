package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var (
	subscriptionID = flag.String("subscription-id", "", "The ID of the subscription.")
	addr           = flag.String("listen-address", ":8080", "The address to listen on for (scraping) HTTP requests.")
)

var authorizer autorest.Authorizer

func main() {
	flag.Parse()
	if *subscriptionID == "" {
		log.Fatal("You must provide a subscription id by using the --subscription-id flag.")
	}

	// Initialize authorizer
	var err error
	authorizer, err = auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	periodsIterator, err := getPeriodsIterator(*subscriptionID)
	if err != nil {
		log.Fatal(err)
	}
	extractUsage(periodsIterator)
}

func getUsageIterator(billingPeriod string) consumption.UsageDetailsListResultIterator {
	usageClient := consumption.NewUsageDetailsClient(*subscriptionID)
	usageClient.Authorizer = authorizer

	expand := ""
	// filter := "properties/usageEnd le '2018-07-02' AND properties/usageEnd ge '2018-06-30'"
	filter := ""
	skiptoken := ""
	var top int32 = 100
	apply := ""
	log.Println("Trying to get list from billing period", billingPeriod)
	result, err := usageClient.ListByBillingPeriodComplete(context.Background(), billingPeriod, expand, filter, apply, skiptoken, &top)

	if err == nil {
		log.Println("Got a list!")
		return result
	}
	log.Fatal(err)
	return result
}

func extractUsage(periodsIterator billing.PeriodsListResultIterator) {
	for periodsIterator.NotDone() {
		billingPeriod := *periodsIterator.Value().Name
		usageIterator := getUsageIterator(billingPeriod)
		printUsage(usageIterator)
		periodsIterator.Next()
	}
}

func printUsage(usageIterator consumption.UsageDetailsListResultIterator) {
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

func getPeriodsIterator(subscriptionID string) (billing.PeriodsListResultIterator, error) {
	periodsClient := billing.NewPeriodsClient(subscriptionID)
	periodsClient.Authorizer = authorizer

	// filter := "billingPeriodEndDate lt 2018-05-30"
	filter := ""

	return periodsClient.ListComplete(context.Background(), filter, "", nil)
}
