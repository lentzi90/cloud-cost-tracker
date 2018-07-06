package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// A UsageExplorer can be used to investigate usage cost
type UsageExplorer struct {
	authorizer     autorest.Authorizer
	subscriptionID string
}

// NewUsageExplorer initializes a UsageExplorer
func NewUsageExplorer(subscriptionID string) UsageExplorer {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	return UsageExplorer{authorizer: authorizer, subscriptionID: subscriptionID}
}

// PrintCurrentUsage prints the usage for the current billing period
func (e *UsageExplorer) PrintCurrentUsage() {
	periods := e.getPeriodsIterator()
	usageIterator := e.getUsageIterator(*periods.Value().Name)
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

func (e *UsageExplorer) getPeriodsIterator() billing.PeriodsListResultIterator {
	periodsClient := billing.NewPeriodsClient(e.subscriptionID)
	periodsClient.Authorizer = e.authorizer

	// filter := "billingPeriodEndDate lt 2018-05-30"
	filter := ""

	periods, err := periodsClient.ListComplete(context.Background(), filter, "", nil)
	if err != nil {
		log.Fatal(err)
	}

	return periods
}

func (e *UsageExplorer) getUsageIterator(billingPeriod string) consumption.UsageDetailsListResultIterator {
	usageClient := consumption.NewUsageDetailsClient(e.subscriptionID)
	usageClient.Authorizer = e.authorizer

	expand := ""
	// filter := "properties/usageEnd le '2018-07-02' AND properties/usageEnd ge '2018-06-30'"
	filter := ""
	skiptoken := ""
	var top int32 = 100
	apply := ""
	log.Println("Trying to get list from billing period", billingPeriod)
	result, err := usageClient.ListByBillingPeriodComplete(context.Background(), billingPeriod, expand, filter, apply, skiptoken, &top)

	if err != nil {
		log.Fatal(err)
	}

	return result
}
