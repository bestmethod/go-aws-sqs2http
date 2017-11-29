package main

import (
	"net/http"
	"testing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws/client"
	"io"
	"time"
)

func httpGetHead(url string) (resp *http.Response, err error) {
	resp = new(http.Response)
	resp.StatusCode = 200
	err = nil
	return resp,err
}

func sessionNewSession(cfgs ...*aws.Config) (*session.Session, error) {
	sess := new(session.Session)
	var err error
	err = nil
	return sess,err
}

func sqsNew(client.ConfigProvider, ...*aws.Config) *sqs.SQS {
	svc := new(sqs.SQS)
	return svc
}

func sqsQueueUrl(svc *sqs.SQS,s *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput,error) {
	o := new(sqs.GetQueueUrlOutput)
	surl := "someUrl"
	o.QueueUrl = &surl
	var err error
	err = nil
	return o,err
}

func sqsReceiveMessage (svc *sqs.SQS,s *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput,error) {
	var err error
	err = nil
	o := new(sqs.ReceiveMessageOutput)
	message := new(sqs.Message)
	body := "body"
	mid := "messageid"
	rh := "receiptHandle"
	attr := "attribute1value"
	var mattr1 sqs.MessageAttributeValue
	mattr1.SetDataType("string")
	mattr1.SetStringValue("messageAttribute1stringValue")
	var mattr2 sqs.MessageAttributeValue
	mattr2.SetDataType("binary")
	var val []byte
	val = append(val, 'a')
	val = append(val, 0)
	mattr2.SetBinaryValue(val)
	message.Body = &body
	message.MessageId = &mid
	message.ReceiptHandle = &rh
	message.Attributes = make(map[string]*string)
	message.Attributes["attribute1name"] = &attr
	message.MessageAttributes = make(map[string]*sqs.MessageAttributeValue)
	message.MessageAttributes["messageAttribute1nameString"] = &mattr1
	message.MessageAttributes["messageAttribute1nameBool"] = &mattr2
	o.Messages = append(o.Messages,message)
	return o,err
}

func sqsDeleteMessage(svc *sqs.SQS, s *sqs.DeleteMessageInput) (error,error) {
	return nil,nil
}

func mockRestCall(method string, url string, reqbody io.Reader) (*http.Response,[]byte,error) {
	resp := new(http.Response)
	resp.StatusCode = 200
	resp.Status = "OK"
	return resp,nil,nil
}

func TestFullSuccessfulFlow(t *testing.T) {
	httpGet = httpGetHead
	httpHead = httpGetHead
	awsNewSession = sessionNewSession
	awsSqs = sqsNew
	awsQueueUrl = sqsQueueUrl
	awsReceiveMessage = sqsReceiveMessage
	awsDeleteMessage = sqsDeleteMessage
	makeRestCall = mockRestCall
	go main()
	time.Sleep(time.Second)
	run = false
}
