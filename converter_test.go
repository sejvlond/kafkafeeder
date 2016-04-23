package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConvertString(t *testing.T) {
	cfg := TopicConfig{
		Topic:     "topic",
		Type:      "kafkalog",
		Broker:    "kafka",
		Retention: 24 * time.Hour,
		Ack:       ACK_DISK_WRITE,
	}
	var b bytes.Buffer
	c, err := NewConverter(map[string][]string{
		"kafka": []string{"kafka1.dev:9092", "kafka2.dev:9092", "kafka3.dev:9092"},
	})
	assert.Nil(t, err)
	err = c.ConvertTopic("name", "/tmp", &cfg, &b)
	assert.Nil(t, err)
	assert.Equal(t, b.String(), `
[KafkaOutput_#tmpname]
type = "KafkaOutput"
message_matcher = "Type == '#tmpname'"
encoder = "Encoder_#tmpname"
addrs = ["kafka1.dev:9092","kafka2.dev:9092","kafka3.dev:9092"]
partitioner = "Hash"
hash_variable = "Fields[key]"
topic = "topic"
required_acks = "WaitForAll"
on_error = "Retry"
error_tries = 0
error_timeout = 10000
create_checkpoints = true
checkpoint_interval = 60
max_buffered_bytes = 102400
max_buffer_time = 15000

[Decoder_#tmpname]
type = "KafkalogDecoder"
msg_type = "#tmpname"

[Encoder_#tmpname]
type = "PayloadEncoder"
append_newlines = false

[Splitter_#tmpname]
type = "KafkalogSplitter"

[LogstreamerInput_#tmpname]
type = "LogstreamerInput"
splitter = "Splitter_#tmpname"
decoder = "Decoder_#tmpname"
log_directory = "/tmp"
file_match = '(?P<Date>\d+)_(?P<Time>\d+)_\d+_UTC-name\.szn'
priority = ["Date", "Time"]
`)
}
