package azure

//go:generate mockgen -destination=./client_mock.go -package=azure -source=client.go

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Client interface is made to simplify testing
type Client interface {
	getPeriodIterator(string) (periodsIterator, error)
	getUsageIterator(billingPeriod, filter string) (usageIterator, error)
	getSubscriptionIterator() (subscriptionIterator, error)
}

// RestClient is a simple implementation of Client
type RestClient struct {
	periodsClient billingClient
	usageClient   consumptionClient
	subsClient    subscriptionClient
}

type billingClient interface {
	ListComplete(ctx context.Context, filter string, skiptoken string, top *int32) (result billing.PeriodsListResultIterator, err error)
}

type consumptionClient interface {
	ListByBillingPeriodComplete(ctx context.Context, billingPeriodName string, expand string, filter string, apply string, skiptoken string, top *int32) (result consumption.UsageDetailsListResultIterator, err error)
}

type subscriptionClient interface {
	ListComplete(ctx context.Context) (result subscription.ListResultIterator, err error)
}

type usageIterator interface {
	Next() error
	NotDone() bool
	Value() consumption.UsageDetail
}

type periodsIterator interface {
	Next() error
	NotDone() bool
	Value() billing.Period
}

type subscriptionIterator interface {
	Next() error
	NotDone() bool
	Value() subscription.Model
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

func (c RestClient) getPeriodIterator(filter string) (periodsIterator, error) {
	var top int32 = 100
	result, err := c.periodsClient.ListComplete(context.Background(), filter, "", &top)
	return &result, err
}

func (c RestClient) getUsageIterator(billingPeriod, filter string) (usageIterator, error) {
	var top int32 = 100
	result, err := c.usageClient.ListByBillingPeriodComplete(context.Background(), billingPeriod, "", filter, "", "", &top)
	return &result, err
}

func (c RestClient) getSubscriptionIterator() (subscriptionIterator, error) {
	result, err := c.subsClient.ListComplete(context.Background())
	return &result, err
}
