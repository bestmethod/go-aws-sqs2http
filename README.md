# go-aws-sqs2http
Go server application which listens to SQS on long poll and send the messages it gets to REST HTTP interface (json post).

## Usage
The below will work, but you will need to install the required imports manually using go get as well (see import statements containing the 'github' name in the start of the go file and go get them).
```
git clone https://github.com/bestmethod/go-aws-sqs2http.git
cd go-aws-sqs2http
go build sqs2http.go
```
Edit the configuration text file and run the resulting binary file :) Done.


The only files needed is the sqs.http binary created by running the go build and the configuration file.

## Using go get
```
go get github.com/bestmethod/go-aws-sqs2http
cd ~/go/src/github.com/bestmethod/go-aws-sqs2http
go build sqs2http.go
```

This should auto-resolve the requirements.

Again, edit the configuration file and start the resulting binary. All you need to ship to production is the binary and the configuration file.

The actual idea is that the configuration file can be built using Jenkins (or similar) for the production system in a CI/CD env.
