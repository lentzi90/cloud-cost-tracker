package cct_azure

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lentzi90/cct-azure/src/cct/db_client"

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

// GetCloudCost fetches the cost for the specified date
func (e *UsageExplorer) GetCloudCost(date time.Time) []db_client.UsageData {
	usageIterator := e.getUsageByDate(date)
	providers := make(map[string]decimal.Decimal)
	var data []db_client.UsageData

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
		log.Printf("%s %s, %s - %s, %s\n", pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), usageEnd.Format("2006-01-02 15:04"), resourceProvider)

		labels := make(map[string]string)
		labels["provider"] = resourceProvider
		data = append(data, db_client.UsageData{Cost: pretaxCost, Currency: currency, Date: date, Labels: labels})

		usageIterator.Next()
	}

	fmt.Println(data)
	return data
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
	billingPeriod := e.getPeriodByDate(date)
	if billingPeriod.Name == nil {
		log.Fatal("No billing available")
	}
	billingPeriodName := *billingPeriod.Name
	filter := fmt.Sprintf("properties/usageStart eq '%s'", date.Format("2006-01-02"))
	log.Println("Trying to get usage for billing period", billingPeriodName)

	result := e.client.GetUsageIterator(billingPeriodName, filter)
	log.Println("Success!")

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
