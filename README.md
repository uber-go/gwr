# GWR: Get / Watch / Report -ing of operational data

NOTE: GWR is currently in experimental prototyping phase, buyer beware.

# Using

GWR exposes an HTTP and Resp (Redis Protocol) interface by default on ports
4040 and 4041.

## HTTP

Example for http:

```
$ curl localhost:4040/meta/nouns
- /meta/nouns formats: <no value>
- /request_log formats: <no value>
- /response_log formats: <no value>

$ curl -X WATCH localhost:4040/request_log&
$ curl -X WATCH localhost:4040/response_log&

$ curl localhost:8080/foo
404 page not found                                         # this is the normal curl output
GET /foo                                                   # this comes from the first watch-curl
404 19 text/plain; charset=utf-8                           # this comes from the first watch-curl
```


## Resp

```
$ redis-cli -p 4041 ls                                     # this is a convenience alias for "get /meta/nouns"
1) - /meta/nouns formats: <no value>
2) - /request_log formats: <no value>
3) - /response_log formats: <no value>

$ redis-cli -p 4041 monitor /request_log text /response_log text&
OK

$ curl localhost:8080/bar
404 page not found                                         # this is the curl output
/request_log> GET /bar                                     # this is from redis-cli
/response_log> 404 19 text/plain; charset=utf-8            # so is this, ordering not guaranteed
```

# Integration

To add gwr to a program, all you need to do is call

```
gwrProto.ListenAndServe(nil, nil)
```

The default is to listen http on port `4040` and resp on port `4041`, which can
be changed with the first nil argument above.

# Defining data sources

To define a data source, the easiest way is to implement the
`gwr.GenericDataSource` interface.

`TODO: example`
