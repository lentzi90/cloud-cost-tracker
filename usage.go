package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/shopspring/decimal"
)

// A UsageExplorer can be used to investigate usage cost
type UsageExplorer struct {
	client Client
}

// NewUsageExplorer initializes a UsageExplorer
func NewUsageExplorer(client Client) UsageExplorer {
	return UsageExplorer{client: client}
}

// PrintCurrentUsage prints the usage for the current billing period
func (e *UsageExplorer) PrintCurrentUsage() {
	periods := e.client.GetPeriodIterator("")
	usageIterator := e.client.GetUsageIterator(*periods.Value().Name, "")
	fmt.Println("Pretax cost Currency, Usage start - Usage end, Provider")
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

		resourceProvider := getProvider(instanceID)
		fmt.Printf("%s %s, %s - %s, %s\n", pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), usageEnd.Format("2006-01-02 15:04"), resourceProvider)
		usageIterator.Next()
	}
}

func (e *UsageExplorer) getPeriodByDate(date time.Time) billing.Period {
	dateStr := date.Format("2006-01-02")
	filter := "billingPeriodEndDate gt " + dateStr

	periods := e.client.GetPeriodIterator(filter)
	// Periods are returned in reverse chronologic order, so we return the first one.
	// This will be the billing period including date
	return periods.Value()
}

func (e *UsageExplorer) getUsageByDate(date time.Time) consumption.UsageDetailsListResultIterator {
	billingPeriod := *e.getPeriodByDate(date).Name
	filter := fmt.Sprintf("properties/usageStart eq '%s'", date.Format("2006-01-02"))
	log.Println("Trying to get list from billing period", billingPeriod)

	result := e.client.GetUsageIterator(billingPeriod, filter)

	return result
}

func getProvider(instanceID string) string {
	// The instance ID is a string like this:
	// /subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{subtype}/{resource-name}
	// See: https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
	// We extract the provider by splitting on /
	parts := strings.Split(instanceID, "/")
	return strings.Join(parts[6:8], "/")
}

func (e *UsageExplorer) getUsageDetails(date time.Time) {
	usageIterator := e.getUsageByDate(date)
	providers := make(map[string]decimal.Decimal)

	for usageIterator.NotDone() {
		usageDetails := usageIterator.Value()
		instanceID := *usageDetails.InstanceID
		pretaxCost := *usageDetails.PretaxCost
		currency := *usageDetails.Currency
		usageStart := *usageDetails.UsageStart
		usageEnd := *usageDetails.UsageEnd
		// isEstimated := *usageDetails.IsEstimated

		resourceProvider := getProvider(instanceID)
		providers[resourceProvider] = decimal.Sum(providers[resourceProvider], pretaxCost)
		fmt.Printf("%s %s, %s - %s, %s\n", pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), usageEnd.Format("2006-01-02 15:04"), resourceProvider)

		usageIterator.Next()
	}

	fmt.Println(providers)
}
