Crossdock Tests
===============

Participating YARPC libraries must ship a Crossdock client with the following
behaviors implemented.

These behaviors stand up clients and send requests to [test subjects][]
running on each container to validate the behavior of YARPC clients and servers
against both well-behaved and misbehaving clients and servers.

[test subject]: test-subject.md

The Crossdock client must arrange for the test subjects to run before listening
on the Crossdock client socket.

raw
---

For a given transport and server, make a call to the `echo/raw` procedure,
returning a list of results.

```
curl localhost:8080'?behavior=raw&server=localhost&transport=tchannel'
```

```json
[
    {
        "transport": "tchannel",
        "encoding": "raw",
        "server": "localhost",
        "status": "passed",
        "output": "made tchannel raw request to echo/raw - got resp: 34bdchk"
    }
]
```

Dimensions: `server`, `transport`

json
----

For a given transport and server, make a call to the `echo` json procedure,
returning a list of results.

```
curl localhost:8080'?behavior=json&server=localhost&transport=http'
```

```json
[
    {
        "transport": "http",
        "encoding": "json",
        "server": "yarpc",
        "status": "passed",
        "output": "made http json request to echo - got token: 34bdchk"
    }
]
```

Dimensions: `server`, `transport`

thrift
------

For a given transport and server, make a call to the `Echo::echo` Thrift
procedure in [Echo.thrift](Echo.thrift), returning a list of results.

```
curl localhost:8080'?behavior=thrift&server=localhost&transport=http'
```

```json
[
    {
        "transport": "http",
        "encoding": "thrift",
        "server": "yarpc",
        "status": "passed",
        "output": "made http thrift request to Echo::echo - got token: 34bdchk"
    }
]
```

Dimensions: `server`, `transport`

thrift-gauntlet
---------------

For a given server and transport, make a call to every Thrift procedure in
[ThriftTest.thrift](ThriftTest.thrift), returning a list of results for each
call.

```
curl localhost:8080'?behavior=thrift-gauntlet&server=localhost&transport=tchannel'
```

```json
[
    {
        "transport": "tchannel",
        "encoding": "thrift",
        "server": "yarpc",
        "procedure": "ThriftTest::testString",
        "status": "passed",
        "output": "made tchannel thrift request to ThriftTest::testString - got resp: stringyding"
    },
    {
        "transport": "http",
        "encoding": "thrift",
        "server": "yarpc",
        "procedure": "ThriftTest::testString",
        "status": "passed",
        "output": "made http thrift request to ThriftTest::testString - got resp: stringyding"
    }
]
```

Each Thrift `procedure` in [ThriftTest.thrift](ThrifTest.thrift) produces an
independent result.

Dimensions: `server`, `transport`.

headers
-------

For each transport and each encoding, make a call to an echo procedure for that encoding to verify:

1. Sending valid header
2. Sending non-string values
3. Sending empty string values
4. Sending no headers, eg an empty map
5. Varying key casing - to ensure case insensitivity
6. A header name like `Rpc-Procedure` that would ordinarily conflict with RPC's own HTTP headers.

Return a response for each call:

```json
[
    {"status": "passed", "output": "made a tchannel+raw call to verify valid headers, got {'this': 'worked'}"},
    {"status": "passed", "output": "made a tchannel+json call to verify valid headers, got {'this': 'worked'}"},
    {"status": "passed", "output": "made a tchannel+thrift call to verify valid headers, got {'this': 'worked'}"},
    {"status": "passed", "output": "made a http+raw call to verify valid headers, got {'this': 'worked'}"},
    {"status": "passed", "output": "made a http+json call to verify valid headers, got {'this': 'worked'}"},
    {"status": "passed", "output": "made a http+thrift call to verify valid headers, got {'this': 'worked'}"},
]

```

Dimensions: `server`, `transport`, `encoding`

errors
------

For each transport, trigger and verify every error listed in [errors.md](errors.md), and return a result for each.

```json
[
    {"status": "passed", "output": "got NoProcedureForOutboundRequestError while using the http transport"},
    {"status": "passed", "output": "got NoProcedureForOutboundRequestError while using the tchannel transpolrt"},
]
```

Dimensions: `server`, `transport`, errors listed in [errors.md](errors.md)

compat-apache-thrift-server
---------------------------

For each Apache Thrift generated server, make a call from a YARPC Thrift client for every Thrift procedure in [ThriftTest.thrift](ThriftTest.thrift),
returning a list of results for each call.

```json
[
    {"status": "passed", "output": "made a YARPC request to an ApacheThrift+HTTP server's ThriftTest::testString - got resp: hollerback"},
    {"status": "passed", "output": "made a YARPC request to an ApacheThrift+HTTP server's ThriftTest::testI32 - got resp: 100"},
]
```

Dimensions: Apache Thrift generated server, each Thrift `procedure` in [ThriftTest.thrift](ThriftTest.thrift)

compat-tchannel-client
----------------------

For every encoding, make a TChannel call to a YARPC server, including for every Thrift procedure in [ThriftTest.thrift](ThriftTest.thrift), and
return a list of results for each call.

```json
[
    {"status": "passed", "output": "made a TChannel request to a YARPC server's raw echo/raw procedure - got resp: sweet"},
    {"status": "passed", "output": "made a TChannel request to a YARPC server's json echo procedure - got resp: 3455b6d"},
    {"status": "passed", "output": "made a TChannel request to a YARPC server's thrift ThriftTest::testI32 - got resp: 100"},
]
```

Dimensions: `server`, TChannel client, each Thrift `procedure` in [ThriftTest.thrift](ThriftTest.thrift)

compat-tchannel-server
----------------------

For every encoding, make a YARPC call to a TChannel server, including for every Thrift procedure in [ThriftTest.thrift](ThriftTest.thrift), and
return a list of results for each call.

```json
[
    {"status": "passed", "output": "made a YARPC request to a TChannel server's raw echo/raw procedure - got resp: sweet"},
    {"status": "passed", "output": "made a YARPC request to a TChannel server's json echo procedure - got resp: 3455b6d"},
    {"status": "passed", "output": "made a YARPC request to a TChannel server's thrift ThriftTest::testI32 - got resp: 100"},
]
```

Dimensions: TChannel server, each Thrift `procedure` in [ThriftTest.thrift](ThriftTest.thrift)
