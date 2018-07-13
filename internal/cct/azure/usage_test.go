package azure

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"github.com/shopspring/decimal"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

// TODO: usage details should be connected to subscriptions to make it possible to properly test multiple subscriptions
type inputData struct {
	subscriptions  []subscription.Model
	billingPeriods []billing.Period
	usageDetails   []consumption.UsageDetail
}

// Base data for building more complex data types
var (
	subscriptionID  = "abcdefgh-1234-1234-abcd-abcdefghijkl"
	subscriptionID2 = "abcdefgh-1234-4321-abcd-abcdefghijkl"
	periodID        = "id"
	periodName      = "name"
	period          = billing.Period{ID: &periodID, Name: &periodName}
	resourceGroup   = "group-name"
	provider        = "Microsoft.ContainerRegistry/registries"
	provider2       = "Microsoft.Compute/disks"
	instance        = "instance1"
	instance2       = "instance2"
	instanceID      = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/" + provider + "/" + instance
	instanceID2     = "/subscriptions/" + subscriptionID2 + "/resourceGroups/" + resourceGroup + "/providers/" + provider2 + "/" + instance2
	cost            = 10.50
	currency        = "SEK"
	currency2       = "USD"
	usageDate       = time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	labels          = map[string]string{"cloud": "azure", "subscription": subscriptionID, "resource_group": resourceGroup, "service": provider, "instance": instance, "currency": currency}
	labels2         = map[string]string{"cloud": "azure", "subscription": subscriptionID, "resource_group": resourceGroup, "service": provider, "instance": instance2, "currency": currency2}
	usageData       = dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels}
	usageData2      = dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels2}
)

// Faked data from API
var (
	singleSubscription = []subscription.Model{subscription.Model{SubscriptionID: &subscriptionID}}
	multiSubscription  = []subscription.Model{subscription.Model{SubscriptionID: &subscriptionID}, subscription.Model{SubscriptionID: &subscriptionID2}}
	singlePeriod       = []billing.Period{billing.Period{ID: &periodID, Name: &periodName}}
	multiPeriod        = []billing.Period{billing.Period{ID: &periodID, Name: &periodName}}
	usageDetail        = fakeUsageDetail(usageDate, cost, currency, instanceID)
	usageDetail2       = fakeUsageDetail(usageDate, cost, currency2, instanceID2)
	usageSlice         = []consumption.UsageDetail{usageDetail}
	usageSlice2        = []consumption.UsageDetail{usageDetail, usageDetail2}
)

// Input data
var (
	input  = inputData{subscriptions: singleSubscription, billingPeriods: singlePeriod, usageDetails: usageSlice}
	input2 = inputData{subscriptions: multiSubscription, billingPeriods: multiPeriod, usageDetails: usageSlice2}
)

// Output usageData
var (
	cloudCost  = []dbclient.UsageData{usageData}
	cloudCost2 = []dbclient.UsageData{usageData, usageData2}
)

// var testPairs = map[inputData][]dbclient.UsageData{}

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSubscriptionsIter := NewMocksubscriptionIterator(mockCtrl)
	mockPeriodsIter := NewMockperiodsIterator(mockCtrl)
	mockUsageIter := NewMockusageIterator(mockCtrl)
	mockClient := NewMockClient(mockCtrl)
	ue := UsageExplorer{client: mockClient}

	t.Run("Fail to get subscriptions iterator", func(t *testing.T) {
		err0 := errors.New("error")
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, err0)
		date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)

		_, err := ue.GetCloudCost(date)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get period iterator", func(t *testing.T) {
		err0 := errors.New("error")
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, nil)
		mockClient.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(mockPeriodsIter, err0)
		setupSubscriptionIterator(*mockSubscriptionsIter, singleSubscription)

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get usage iterator", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, nil)
		mockClient.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(mockPeriodsIter, nil)
		mockClient.EXPECT().getUsageIterator(subscriptionID, periodName, gomock.Any()).Return(mockUsageIter, errors.New("error"))
		setupSubscriptionIterator(*mockSubscriptionsIter, singleSubscription)

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get cloud cost", func(t *testing.T) {
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter, input.subscriptions)
		setupIterators(*mockClient, *mockSubscriptionsIter, *mockPeriodsIter, *mockUsageIter, input)

		expected := cloudCost
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})

	t.Run("Get partial cloud cost", func(t *testing.T) {
		// Properties are missing
		partial := consumption.UsageDetail{ID: &periodID, Name: &periodName, UsageDetailProperties: nil}
		data := []consumption.UsageDetail{partial}

		mockPeriodsIter.EXPECT().Value().Return(period)
		setupSubscriptionIterator(*mockSubscriptionsIter, singleSubscription)
		setupUsageIterator(*mockUsageIter, data)
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter, singleSubscription)

		// We expect no data because of the missing properties
		expected := []dbclient.UsageData{}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})

	t.Run("Get multiple cloud cost", func(t *testing.T) {
		// TODO: This test should use input2 and cloudCost2
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter, input2.subscriptions)
		setupIterators(*mockClient, *mockSubscriptionsIter, *mockPeriodsIter, *mockUsageIter, input)

		expected := cloudCost
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})
}

// Create a UsageDetail object from the provided input
func fakeUsageDetail(usageDate time.Time, cost float64, currency string, instanceID string) consumption.UsageDetail {
	id := "id"
	name := "name"
	pretaxCost := decimal.NewFromFloat(10.50)
	usageStart := date.Time{Time: usageDate}
	usageEnd := date.Time{Time: usageDate.AddDate(0, 0, 1)}
	usageProps := consumption.UsageDetailProperties{InstanceID: &instanceID, PretaxCost: &pretaxCost, Currency: &currency, UsageStart: &usageStart, UsageEnd: &usageEnd}
	return consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: &usageProps}
}

func setupIterators(client MockClient, subsIter MocksubscriptionIterator, periodsIter MockperiodsIterator, usageIter MockusageIterator, input inputData) {
	setupSubscriptionIterator(subsIter, input.subscriptions)
	setupPeriodsIterator(periodsIter, input.billingPeriods)
	setupUsageIterator(usageIter, input.usageDetails)
}

func setupPeriodsIterator(mock MockperiodsIterator, data []billing.Period) {
	// Filters are used to select the correct period,
	// so no iterating is needed as of now
	mock.EXPECT().Value().Return(data[0])
}

// Add logic to let the mocked iterator iterate over the data
func setupUsageIterator(mock MockusageIterator, data []consumption.UsageDetail) {
	mock.EXPECT().Next().AnyTimes()
	// Allow mock to iterate over all the data
	for _, usage := range data {
		mock.EXPECT().NotDone().Return(true)
		mock.EXPECT().Value().Return(usage)
	}

	mock.EXPECT().NotDone().Return(false)
}

// Add logic to let the mocked iterator iterate over the data
func setupSubscriptionIterator(mock MocksubscriptionIterator, data []subscription.Model) {
	mock.EXPECT().Next().AnyTimes()
	// Allow mock to iterate over all the data
	for _, item := range data {
		mock.EXPECT().NotDone().Return(true)
		mock.EXPECT().Value().Return(item)
	}

	mock.EXPECT().NotDone().Return(false)
}

// Make the mocked client return desired iterators
func setupClient(mock MockClient, subscriptionsIter subscriptionIterator, periodsIter periodsIterator, usageIter usageIterator, subscriptions []subscription.Model) {
	mock.EXPECT().getSubscriptionIterator().Return(subscriptionsIter, nil)
	// TODO: fix subscription IDs.
	mock.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(periodsIter, nil)
	mock.EXPECT().getUsageIterator(subscriptionID, gomock.Any(), gomock.Any()).Return(usageIter, nil)
}

// Check that the actual data resambles the expected data
func checkCloudCost(t *testing.T, expected, actual []dbclient.UsageData) {
	if len(actual) != len(expected) {
		t.Log("UsageData slince lengths differ!")
		t.FailNow()
	}
	for i := range expected {
		if (actual[i].Cost != expected[i].Cost) ||
			(actual[i].Date != expected[i].Date) {
			t.Errorf("UsageData differ. Expected: %f %s, Actual: %f %s",
				expected[i].Cost, expected[i].Date.Format("2006-01-02"),
				actual[i].Cost, actual[i].Date.Format("2006-01-02"))
		}

		if len(expected[i].Labels) != len(actual[i].Labels) {
			t.Errorf("Expected %d labels, actual %d", len(expected[i].Labels), len(actual[i].Labels))
		}

		for k, v := range expected[i].Labels {
			if v != actual[i].Labels[k] {
				t.Errorf("Expected: %s=%s, actual: %s=%s", k, v, k, actual[i].Labels[k])
			}
		}
	}
}
