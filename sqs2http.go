package main

//TODO: unittests

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bestmethod/go-logger"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"strings"
	"net/http"
	"time"
	"bytes"
	"io/ioutil"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"io"
)

type configLogger struct {
	LogToConsole  bool
	ErrorToStderr bool
	LogToDevlog   bool
}

type configSqs struct {
	KeyId       *string
	SecretKey   *string
	QueueName   *string
	QueueRegion *string
	LongPollWaitSeconds int64
	HideMessageForSeconds int64
	OnReceiveError *string
}

type configRest struct {
	Endpoint   *string
	Method     *string
	Test       *string
	TestMethod *string
	ExitOnFail bool
	ExpectStatusCode int
	ExpectTestStatusCode int
}

type config struct {
	Logger *configLogger
	Sqs    *configSqs
	Rest   *configRest
}

var logger *Logger.Logger

//for monkey patching
var httpGet = http.Get
var httpHead = http.Head
var awsNewSession = session.NewSession
var awsSqs = sqs.New
var awsQueueUrl = getQueueUrl
var awsReceiveMessage = svcReceiveMessage
var awsDeleteMessage = svcDeleteMessage
var makeRestCall = restCall

//tests - do we can actually finish
var run = true

func main() {
	//vars
	var configFile string
	var err error
	var conf config
	var resp *http.Response
	var message *sqs.Message

	//parse config filename
	fmt.Printf("Parsing command line arguments...")
	flag.StringVar(&configFile, "config", "sqs-config.txt", "Specify configuration file name and path")
	flag.Parse()
	fmt.Println("OK")

	//init basic logger
	fmt.Printf("Initializing logger...")
	logger = new(Logger.Logger)
	err = logger.Init("LOADER", "Jarvis-SQS-Handler", Logger.LEVEL_DEBUG|Logger.LEVEL_INFO|Logger.LEVEL_WARN, Logger.LEVEL_ERROR|Logger.LEVEL_CRITICAL, Logger.LEVEL_NONE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CRITICAL Could not initialize logger. Quitting. Details: %s\n", err)
		os.Exit(3)
	}
	fmt.Println("OK")

	//load configuration
	logger.Info("Loading configuration file")
	if _, err = toml.DecodeFile(configFile, &conf); err != nil {
		logger.Fatal(fmt.Sprintf("Could not parse configuration file. Quitting. Details: %s", err), 1)
	}

	//reload logger
	logger.Info("Reloading logger parameters")
	oldLogger := logger
	logger = new(Logger.Logger)
	var devlog int
	var stdout int
	var stderr int
	if conf.Logger.LogToDevlog == true {
		devlog = Logger.LEVEL_DEBUG | Logger.LEVEL_INFO | Logger.LEVEL_WARN | Logger.LEVEL_ERROR | Logger.LEVEL_CRITICAL
	} else {
		devlog = Logger.LEVEL_NONE
	}
	if conf.Logger.LogToConsole == true {
		stdout = Logger.LEVEL_DEBUG | Logger.LEVEL_INFO | Logger.LEVEL_WARN
	}
	if conf.Logger.ErrorToStderr == true {
		stderr = Logger.LEVEL_ERROR | Logger.LEVEL_CRITICAL
	} else {
		stderr = Logger.LEVEL_NONE
		stdout = stdout | Logger.LEVEL_ERROR | Logger.LEVEL_CRITICAL
	}
	err = logger.Init(fmt.Sprintf("%s:%s", strings.ToUpper(*conf.Sqs.QueueRegion), strings.ToUpper(*conf.Sqs.QueueName)), "Jarvis-SQS-Handler", stdout, stderr, devlog)
	if err != nil {
		oldLogger.Fatal(fmt.Sprintf("Could not reload logger from configuration, devlog issue? Quitting. Details: %s", err), 4)
	}

	//test rest interface connectivity
	logger.Info("Testing REST interface connectivity")
	if *conf.Rest.TestMethod == "GET" {
		resp, err = httpGet(*conf.Rest.Test)
	} else if *conf.Rest.TestMethod == "HEAD" {
		resp, err = httpHead(*conf.Rest.Test)
	} else if *conf.Rest.TestMethod == "NONE" {
		resp = nil
		err = nil
	} else {
		logger.Fatal("The test method MUST be GET|HEAD|NONE, with NONE meaning do-not-run.",5)
	}
	if err != nil {
		if conf.Rest.ExitOnFail == true {
			logger.Fatal(fmt.Sprintf("REST test failed. Quitting. Details: %s",err), 6)
		} else {
			logger.Error(fmt.Sprintf("REST test failed. Details: %s",err))
		}
	}
	if err == nil && resp != nil {
		if resp.StatusCode != conf.Rest.ExpectTestStatusCode {
			if conf.Rest.ExitOnFail == true {
				logger.Fatal(fmt.Sprintf("REST test failed with status code %d. Quitting. Status: %s, Header: %s, Body: %s", resp.StatusCode, resp.Status, resp.Header, resp.Body), 6)
			} else {
				logger.Error(fmt.Sprintf("REST test failed with status code %d. Status: %s, Header: %s, Body: %s", resp.StatusCode, resp.Status, resp.Header, resp.Body))
			}
		}
	}

	logger.Info("Initializing SQS")

	//create a session
	svc, qURL := conf.initSqs()

	//pull in loop forever
	logger.Info("Starting message pulling")

	for run == true {
		logger.Debug("Running Loop")
		message, err = conf.receiveMessage(svc, qURL)
		if err != nil {
			if *conf.Sqs.OnReceiveError == "IGNORE" {
				logger.Error(fmt.Sprintf("Could not pull messages. Will try again in a second. Details: %s", err))
				time.Sleep(time.Second)
			} else if *conf.Sqs.OnReceiveError == "QUIT" {
				logger.Fatal(fmt.Sprintf("Could not pull messages, quitting. Details: %s", err),9)
			} else {
				logger.Error(fmt.Sprintf("Could not pull messages. Reinitializing connection. Details: %s", err))
				time.Sleep(time.Second)
				svc, qURL = conf.initSqs()
			}
		} else if message != nil {
			go conf.processMessage(svc, qURL, message)
		}
	}
}

func (c config) initSqs() (*sqs.SQS,string) {
	//create connection session
	var sess *session.Session
	var err error
	if *c.Sqs.KeyId != "" && *c.Sqs.SecretKey != "" {
		sess, err = awsNewSession(&aws.Config{
			Region:      aws.String(*c.Sqs.QueueRegion),
			Credentials: credentials.NewStaticCredentials(*c.Sqs.KeyId, *c.Sqs.SecretKey, ""),
		})
	} else {
		sess, err = awsNewSession(&aws.Config{
			Region:      aws.String(*c.Sqs.QueueRegion),
			Credentials: credentials.NewEnvCredentials(),
		})
	}
	if err != nil {
		logger.Fatal(fmt.Sprintf("Could not contact AWS with new session. Quitting. Details: %s",err),8)
	}

	svc := awsSqs(sess)

	//get queue url
	result, err := awsQueueUrl(svc,&sqs.GetQueueUrlInput{
		QueueName: aws.String(*c.Sqs.QueueName),
	})

	if err != nil {
		logger.Fatal(fmt.Sprintf("Could not contact AWS to get Queue URL. Quitting. Details: %s",err),7)
	}
	return svc,*result.QueueUrl
}

func getQueueUrl(svc *sqs.SQS,s *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput,error) {
	return svc.GetQueueUrl(s)
}

func restCall(method string, url string, reqbody io.Reader) (*http.Response,[]byte,error) {
	req, err := http.NewRequest(method,url, reqbody)
	req.Header.Set("X-Caller", "Jarvis-SQS-Handler")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil,nil,err
	}
	defer resp.Body.Close()
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		body = nil
	}
	return resp,body,nil
}

func (c config) processMessage(svc *sqs.SQS,qURL string, message *sqs.Message) {
	var err error

	logger.Info(fmt.Sprintf("Processing start messageId=%s",*message.MessageId))
	logger.Debug(fmt.Sprintf("messageId=%s json.marshall",*message.MessageId))
	//build json
	var jsonStr []byte
	jsonStr, err = json.Marshal(message)
	if err != nil {
		logger.Error(fmt.Sprintf("messageId=%s Could not build json from message. Details: %s",*message.MessageId,err))
		return
	}

	//http call
	logger.Debug(fmt.Sprintf("messageId=%s makeRestCall",*message.MessageId))
	resp, body, err := makeRestCall(*c.Rest.Method,*c.Rest.Endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Error(fmt.Sprintf("messageId=%s Could not make REST call, not removing message. Details: %s",*message.MessageId,err))
		return
	}

	if resp.StatusCode != c.Rest.ExpectStatusCode {
		logger.Error(fmt.Sprintf("messageId=%s Wrong status code received: %s, Status: %s, Header: %s",*message.MessageId,resp.StatusCode,resp.Status,resp.Header))
		logger.Debug(fmt.Sprintf("messageId=%s Returned body=%s",*message.MessageId,body))
		return
	}

	//if http call successful
	logger.Debug(fmt.Sprintf("messageId=%s REST success, sqs.deleteMessage",*message.MessageId))
	err = c.deleteMessage(svc,qURL,message.ReceiptHandle)
	if err != nil {
		logger.Error(fmt.Sprintf("messageId=%s Message could not be deleted and will be reprocessed automatically! Details: %s",*message.MessageId,err))
	}
	logger.Info(fmt.Sprintf("Processing finish messageId=%s",*message.MessageId))
}

func svcDeleteMessage(svc *sqs.SQS, s *sqs.DeleteMessageInput) (error,error) {
	_, err := svc.DeleteMessage(s)
	return nil,err
}

func (c config) deleteMessage(svc *sqs.SQS,qURL string, receiptHandle *string) error {
	_, err := awsDeleteMessage(svc,&sqs.DeleteMessageInput{
		QueueUrl:      &qURL,
		ReceiptHandle: receiptHandle,
	})

	return err
}

func svcReceiveMessage (svc *sqs.SQS,s *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput,error) {
	return svc.ReceiveMessage(s)
}

func (c config) receiveMessage(svc *sqs.SQS,qURL string) (*sqs.Message,error) {
	result, err := awsReceiveMessage(svc,&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &qURL,
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(c.Sqs.HideMessageForSeconds),
		WaitTimeSeconds:     aws.Int64(c.Sqs.LongPollWaitSeconds),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Messages) == 0 {
		return nil,nil
	}

	return result.Messages[0],nil
}
