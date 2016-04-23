package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseString(t *testing.T) {
	var data = `
topics:
  componenta:
    topic: TOPIC
    type: TYPE
    broker: BROKER
    retention: 24h
    ack: -1
  componenta2:
    topic: TOPIC2
    type: TYPE2
    broker: BROKER2
`
	cfg, err := Parse([]byte(data))
	assert.Nil(t, err)

	assert.Equal(t, cfg.Topics["componenta"].Topic, "TOPIC")
	assert.Equal(t, cfg.Topics["componenta"].Type, "TYPE")
	assert.Equal(t, cfg.Topics["componenta"].Broker, "BROKER")
	rtn, err := time.ParseDuration("24h")
	assert.Nil(t, err)
	assert.Equal(t, cfg.Topics["componenta"].Retention, rtn)
	assert.Equal(t, cfg.Topics["componenta"].Ack, -1)

	assert.Equal(t, cfg.Topics["componenta2"].Topic, "TOPIC2")
	assert.Equal(t, cfg.Topics["componenta2"].Type, "TYPE2")
	assert.Equal(t, cfg.Topics["componenta2"].Broker, "BROKER2")
	assert.True(t, cfg.Topics["componenta2"].Retention < 0)
	assert.Equal(t, cfg.Topics["componenta2"].Ack, -1)

	assert.Equal(t, cfg.Directory, "")
}

func TestParseFile(t *testing.T) {
	cfg, err := ParseFile("./tests/kafkafeeder.yaml")
	assert.Nil(t, err)
	assert.Equal(t, cfg.Topics["test-zpravy"].Topic, "test-topic")
	assert.Equal(t, cfg.Topics["test-zpravy"].Type, "kafkalog")
	assert.Equal(t, cfg.Topics["test-zpravy"].Broker, "kafka")
	rtn, err := time.ParseDuration("24h")
	assert.Nil(t, err)
	assert.Equal(t, cfg.Topics["test-zpravy"].Retention, rtn)
	assert.Equal(t, cfg.Topics["test-zpravy"].Ack, -1)

	assert.Equal(t, cfg.Directory, "")
}
