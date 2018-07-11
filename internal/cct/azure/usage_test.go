package azure

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"github.com/shopspring/decimal"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

func TestParseProvider(t *testing.T) {
	tests := map[string]string{
		"Microsoft.ContainerRegistry/registries": "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.ContainerRegistry/registries/elastisys",
		"Microsoft.Compute/disks":                "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.Compute/disks/elastisys",
	}

	for expected, id := range tests {
		actual := getProvider(id)
		if actual != expected {
			t.Errorf("Wanted: %s got: %s", expected, actual)
		}
	}

}

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPeriodsIter := NewMockperiodsIterator(mockCtrl)
	mockUsageIter := NewMockusageIterator(mockCtrl)
	mockClient := NewMockClient(mockCtrl)
	ue := NewUsageExplorer(mockClient)

	// Test data
	id := "id"
	name := "name"
	period := billing.Period{ID: &id, Name: &name}
	provider := "Microsoft.ContainerRegistry/registries"
	instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/" + provider + "/elastisys"
	cost := 10.50
	currency := "SEK"
	usageDate := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	usage := fakeUsageDetail(usageDate, cost, currency, instanceID)
	usageSlice := []consumption.UsageDetail{usage}

	t.Run("Fail to get period iterator", func(t *testing.T) {
		err0 := errors.New("error")
		mockClient.EXPECT().getPeriodIterator(gomock.Any()).Return(mockPeriodsIter, err0)
		date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)

		_, err := ue.GetCloudCost(date)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get usage iterator", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)

		mockClient.EXPECT().getPeriodIterator(gomock.Any()).Return(mockPeriodsIter, nil)
		mockClient.EXPECT().getUsageIterator(name, gomock.Any()).Return(mockUsageIter, errors.New("error"))

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get cloud cost", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)
		setupUsageIterator(*mockUsageIter, usageSlice)
		setupClient(*mockClient, mockPeriodsIter, mockUsageIter)

		labels := map[string]string{"cloud": "azure", "service": provider, "currency": "SEK"}
		data := dbclient.UsageData{Cost: cost, Currency: currency, Date: usageDate, Labels: labels}
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
		setupUsageIterator(*mockUsageIter, data)
		setupClient(*mockClient, mockPeriodsIter, mockUsageIter)

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
		setupUsageIterator(*mockUsageIter, usageSlice)
		setupClient(*mockClient, mockPeriodsIter, mockUsageIter)

		labels := map[string]string{"cloud": "azure", "service": provider, "currency": "SEK"}
		data := dbclient.UsageData{Cost: cost, Currency: currency, Date: usageDate, Labels: labels}
		expected := []dbclient.UsageData{data, data, data}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})
}

func fakeUsageDetail(usageDate time.Time, cost float64, currency string, instanceID string) consumption.UsageDetail {
	id := "id"
	name := "name"
	pretaxCost := decimal.NewFromFloat(10.50)
	usageStart := date.Time{usageDate}
	usageEnd := date.Time{usageDate.AddDate(0, 0, 1)}
	usageProps := consumption.UsageDetailProperties{InstanceID: &instanceID, PretaxCost: &pretaxCost, Currency: &currency, UsageStart: &usageStart, UsageEnd: &usageEnd}
	return consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: &usageProps}
}

// Add logic to let the mocker iterator iterate over the data
func setupUsageIterator(mock MockusageIterator, data []consumption.UsageDetail) {
	mock.EXPECT().Next().AnyTimes()
	// Allow mock to iterate over all the data
	for _, usage := range data {
		mock.EXPECT().NotDone().Return(true)
		mock.EXPECT().Value().Return(usage)
	}

	mock.EXPECT().NotDone().Return(false)
}

func setupClient(mock MockClient, periodsIter periodsIterator, usageIter usageIterator) {
	mock.EXPECT().getPeriodIterator(gomock.Any()).Return(periodsIter, nil)
	mock.EXPECT().getUsageIterator(gomock.Any(), gomock.Any()).Return(usageIter, nil)
}

func checkCloudCost(t *testing.T, expected, actual []dbclient.UsageData) {
	if len(actual) != len(expected) {
		t.Errorf("UsageData slince lengths differ!")
	}
	for i := range expected {
		if (actual[i].Cost != expected[i].Cost) ||
			(actual[i].Currency != expected[i].Currency) ||
			(actual[i].Date != expected[i].Date) {
			t.Errorf("UsageData differ. Expected: %f %s %s, Actual: %f %s %s",
				expected[i].Cost, expected[i].Currency, expected[i].Date.Format("2006-01-02"),
				actual[i].Cost, actual[i].Currency, actual[i].Date.Format("2006-01-02"))
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
