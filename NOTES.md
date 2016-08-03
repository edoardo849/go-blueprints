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
- HTTP methods describe the kind of action to take. GET is to read, POST is to create, UPDATE/PUT is to update, DELETE is to delete.
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

Run https://github.com/deviantony/docker-elk and then install graph on it.
https://www.elastic.co/blog/elasticsearch-docker-plugin-management

## Context Package.
Following article here: https://blog.golang.org/context

Pattern for API, packages:
- server: provides the main function and the handlers
- userip: provides functions for extracting user IP address and associating it with a Context (Security)
- google: provides Search functions for sending a query to Google. (Business Logic)

### The server program
The server program has the logic for managing a server. In particular, provides the main entry-point and the handlers.


So.. API keys are NOT passed around by Context because it's not properly a state. Context is for STATES only.

## Colibri proposal
- server : handlers, routers, middleware structure... not middleware logic (like authentication)
- colibri : Colibri business logic, like talking to the database etc...
- aimia: interface for talking to AIMIA APIs

https://tip.golang.org/src/net/http/http.go

## Context
https://peter.bourgon.org/blog/2016/07/11/context.html

Contexts are, among other things, a method of moving information between callsites in a request chain. In many cases this reduces to moving information between middlewares.

The first thing to understand about passing values through the context is that it’s completely type-unsafe, and cannot be checked by the compiler.

So, if you can avoid using the context to pass information around, you probably should.

When we write functions with a lot of parameters, we generally don’t throw up our hands at some threshold and write.

But there are some classes of information for which a context is necessary. This is so-called **request scoped data**, i.e. information that can only exist once a request has begun.

Good examples of request scoped data include
- user IDs extracted from headers,
- authentication tokens tied to cookies or session IDs,
- distributed tracing IDs, and so on.

One important property of that data is that **it might not be present**. So, for example, if your middleware tries to extract an auth token from the context to do some work, be sure to explicitly code the error path when the token isn’t present, e.g. by responding with 401 Unauthorized.

To know if you should use the context, ask yourself if the information you’re putting in there is available to the middleware chain before the request lifecycle begins. A database handle, or a logger, is generally created with the server, not with the request. So don’t use the context to pass those things around; instead, provide them to the middleware(s) that need them at construction time.

I think a good rule of thumb might be: use context to store values, like strings and data structs; avoid using it to store references, like pointers or handles. As Sameer pointed out, this isn’t a bulletproof rule: **you could make a case for a request-scoped logger**, which could go into the context. But it’s a good place to start.

```go
// Don't do this.
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
        db := r.Context().Value(DatabaseContextKey).(Database)
        // Use db somehow
        next.ServeHTTP(w, r)
    })
}

// Do this instead.
func MyMiddleware(db Database, next http.Handler) http.Handler {
    return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
        // Use db here.
        next.ServeHTTP(w, r)
    })
}

```

## Logger
https://peter.bourgon.org/go-best-practices-2016/#top-tip-9
- Log only actionable information, which will be read by a human or a machine
- Avoid fine-grained log levels — info and debug are probably enough
- Use structured logging — I’m biased, but I recommend go-kit/log https://github.com/go-kit/kit/tree/master/log
- Loggers are dependencies!

https://twitter.com/peterbourgon/status/752022730812317696

## testing
https://www.youtube.com/watch?v=yszygk1cpEc

use the standard Go

## vendoring
Without getting too deep in the weefds, the lesson is clear: libraries should never vendor dependencies.


# The Top Tips

1. Put $GOPATH/bin in your $PATH, so installed binaries are easily accessible.  link
1. If your repo foo is primarily a binary, put your library code in a lib/ subdir, and call it package foo.   https://github.com/tsenart/vegeta
1. If your repo is primarily a library, put your binaries in separate subdirectories under cmd/.
1. Defer to Andrew Gerrand’s naming conventions.  
1. Only func main has the right to decide which flags are available to the user.  
1. Use struct literal initialization to avoid invalid intermediate state.  
1. Avoid nil checks via default no-op implementations.  
1. Make the zero value useful, especially in config objects.  
1. Make dependencies explicit!  
1. Loggers are dependencies, just like references to other components, database handles, commandline flags, etc.  
1. Use many small interfaces to model dependencies.  
1. Tests only need to test the thing being tested.  
1. 1.  a top tool to vendor dependencies for your binary.  
Libraries should never vendor their dependencies.  
Prefer go install to go build.  

If your repo is a combination of binaries and libraries, then you should determine which is the primary artifact, and put that in the root of the repo. For example, if your repo is primarily a binary, but can also be used as a library, then you’ll want to structure it like this:


Code Review comments
https://github.com/golang/go/wiki/CodeReviewComments
