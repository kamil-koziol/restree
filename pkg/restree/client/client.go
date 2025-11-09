package client

import "net/http"

type Client struct {
	http.Client
}

func New(transport *http.Transport) *Client {
	return &Client{
		http.Client{
			Transport: transport,
		},
	}
}
