package kafka

import (
	"context"
	"log"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
)

type Configuration struct {
	Kafka struct {
		Address       string `yaml:"address"`
		ProducerTopic string `yaml:"producerTopic"`
	} `yaml:"kafka"`
}

var config = getKafkaConfig()

func getKafkaConfig() Configuration {
	viper.SetConfigFile("config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln("Error reading config file, ", err)
	}

	var configuration Configuration
	err = viper.Unmarshal(&configuration)
	if err != nil {
		log.Fatalln("Unable to decode into struct, ", err)
	}
	createKafkaTopics(configuration.Kafka.ProducerTopic, configuration)
	return configuration
}

func createKafkaTopics(topic string, config Configuration) {
	conn, err := kafka.Dial("tcp", config.Kafka.Address) 
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()
	topicConfigs := []kafka.TopicConfig{
		kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}
	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}
}

func ProduceToKafka(data string) error {
	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{config.Kafka.Address},
		Topic:   config.Kafka.ProducerTopic,
	})
	ctx := context.Background()
	err := kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte("0"),
		Value: []byte(data),
	})
	if err != nil {
		log.Println("failed to send message to kafka, err:", err)
	}
	return err
}
