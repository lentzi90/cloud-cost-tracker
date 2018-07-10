package cct_azure

//go:generate mockgen -destination=./mocks/client_mock.go -package=mocks -source=client.go

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Client interface is made to simplify testing
type Client interface {
	GetPeriodIterator(string) billing.PeriodsListResultIterator
	GetUsageIterator(billingPeriod, filter string) consumption.UsageDetailsListResultIterator
}

// RestClient is a simple implementation of Client
type RestClient struct {
	periodsClient billingClient
	usageClient   consumptionClient
}

type billingClient interface {
	ListComplete(ctx context.Context, filter string, skiptoken string, top *int32) (result billing.PeriodsListResultIterator, err error)
}

type consumptionClient interface {
	ListByBillingPeriodComplete(ctx context.Context, billingPeriodName string, expand string, filter string, apply string, skiptoken string, top *int32) (result consumption.UsageDetailsListResultIterator, err error)
}

// NewRestClient returns a RestClient for the given subscription ID.
func NewRestClient(subscriptionID string) Client {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	periodsClient := billing.NewPeriodsClient(subscriptionID)
	periodsClient.Authorizer = authorizer
	usageClient := consumption.NewUsageDetailsClient(subscriptionID)
	usageClient.Authorizer = authorizer

	return RestClient{periodsClient: periodsClient, usageClient: usageClient}
}

// GetPeriodIterator returns a PeriodsListResultIterator given a filter string
func (c RestClient) GetPeriodIterator(filter string) billing.PeriodsListResultIterator {
	var top int32 = 100
	result, err := c.periodsClient.ListComplete(context.Background(), filter, "", &top)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// GetUsageIterator returns a new UsageDetailsListResultIterator over a given billing period and filter
func (c RestClient) GetUsageIterator(billingPeriod, filter string) consumption.UsageDetailsListResultIterator {
	var top int32 = 100
	result, err := c.usageClient.ListByBillingPeriodComplete(context.Background(), billingPeriod, "", filter, "", "", &top)
	if err != nil {
		log.Fatal(err)
	}
	return result
}
