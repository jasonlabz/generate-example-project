package resource

import (
	"github.com/jasonlabz/potato/es"
	"github.com/jasonlabz/potato/goredis"
	"github.com/jasonlabz/potato/log"
	"github.com/jasonlabz/potato/rabbitmqx"
)

// 文件服务账号密码
var (
	Username string
	Password string
)

// Logger 日志对象
var Logger *log.LoggerWrapper

// RMQClient rabbitmq 客户端
var RMQClient *rabbitmqx.RabbitMQOperator

// RedisClient redis 客户端
var RedisClient *goredis.RedisOperator

// EsClient es 客户端
var EsClient *es.ElasticSearchOperator
