package azure

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
)

// A UsageExplorer can be used to investigate usage cost
type UsageExplorer struct {
	client Client
}

// NewUsageExplorer initializes a UsageExplorer
func NewUsageExplorer() UsageExplorer {
	return UsageExplorer{client: NewRestClient()}
}

// GetCloudCost fetches the cost for the specified date
func (e *UsageExplorer) GetCloudCost(date time.Time) ([]dbclient.UsageData, error) {
	var data []dbclient.UsageData
	subscriptions, err := e.getSubscriptions()
	if err != nil {
		return data, err
	}
	for _, sub := range subscriptions {
		log.Println("Trying to get cost for subscription", sub)
		subCost, err := e.getSubscriptionCost(sub, date)
		if err == nil {
			data = append(data, subCost...)
		} else {
			log.Println("Warning: Unable to get cost for subscription", sub, err)
			return data, err
		}
	}

	return data, nil
}

func (e *UsageExplorer) getPeriodByDate(subscriptionID string, date time.Time) (billing.Period, error) {
	dateStr := date.Format("2006-01-02")
	filter := "billingPeriodEndDate gt " + dateStr

	periods, err := e.client.getPeriodIterator(subscriptionID, filter)
	if err != nil {
		return billing.Period{}, err
	}
	// Periods are returned in reverse chronologic order, so we return the first one.
	// This will be the billing period including date
	return periods.Value(), nil
}

func (e *UsageExplorer) getUsageByDate(subscriptionID string, date time.Time) (usageIterator, error) {
	billingPeriod, err := e.getPeriodByDate(subscriptionID, date)
	if err != nil {
		return &consumption.UsageDetailsListResultIterator{}, err
	}
	billingPeriodName := *billingPeriod.Name
	filter := fmt.Sprintf("properties/usageStart eq '%s'", date.Format("2006-01-02"))
	log.Println("Trying to get usage for billing period", billingPeriodName)

	result, err := e.client.getUsageIterator(subscriptionID, billingPeriodName, filter)
	if err != nil {
		return &consumption.UsageDetailsListResultIterator{}, err
	}

	return result, nil
}

func (e *UsageExplorer) getSubscriptions() ([]string, error) {
	result := []string{}
	subIter, err := e.client.getSubscriptionIterator()
	if err != nil {
		return result, err
	}

	for subIter.NotDone() {
		sub := subIter.Value()
		subIter.Next()
		if sub.SubscriptionID == nil {
			continue
		}

		result = append(result, *sub.SubscriptionID)
	}

	return result, err
}

func (e *UsageExplorer) getSubscriptionCost(subscriptionID string, date time.Time) ([]dbclient.UsageData, error) {
	var data []dbclient.UsageData
	usageIter, err := e.getUsageByDate(subscriptionID, date)
	if err != nil {
		return data, err
	}

	for usageIter.NotDone() {
		usageDetails := usageIter.Value()
		usageIter.Next()
		// Check that fields actually exist!
		if !propertiesOK(usageDetails) {
			continue
		}

		instanceID := *usageDetails.InstanceID
		pretaxCost := *usageDetails.PretaxCost
		currency := *usageDetails.Currency
		usageStart := *usageDetails.UsageStart

		labels := getLabels(instanceID, currency)
		log.Println(pretaxCost, currency, usageStart.Format("2006-01-02 15:04"), labels)

		cost, _ := pretaxCost.Float64()

		data = append(data, dbclient.UsageData{Cost: cost, Date: date, Labels: labels})
	}

	return data, nil
}

func getLabels(instanceID, currency string) map[string]string {
	// The instance ID is a string like this:
	// /subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{subtype}/{resource-name}
	// See: https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
	// We extract the provider by splitting on /
	parts := strings.Split(instanceID, "/")
	labels := make(map[string]string)
	labels["cloud"] = "azure"
	labels["subscription"] = parts[2]
	labels["resource_group"] = parts[4]
	labels["service"] = strings.Join(parts[6:8], "/")
	labels["currency"] = currency
	labels["instance"] = parts[8]
	return labels
}

func propertiesOK(usageDetails consumption.UsageDetail) bool {
	if (usageDetails.UsageDetailProperties == nil) ||
		(usageDetails.UsageStart == nil) ||
		(usageDetails.PretaxCost == nil) ||
		(usageDetails.Currency == nil) ||
		(usageDetails.InstanceID == nil) {
		return false
	}
	return true
}
