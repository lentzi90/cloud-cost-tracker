package cct_azure

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cct-azure/internal/cct/cct_azure/mocks"
)

func setUp() UsageExplorer {
	subscriptionID := "abcdefgh-1234-1234-abcd-abcdefghijkl"
	client := NewRestClient(subscriptionID)
	usageExplorer := NewUsageExplorer(client)
	return usageExplorer
}

func TestSplitString(t *testing.T) {
	instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.ContainerRegistry/registries/elastisys"
	expected := "Microsoft.ContainerRegistry/registries"
	actual := getProvider(instanceID)

	if actual != expected {
		t.Errorf("Wanted: %s got: %s", expected, actual)
	}
}

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := mocks.NewMockClient(mockCtrl)
	ue := NewUsageExplorer(mockClient)

	// id := "id"
	// name := "name"
	// periodProps := billing.PeriodProperties{}
	// period := billing.Period{ID: &id, Name: &name}
	// periods := []billing.Period{period}
	// plr := billing.PeriodsListResult{Value: &periods}
	// i := 0
	// page := billing.PeriodsListResultPage{}
	periodIter := billing.PeriodsListResultIterator{}
	mockClient.EXPECT().GetPeriodIterator(gomock.Any()).Return(periodIter, errors.New("error"))

	date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	// var expected []UsageData = nil
	actual, err := ue.GetCloudCost(date)

	if err == nil {
		t.Errorf("Expected error but got none!")
	}

	if actual != nil {
		t.Errorf("Expected and actual differ!")
	}
}
