package main

const (
	DefaultPort = 8080
)

type Config struct {
	IP       string
	Port     int
	MongoUri string
}

var g_config = Config{}
