package main

import "time"

// options are the command line options parsed by go-flags
var options struct {
	BindAddress string        `short:"a" long:"bind-address" env:"BIND_ADDRESS" default:":8080" description:"The address on which the server listens."`
	CacheTTL    time.Duration `long:"ttl" env:"CACHE_TTL" default:"1h" description:"The amount of time after which cached events expire and need to be re-fetched."`
	CertFile    string        `long:"cert-file" env:"TLS_CERT" description:"Path to the TLS server certificate."`
	KeyFile     string        `long:"key-file" env:"TLS_KEY" description:"Path to the TLS private key."`
	Creator     string        `short:"c" long:"creator" env:"ICS_CREATOR" default:"elphi-calendar" description:"The value of the creator field in generated ICS files."`
	Name        string        `short:"n" long:"name" env:"CALENDAR_NAME" default:"Elbphilharmonie Merkliste" description:"The suggested name for the calendar."`
}
