//  Copyright © 2018 Pixel Brewery Co. All rights reserved.

package waysense

import (
	"encoding/json"
	"sync"
	"time"
)

// safe to use client from multiple goroutines
type Client struct {
	writer        metricWriter
	bufferLength  int
	flushInterval time.Duration
	things        []ThingMetric
	stop          chan struct{}
	sync.Mutex
}

// use short key names to reduce payload size
type ThingMetric struct {
	ThingId    string                 `json:"id"`
	ThingValue map[string]interface{} `json:"v"`
	ThingTag   map[string]string      `json:"tag"`
	Time       int64                  `json:"t"`
}

type metricWriter interface {
	Write(data []byte) (n int, err error)
	SetWriteTimeout(time.Duration) error
	Close() error
}

/*
Stat suffixes
*/
const (
	HttpEndpoint     = "https://api-prod.pixelbrewery.co/v1/waysense/write"
	ThingTypeGeohash = "geo"
	ThingTypeLat     = "lat"
	ThingTypeLon     = "lon"
)

func newClient(addr, apiKey, apiSecret string) (*Client, error) {
	w, err := newHttpWriter(addr, apiKey, apiSecret, "20s", true)
	if err != nil {
		return nil, err
	}
	client := &Client{writer: w}
	return client, err
}

func New(apiKey, apiSecret string) (*Client, error) {
	return NewBuffered(HttpEndpoint, apiKey, apiSecret, 10, time.Duration(time.Second*30))
}

func NewWithEndpoint(addr, apiKey, apiSecret string) (*Client, error) {
	return NewBuffered(addr, apiKey, apiSecret, 10, time.Duration(time.Second*30))
}

func NewTest(addr, apiKey, apiSecret string) (*Client, error) {
	return NewBuffered(addr, apiKey, apiSecret, 0, time.Duration(time.Second*2))
}

func NewBuffered(addr, apiKey, apiSecret string, buflen int, flushSec time.Duration) (*Client, error) {
	client, err := newClient(addr, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	client.bufferLength = buflen
	client.flushInterval = flushSec
	client.stop = make(chan struct{}, 1)
	go client.watch()

	return client, nil
}

func (c *Client) watch() {
	ticker := time.NewTicker(c.flushInterval)

	for {
		select {
		case <-ticker.C:
			c.Lock()
			if len(c.things) > 0 {
				// FIXME: eating error here
				c.flushBuffer()
			}
			c.Unlock()
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
}

func (c *Client) Flush() error {
	if c == nil {
		return nil
	}
	c.Lock()
	defer c.Unlock()

	return c.flushBuffer()
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	select {
	case c.stop <- struct{}{}:
	default:
	}

	// if this client is buffered, flush before closing the writer
	if c.bufferLength > 0 {
		if err := c.Flush(); err != nil {
			return err
		}
	}

	return nil
}

// Gauge measures the value of a metric at a particular time
// ex> 	thingValue := map[string]interface{}{"waysense.memory": 10.0}
// ex> 	thingTag := map[string]string{"company": "waysense"}
func (c *Client) SendGuage(thingId string, thingValue map[string]interface{}, tag map[string]string) error {
	if tag == nil {
		tag = make(map[string]string)
	}

	tm := &ThingMetric{
		ThingId:    thingId,
		ThingValue: thingValue,
		ThingTag:   tag,
	}

	return c.sendThing(tm)
}

func (c *Client) SendGeoHash(thingId string, geoHash string, tag map[string]string) error {
	thingValue := make(map[string]interface{})
	thingValue[ThingTypeGeohash] = geoHash

	if tag == nil {
		tag = make(map[string]string)
	}

	tm := &ThingMetric{
		ThingId:    thingId,
		ThingValue: thingValue,
		ThingTag:   tag,
	}

	return c.sendThing(tm)
}

func (c *Client) SendLocation(thingId string, lat, lon float64, tag map[string]string) error {
	thingValue := make(map[string]interface{})
	thingValue[ThingTypeLat] = lat
	thingValue[ThingTypeLon] = lon

	if tag == nil {
		tag = make(map[string]string)
	}

	tm := &ThingMetric{
		ThingId:    thingId,
		ThingValue: thingValue,
		ThingTag:   tag,
	}

	return c.sendThing(tm)
}

func (c *Client) sendThing(thingMetric *ThingMetric) error {
	// client is buffered, just append for now
	if c.bufferLength > 0 {
		return c.appendThing(thingMetric)
	}

	// else, send it now
	ts := []ThingMetric{*thingMetric}
	d, err := json.Marshal(ts)
	if err != nil {
		return nil
	}
	_, err = c.writer.Write(d)

	return err
}

func (c *Client) appendThing(thingMetric *ThingMetric) error {
	c.Lock()
	defer c.Unlock()

	c.things = append(c.things, *thingMetric)
	if len(c.things) == c.bufferLength {
		c.flushBuffer()
	}

	return nil
}

// should always send this on mutex lock
func (c *Client) flushBuffer() error {
	var err error

	// should only flush when there are things
	if len(c.things) > 0 {
		var thingsData []byte

		thingsData, err = json.Marshal(c.things)
		if err != nil {
			return nil
		}

		_, err = c.writer.Write(thingsData)
		c.things = c.things[:0]
	}

	return err
}
