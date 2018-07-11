package aws

import (
	"io/ioutil"
	"log"
	"testing"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

func TestS3ToUnix(t *testing.T) {
	form := "2006-01-02T15:04:05Z"
	timestamp := "2018-07-01T13:14:15Z"
	actual := s3ToUnix(timestamp, form)
	expected := 1530450855
	if actual != expected {
		t.Errorf("Expected value %d and actual value %d are not the same!", expected, actual)
	}
}
