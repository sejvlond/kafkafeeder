package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"
)

const heka_template = `
[KafkaOutput_{{.Id}}]
type = "KafkaOutput"
message_matcher = "Type == '{{.Id}}'"
encoder = "Encoder_{{.Id}}"
addrs = {{.Output.Brokers}}
partitioner = "Hash"
hash_variable = "Fields[key]"
topic = "{{.Output.Topic}}"
required_acks = "{{.Output.Ack}}"
on_error = "Retry"
error_tries = 0
error_timeout = 10000
create_checkpoints = true
checkpoint_interval = 60
max_buffered_bytes = 102400
max_buffer_time = 15000

[Decoder_{{.Id}}]
{{.Decoder}}

[Encoder_{{.Id}}]
{{.Encoder}}

[Splitter_{{.Id}}]
{{.Splitter}}

[LogstreamerInput_{{.Id}}]
type = "LogstreamerInput"
splitter = "Splitter_{{.Id}}"
decoder = "Decoder_{{.Id}}"
log_directory = "{{.Input.Directory}}"
file_match = '{{.Input.FileMatch}}'
priority = {{.Input.Priority}}
`
const replacement = "#"

var idregexp *regexp.Regexp

func init() {
	idregexp = regexp.MustCompile(`[^a-zA-Z\-_0-9]`)
}

type Converter struct {
	hekaTemplate *template.Template
	brokers      map[string]string
}

func NewConverter(brokers map[string][]string) (*Converter, error) {
	hekaTemplate, err := template.New("heka_conf").Parse(heka_template)
	if err != nil {
		return nil, err
	}
	brokersStr := make(map[string]string)
	for key, br := range brokers {
		brokersStr[key] = `["` + strings.Join(br, `","`) + `"]`
	}
	cnv := &Converter{
		hekaTemplate: hekaTemplate,
		brokers:      brokersStr,
	}
	return cnv, nil
}

func IdFromString(str string) string {
	return string(idregexp.ReplaceAll([]byte(str), []byte(replacement)))
}

type TemplateData struct {
	Id       string
	Decoder  string
	Encoder  string
	Splitter string
	Output   struct {
		Topic   string
		Brokers string
		Ack     string
	}
	Input struct {
		Directory string
		FileMatch string
		Priority  string
	}
}

func (c *Converter) ConvertTopic(name, dir string, cfg *TopicConfig,
	wr io.Writer) error {

	data := TemplateData{}
	data.Id = IdFromString(dir + name)
	data.Output.Topic = cfg.Topic
	data.Encoder = `type = "PayloadEncoder"` + "\n" +
		`append_newlines = false`
	data.Input.Directory = dir
	switch cfg.Type {
	case "kafkalog":
		data.Input.FileMatch = fmt.Sprintf(
			`(?P<Date>\d+)_(?P<Time>\d+)_\d+_UTC-%s\.szn`, name)
		data.Input.Priority = `["Date", "Time"]`
		data.Decoder = fmt.Sprintf(
			`type = "KafkalogDecoder"`+"\n"+
				`msg_type = "%s"`, data.Id)
		data.Splitter = `type = "KafkalogSplitter"`
		break
	default:
		return fmt.Errorf("Convert Topic: unsupported type %q", cfg.Type)
	}

	var ok bool
	data.Output.Brokers, ok = c.brokers[cfg.Broker]
	if !ok {
		return fmt.Errorf("Convert Topic: unsupported broker %q", cfg.Broker)
	}

	switch cfg.Ack {
	case ACK_DISABLED:
		data.Output.Ack = `NoResponse`
		break
	case ACK_MEMORY_WRITE:
		data.Output.Ack = `WaitForLocal`
		break
	case ACK_DISK_WRITE:
		data.Output.Ack = `WaitForAll`
		break
	default:
		return fmt.Errorf("Convert Topic: unsupported ack level %q", cfg.Ack)
	}

	return c.hekaTemplate.Execute(wr, data)
}

func (c *Converter) Convert(cfg *LogConfig, wr io.Writer) (err error) {
	for name, topicCfg := range cfg.Topics {
		if err = c.ConvertTopic(name, cfg.Directory, topicCfg, wr); err != nil {
			return
		}
	}
	return
}
