# Peers

Participating YARPC libraries must support multiple peers for every transport.

The peers should be configurable in YAML:

```yaml
yarpc:
  name: myservice
  outbounds:

    # the following 2 outbounds are explicit
    - service: users
      http:
      peers:
        - scheme: http
          host: 127.0.0.1
          port: 5436
        - scheme: http
          host: 127.0.0.1
          port: 5437

    - services:
        - sms
        - email
      tchannel:
        peers:
          - host: 127.0.0.1
            port: 5433
          - host: 127.0.0.1
            port: 5455

    # this outbound uses a stock config, which has the multiple peers inside
    - service: blackbear
      http:
        with: sidecar
```

And during instantiaton of the RPC-object:

```
rpc := yarpc.New(yarpc.Config{
    Name: "myservice",
    Outbounds: transport.Outbounds{
        "users": htt.NewOutbound([
            "http://127.0.0.1:5436/",
            "http://127.0.0.1:5437/",
        ]),
        "sms": tch.NewOutbound(tch, tch.Peers([
            "127.0.0.1:5433",
            "127.0.0.1:5435",
        ])),
    },
)
rpc.Start()
```

This approach enables peer-based features like:

* load balancing (TBD)
* retries (TBD)
* circuit breaking (TBD)

When instantiating the RPC object, it will be (TBD) possible to configure these features like so:


```
rpc := yarpc.New(yarpc.Config{
    Name: "myservice",
    Outbounds: transport.Outbounds{
        ...
    },
    LoadBalancingPolicy: balance.NewRoundRobin(),
    RetryPolicy: retry.NewPolicy(retry.MaxAttempts(7)),
    CircuitBreaker: circuit.New(),
)
rpc.Start()
```

Internally, these configs should be combined into a Transport Options object that gets passed to the
transport during `rpc.Start()`. This lets the the tranport know *how* to implement these features, while letting
the transport maintain fine-grain control over wire-level details like connection management.

Transports should document how they handle these policies, or if they are unable to entirely.
