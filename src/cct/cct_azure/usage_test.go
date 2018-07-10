package cct_azure

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
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

func TestGetPeriod(t *testing.T) {
	ue := setUp()
	date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	id := "id"
	name := "name"
	expected := billing.Period{ID: &id, Name: &name}
	actual := ue.getPeriodByDate(date)

	if actual != expected {
		t.Errorf("Expected and actual differ!")
	}
}
