package azure

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"

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
func (e *UsageExplorer) GetCloudCost(date time.Time) ([]dbclient.UsageData, error) {
	var data []dbclient.UsageData
	usageIter, err := e.getUsageByDate(date)
	if err != nil {
		return data, err
	}
	providers := make(map[string]decimal.Decimal)

	for usageIter.NotDone() {
		usageDetails := usageIter.Value()
		// Check that these fields actually exist!
		if (usageDetails.UsageDetailProperties == nil) ||
			(usageDetails.UsageStart == nil) ||
			(usageDetails.PretaxCost == nil) ||
			(usageDetails.Currency == nil) ||
			(usageDetails.InstanceID == nil) {
			continue
		}
		instanceID := *usageDetails.InstanceID
		pretaxCost := *usageDetails.PretaxCost
		currency := *usageDetails.Currency
		usageStart := *usageDetails.UsageStart
		// isEstimated := *usageDetails.IsEstimated

		resourceProvider := getProvider(instanceID)
		providers[resourceProvider] = decimal.Sum(providers[resourceProvider], pretaxCost)
		log.Printf("%s %s, %s, %s\n", pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), resourceProvider)

		labels := make(map[string]string)
		labels["provider"] = resourceProvider
		cost, _ := pretaxCost.Float64()
		data = append(data, dbclient.UsageData{Cost: cost, Currency: currency, Date: date, Labels: labels})

		usageIter.Next()
	}

	return data, nil
}

func (e *UsageExplorer) getPeriodByDate(date time.Time) (billing.Period, error) {
	dateStr := date.Format("2006-01-02")
	filter := "billingPeriodEndDate gt " + dateStr

	periods, err := e.client.getPeriodIterator(filter)
	if err != nil {
		return billing.Period{}, err
	}
	// Periods are returned in reverse chronologic order, so we return the first one.
	// This will be the billing period including date
	return periods.Value(), nil
}

func (e *UsageExplorer) getUsageByDate(date time.Time) (usageIterator, error) {
	billingPeriod, err := e.getPeriodByDate(date)
	if err != nil {
		return &consumption.UsageDetailsListResultIterator{}, err
	}
	billingPeriodName := *billingPeriod.Name
	filter := fmt.Sprintf("properties/usageStart eq '%s'", date.Format("2006-01-02"))
	log.Println("Trying to get usage for billing period", billingPeriodName)

	result, err := e.client.getUsageIterator(billingPeriodName, filter)
	if err != nil {
		return &consumption.UsageDetailsListResultIterator{}, err
	}
	log.Println("Success!")

	return result, nil
}

func getProvider(instanceID string) string {
	// The instance ID is a string like this:
	// /subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{subtype}/{resource-name}
	// See: https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
	// We extract the provider by splitting on /
	parts := strings.Split(instanceID, "/")
	return strings.Join(parts[6:8], "/")
}