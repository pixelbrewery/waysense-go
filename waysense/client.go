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

type ThingMetric struct {
	ThingId          string
	ThingDescription string
	ThingValue       map[string]interface{}
	Time             int64
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
	GuaugeThingType   = "g"
	LocationThingType = "l"
)

func newClient(addr, apiKey, apiSecret string) (*Client, error) {
	w, err := newHttpWriter(addr, apiKey, apiSecret, "30s", false)
	if err != nil {
		return nil, err
	}
	client := &Client{writer: w}
	return client, err
}

func New(addr, apiKey, apiSecret string) (*Client, error) {
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

// Gauge measures the value of a metric at a particular time.
func (c *Client) Gauge(thingId string, value float64) error {
	thingValue := make(map[string]interface{})
	thingValue[GuaugeThingType] = value

	tm := &ThingMetric{
		ThingId:    thingId,
		ThingValue: thingValue,
	}

	return c.sendThing(tm)
}

func (c *Client) Location(thingId string, geoHash string) error {
	thingValue := make(map[string]interface{})
	thingValue[LocationThingType] = geoHash

	tm := &ThingMetric{
		ThingId:    thingId,
		ThingValue: thingValue,
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
	d, err := json.Marshal(c.things)
	if err != nil {
		return nil
	}

	_, err = c.writer.Write(d)

	if len(c.things) > 0 {
		c.things = c.things[:0]
	}

	return err
}
