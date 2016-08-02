Project Notes
====

*Tutorial from Ryer's book Go Programming Blueprints, Chapter 5 & 6 .* Maybe could also link it to the new Elastic Graph for analysis...

We'll build:
1. Twittervotes, a program that reads tweets and pushes the votes into the messaging queue.
1. Counter, a program that listens for votes on the messaging queue and periodically saves the results in the MongoDB database.


For NSQ and Mongo we will run them into Docker. Rune the environment:
```bash
docker-compose up -d
docker-compose stop
```

Installing go dependencies for nsq and mongo

```bash

go get github.com/bitly/go-nsq
go get gopkg.in/mgo.v2

```

Gracefully stop the programs.


To run the program first create a topic in Mongo

```bash
mongo
use ballots
db.polls.insert({"title":"US Election", "options":["Trump", "Clinton"]})
```

Connect to the container using Kitematic GUI... easier... then on the admin nsq type

```bash
nsq_tail --topic="votes" --lookupd-http-address=192.168.99.100:4161 --broadcast-address=192.168.99.100
```
then load the environment variables
```bash
source ./prod_setup.sh
make
```
To Load data into ElasticSearch
https://hub.docker.com/r/willdurand/elk/
http://williamdurand.fr/2014/12/17/elasticsearch-logstash-kibana-with-docker/
https://www.elastic.co/blog/how-to-make-a-dockerfile-for-elasticsearch


## Key issues
- Execution pipeline
- Share data between handlers
- Best practices for writing handlers responsivle for exposing data
- Room for future implementations
- Prevent adding dependencies on external packages

### RESTful API design
- HTTP methods describe the kind of action to take. GET is to read, POST is to create, UPDATE is to update, DELETE is to delete.
- Data is expressed as a collection of resources
- Actions are expressed as changes to data
- URLs are used to refer to specific data
- HTTP headers are used to describe the kind of representation coming into and going out of the server

## Sharing data between handlers
Goal: keep our handlers as pure as the `http.Handler` interface from the Go Library.

```go
type HandlerFunc func(http.ResponseWriter, *http.Request)
```

therefore, we cannot create and manage database session objects in one place and pass them into our handlers.

We are going to create an in-memory map of per-request data, and provide an easy way for handlers to access it.
