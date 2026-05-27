package apiclient

import (
	"net/http"
	"time"
)

func (c *Client) sharedTransport() *http.Transport {
	c.initHTTP()
	return c.transport
}

func (c *Client) apiHTTP() *http.Client {
	c.initHTTP()
	return c.apiClient
}

func (c *Client) sseHTTP() *http.Client {
	c.initHTTP()
	return c.sseClient
}

func (c *Client) initHTTP() {
	c.httpOnce.Do(func() {
		c.transport = &http.Transport{
			DialContext:           c.Dial.DialContext,
			MaxIdleConnsPerHost:   8,
			MaxConnsPerHost:       16,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		}
		c.apiClient = &http.Client{
			Timeout:   120 * time.Second,
			Transport: c.transport,
		}
		c.sseClient = &http.Client{
			Timeout:   0,
			Transport: c.transport,
		}
	})
}
