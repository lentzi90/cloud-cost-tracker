package main

import (
	"testing"
)

func TestSplitString(t *testing.T) {
	instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.ContainerRegistry/registries/elastisys"
	expected := "Microsoft.ContainerRegistry"
	actual := getProvider(instanceID)

	if actual != expected {
		t.Errorf("Wanted: %s got: %s", expected, actual)
	}
}
