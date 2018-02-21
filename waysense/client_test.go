//  Copyright © 2018 Pixel Brewery Co. All rights reserved.
package waysense

import (
	"fmt"
	"testing"
	"time"
)

var thingMetric = ThingMetric{}

const TestAddress = "http://localhost:8100/v1/waysense/write"

func assertNotPanics(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()
	f()
}

func TestClientSingle(t *testing.T) {
	testUrl := TestAddress
	client, err := NewTest(testUrl, "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.Location("testid", "GFJR"); err != nil {
		t.Fatal(err)
	}
}

func TestClientFlushBuffer(t *testing.T) {
	testUrl := TestAddress
	bufferLength := 9
	client, err := NewBuffered(testUrl, "", "", bufferLength, time.Duration(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	i := 0
	for i < bufferLength-1 {
		id := fmt.Sprintf("test-%d", i)
		if err := client.Location(id, "GFJR1"); err != nil {
			t.Fatal(err)
		}
		i += 1
	}

	if len(client.things) != (bufferLength - 1) {
		t.Errorf("Expected client to have buffered %d commands, but found %d\n", (bufferLength - 1), len(client.things))
	}

}
