package main

import (
	"errors"
	"io/ioutil"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

type kafkafeederYamlTopic struct {
	Topic     string `yaml:"topic"`
	Type      string `yaml:"type"`
	Broker    string `yaml:"broker"`
	Retention string `yaml:"retention"`
	Ack       string `yaml:"ack"`
}
type kafkafeederYaml struct {
	Topics map[string]*kafkafeederYamlTopic `yaml:"topics"`
}

func newTopicConfig(kfYaml *kafkafeederYamlTopic) (_ *TopicConfig, err error) {
	if kfYaml.Topic == "" {
		return nil, errors.New("Topic can not be empty")
	}
	if kfYaml.Type == "" {
		return nil, errors.New("Type can not be empty")
	}
	if kfYaml.Broker == "" {
		return nil, errors.New("Broker can not be empty")
	}
	var retention time.Duration
	if kfYaml.Retention == "" {
		retention = -1 // just a negative value
	} else {
		if retention, err = time.ParseDuration(kfYaml.Retention); err != nil {
			// if parsing fails, try to append "h"
			if retention, err = time.ParseDuration(
				kfYaml.Retention + "h"); err != nil {
				return nil, errors.New("Invalid retention value")
			}
		}
		if retention <= 0 {
			return nil, errors.New("Retention have to be positive")
		}
	}
	if kfYaml.Ack == "" {
		kfYaml.Ack = "-1"
	}
	var ack int
	if ack, err = strconv.Atoi(kfYaml.Ack); err != nil {
		return nil, errors.New("Unknown ack level")
	}
	if ack != ACK_DISK_WRITE && ack != ACK_MEMORY_WRITE && ack != ACK_DISABLED {
		return nil, errors.New("Unknown ack level")
	}

	return &TopicConfig{
		Topic:     kfYaml.Topic,
		Type:      kfYaml.Type,
		Broker:    kfYaml.Broker,
		Retention: retention,
		Ack:       ack,
	}, nil
}

func newLogConfig(kfYaml *kafkafeederYaml) (cfg *LogConfig, err error) {
	if len(kfYaml.Topics) == 0 {
		return nil, errors.New("There is no topic in kafkafeeder")
	}
	cfg = &LogConfig{
		Topics: make(map[string]*TopicConfig, len(kfYaml.Topics)),
	}
	for name, topicCfg := range kfYaml.Topics {
		cfg.Topics[name], err = newTopicConfig(topicCfg)
		if err != nil {
			return
		}
	}
	return
}

func Parse(data []byte) (*LogConfig, error) {
	kfYaml := &kafkafeederYaml{}
	if err := yaml.Unmarshal(data, kfYaml); err != nil {
		return nil, err
	}
	return newLogConfig(kfYaml)
}

func ParseFile(filename string) (cfg *LogConfig, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return Parse(data)
}
