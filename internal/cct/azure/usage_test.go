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

// TODO: Add test for multiple instances per provider. This is not working correctly now

var (
	subscriptionID = "abcdefgh-1234-1234-abcd-abcdefghijkl"
)

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSubscriptionsIter := NewMocksubscriptionIterator(mockCtrl)
	mockPeriodsIter := NewMockperiodsIterator(mockCtrl)
	mockUsageIter := NewMockusageIterator(mockCtrl)
	mockClient := NewMockClient(mockCtrl)
	ue := UsageExplorer{client: mockClient}

	// Test data
	id := "id"
	name := "name"
	period := billing.Period{ID: &id, Name: &name}
	resourceGroup := "elastisys-container-registry"
	provider := "Microsoft.ContainerRegistry/registries"
	instance := "elastisys"
	instanceID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/" + provider + "/" + instance
	cost := 10.50
	currency := "SEK"
	usageDate := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	usage := fakeUsageDetail(usageDate, cost, currency, instanceID)
	usageSlice := []consumption.UsageDetail{usage}
	labels := map[string]string{"cloud": "azure", "subscription": subscriptionID, "resource_group": resourceGroup, "service": provider, "instance": instance, "currency": currency}
	data := dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels}

	subscriptions := []subscription.Model{subscription.Model{SubscriptionID: &subscriptionID}}

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
		setupSubscriptionIterator(*mockSubscriptionsIter, subscriptions)

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get usage iterator", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, nil)
		mockClient.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(mockPeriodsIter, nil)
		mockClient.EXPECT().getUsageIterator(subscriptionID, name, gomock.Any()).Return(mockUsageIter, errors.New("error"))
		setupSubscriptionIterator(*mockSubscriptionsIter, subscriptions)

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get cloud cost", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)
		setupSubscriptionIterator(*mockSubscriptionsIter, subscriptions)
		setupUsageIterator(*mockUsageIter, usageSlice)
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter)

		expected := []dbclient.UsageData{data}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})

	t.Run("Get partial cloud cost", func(t *testing.T) {
		// Properties are missing
		partial := consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: nil}
		data := []consumption.UsageDetail{partial}

		mockPeriodsIter.EXPECT().Value().Return(period)
		setupSubscriptionIterator(*mockSubscriptionsIter, subscriptions)
		setupUsageIterator(*mockUsageIter, data)
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter)

		// We expect no data because of the missing properties
		expected := []dbclient.UsageData{}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})

	t.Run("Get multiple cloud cost", func(t *testing.T) {
		usageSlice := []consumption.UsageDetail{usage, usage, usage}

		mockPeriodsIter.EXPECT().Value().Return(period)
		setupSubscriptionIterator(*mockSubscriptionsIter, subscriptions)
		setupUsageIterator(*mockUsageIter, usageSlice)
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter)

		expected := []dbclient.UsageData{data, data, data}
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
	usageStart := date.Time{usageDate}
	usageEnd := date.Time{usageDate.AddDate(0, 0, 1)}
	usageProps := consumption.UsageDetailProperties{InstanceID: &instanceID, PretaxCost: &pretaxCost, Currency: &currency, UsageStart: &usageStart, UsageEnd: &usageEnd}
	return consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: &usageProps}
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
func setupClient(mock MockClient, subscriptionsIter subscriptionIterator, periodsIter periodsIterator, usageIter usageIterator) {
	mock.EXPECT().getSubscriptionIterator().Return(subscriptionsIter, nil)
	mock.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(periodsIter, nil)
	mock.EXPECT().getUsageIterator(subscriptionID, gomock.Any(), gomock.Any()).Return(usageIter, nil)
}

// Check that the actual data resambles the expected data
func checkCloudCost(t *testing.T, expected, actual []dbclient.UsageData) {
	if len(actual) != len(expected) {
		t.Errorf("UsageData slince lengths differ!")
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
