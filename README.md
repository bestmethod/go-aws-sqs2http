# go-aws-sqs2http
Go server application which listens to SQS on long poll and send the messages it gets to REST HTTP interface (json post).

## Usage
```
git clone https://github.com/bestmethod/go-aws-sqs2http.git
cd go-aws-sqs2http
go build sqs2http.go
```
Edit the configuration text file and run the resulting binary file :) Done.


The only files needed is the sqs.http binary created by running the go build and the configuration file.

