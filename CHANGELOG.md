# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Observability middleware now emits metrics for panics that occur on the stack
  of an inbound call handler.
- The `transporttest` package now provides a `Pipe` constructor, which creates
  a bound pair of transport layer streams, for testing streaming protocols like
  gRPC.
- The `yarpctest.FakeOutbound` can now send requests to a `transport.Router`.
  This allows end to end testing with a client and server in memory.
  The `OutboundCallOverride`, `OutboundCallOnewayOverride` (new), and
  `OutboundCallStreamOverride` (new) are now a complete set that allow tests to
  hook any of the call behaviors.
- All outbounds now implement `Name` and a new `transport.Namer` interface.
  This will allow outbound observability middleware to carry the transport name
  in metrics properly.
### Changed
- This change reduces the API surface of the peer list implementations to
  remove a previously public embedded type and replace it with implementations
  of the underlying interfaces.
  The new type does not provide all of the public interface of the previous
  concrete types.
  However, we expect that in practice, peer lists are used as either peer.List,
  peer.Chooser, or for the private introspection interface.
### Fixed
- Fixed Streaming Protobuf-flavored-JSON nil pointer panic.
- Log entries for EOF stream messages are now considered successes to avoid
  setting off false alarms.
  The successful log entries still carry the "error" field, which will reflect
  the EOF error.

## [1.42.1] - 2019-11-27 (Gobble)
### Fixed
- Simplified the flow of status change notifications for the gRPC and TChannel
  transports to reduce the liklihood of deadlocks.
- Increase default HTTP timeout to avoid stop timeout errors when the server
  has a new idle connection.
- Close idle connections when the transport is closed.

## [1.42.0] - 2019-10-31 (Spooky)
### Added
- Added fail-fast option to peer lists.  With this option enabled, a peer list
  will return an error if no peers are connected at the time of a call, instead
  of waiting for an available peer or the context to time out.
### Fixed
- Previously, every peer list reported itself as a "single" peer list for
  purposes of debugging, instead of its own name.
- Metrics emit `CodeResourceExhausted` as a client error and `CodeUnimplemented`
  as a server error.
- Simplified the flow of status change notifications for the HTTP transport to
  reduce the liklihood of deadlocks.
- Removed a bug from the gRPC transport that would cause a very rare deadlock
  during production deploys and restarts.
  The gRPC peer release method would synchronize with the connection status
  change monitor loop, waiting for it to exit.
  This would wait forever since retain was called while holding a lock on the
  list.

## [1.41.0] - 2019-10-01
### Fixed
- Fixed TChannel memory pressure that would occur during server-side errors.

## [1.40.0] - 2019-09-19
### Added
- Added improved logging and metrics for streams and streaming messages.
- Log level configuration can now be expressed specifically for every
  combination of inbound and outbound, for success, failure, and application
  error.
- A peer list and transport stress tester is now in the `yarpctest` package.
- Added `direct` peer chooser to enable directly addressable peers.
- Added custom dialer option for outbound HTTP requests.
- Added custom dialer option for outbound gRPC requests.

## [1.39.0] - 2019-06-25
### Fixed
- call.HeaderNames() now specifies a capacity when creating a slice,
  which should improve the call.HeaderNames()'s performance.
- Observability middleware will always emit an error code if the returned error
  is from the `yarpcerrors` package.
### Added
- Added error details support in protobuf over gRPC and HTTP.
- Protobuf JSON encoding can take a custom gogo/protobuf/jsonpb.AnyResolver with
  Fx.

## [1.38.0] - 2019-05-20
### Changed
- The Thrift encoding attempts to close the request body immediately after
  reading the request bytes. This significantly reduces TChannel/Thrift memory
  usage in some scenarios.

## [1.37.4] - 2019-05-02
### Fixed
- Fixed duplicated tracing headers being set with gRPC.

## [1.37.3] - 2019-04-29
### Fixed
- Fixed pending heap deadlock that occured when attempting to remove a peer that
  was already removed.

## [1.37.2] - 2019-04-08
### Removed
- Revert: Use separate context for grpc streams once dial has been completed.

## [1.37.1] - 2019-03-25
### Fixed
- Fix fewest pending heap panic that occurs when calling a peer and removing it
  simultaneously.

## [1.37.0] - 2019-03-14
### Fixed
- Use separate context for grpc streams once dial has been completed.

## [1.36.2] - 2019-02-25
### Fixed
- Removed error name validation.

## [1.36.1] - 2019-01-23
### Fixed
- Updated dependency on ThriftRW.

## [1.36.0] - 2019-01-23
### Added
- The log level for application errors is now configurable with yarpcconfig and
  with yarpc.NewDispatcher.

### Fixed
- Upgrade a read-lock to a read-write lock around peer selection.
  This addresses a data race observed in production that results in broken peer
  list invariants.

## [1.35.2] - 2018-11-06
### Removed
- Reverted HTTP transport marking peers as unavailable when the remote side
  closes the connection due to a deadlock.

## [1.35.1] - 2018-10-17
### Fixed
- Fixed a deadlock issue when the HTTP transport detects a connection failure
  and attempts to lock once to obtain the peer, then again to send
  notifications.

## [1.35.0] - 2018-10-15
### Added
- Added `encoding/protobuf/reflection` for exposing server reflection related
  information through codegeneration. For docs related to server reflection read
  https://github.com/grpc/grpc/blob/master/doc/server-reflection.md.
- `protoc-gen-yarpc-go` now generates `yarpcfx` fx groups containing
  information required for building server reflection API's.

### Fixed
- Using a `http.Outbound` previously leaked implementation details that it was using a
    `*http.Client` underneath, when attempting to cast a `http.Outbound` into a `http.RoundTripper`

## [1.34.0] - 2018-10-03
### Added
- Adds `thrift.Named` option for appropriately labelling procedures inherited
  from other thrift services.
- The HTTP protocol now marks peers as unavailable immediately when the remote
side closes the connection.

### Fixed
- Calling extended Thrift service procedures previously called the base service's
  procedures.

## [1.33.0] - 2018-09-26
### Added
- x/yarpctest: Add a retry option to HTTP/TChannel/GRPCRequest.
- Added `peer/tworandomchoices`, an implementation of the Two Random Choices
  load balancer algorithm.
- Reintroduce Transport field matching for `transporttest.RequestMatcher`.
### Changed
- HTTP inbounds gracefully shutdown with an optional timeout, defaulting to 5
  seconds.

## [1.32.4] - 2018-08-07
### Fixed
- Address data races in yarpctest.FakeTransport

## [1.32.3] - 2018-08-07
### Fixed
- Revert Transport field match from RequestMatcher

## [1.32.2] - 2018-08-07
### Fixed
- CHANGELOG.md and version.go changes were still incorrect for v1.32.1

## [1.32.1] - 2018-08-07
### Fixed
- CHANGELOG.md and version.go changes were incorrect for v1.32.0

## [1.32.0] - 2018-08-06
### Added
- Adds inbound and outbound TLS support for gRPC. See `gprc.InboundCredentials`,
  `grpc.DialerCredentials`, and `grpc.Transport.NewDialer` for usage.
- Added `peer/peerlist/v2` which differs from the original `peer/peerlist` by
  replacing the use of `api/peer.ListImplementation` with
  `peer/peerlist/v2.Implementation`, which threads the peer separately from the
  peer identifier.
  This allows us to thread shard information from the peer list updater to a
  sharding peer list.
- Added connection/disconnection simulation to the `yarpctest` fake transport
  and peers.
- x/yarpctest: Added support for specifying outbound middleware.
- yarpctest: Changed `FakePeer` id to use "go.uber.org/yarpc/api/peer".Identifier
  interface instead of the concrete "go.uber.org/peer/hostport".Identifier type.
### Changed
- The HTTP protocol now mitigates peers that are unavailable due to a half-open
  TCP connection.
  Previously, if a peer shut down unexpectedly, it might fail to send a TCP FIN
  packet, leaving the sender unaware that the peer is unavailable.
  The symptom is that requests sent down this connection will time out.
  This change introduces a suspicion window for peers that time out.
  Once per suspicion window, the HTTP transport's peer manager will attempt
  to establish a fresh TCP connection to the peer.
  Failing to establish a connection will transition the peer to the unavailable
  state until a fresh TCP connection becomes available.
  The HTTP transport now accepts an `InnocenceWindow` duration, and an
  `innocenceWindow` config field.

## [1.31.0] - 2018-07-09
### Added
- Added `Outbounds()` on `Dispatcher` to provide access to the configured outbounds.
- Expose capacity option to configurator for the round-robin peer chooser.
- Expose capacity option to configurator for the fewest pending heap peer chooser.
- Dispatchers now log recovered handler panics via a Zap logger, if present.
- Responses for all transports now include a header containing the name of the
  service that handled the request.

### Changed
- TChannel inbounds will blackhole requests when handlers return resource
  exhausted errors.
- Change log level to reflect error statuses. Previously all logs were logged at
  debug level. Errors are now logged at error level.
- Update pin for gogo/protobuf to ^1

## [1.30.0] - 2018-05-03
### Added
- The YARPC HTTP outbound now implements http.RoundTripper.
  This makes YARPC's load balancers, other peer selectors, and peer
  availability management accessible to REST call sites.
- Adds `Addr()` on `grpc.Inbound` to expose the address the server is listening
  on when the server is running.
- Adds Transport property to transport.Request and transport.RequestMeta
  Metrics will now be tagged with the transport of calls to handlers
### Fixed
- YARPC HTTP, gRPC, and TChannel transports are now compatible with any
  peer.Identifier implementation.
  They previously required a hostport.PeerIdentifier for RetainPeer and
  ReleasePeer calls.

## [1.29.1] - 2018-04-04
### Fixed
- Removed `repo:` from glide.yaml because Apache Thrift development has moved
  to GitHub (https://issues.apache.org/jira/browse/INFRA-16287).

## [1.29.0] - 2018-03-21
### Added
- Add methods to start and stop a dispatcher's transports, inbounds, and
  outbounds separately.
- Add `NewFx{{Service}}YARPCClient` and `NewFx{{Service}}YARPCProcedures`
  generated methods from protoc-gen-yarpc-go for Fx.

## [1.28.0] - 2018-03-13
### Changed
- Enabled random shuffling of peerlist order by default.
### Added
- Reintroduce envelope-agnostic Thrift inbounds. Thrift inbounds will now
  accept Thrift requests with or without envelopes.  This feature was
  originally added in 1.26.0 and removed in 1.26.1 because the implementation
  introduced an inbound request data corruption hazard.
- Adds an option to the TChannel transport to carry headers in their original
  form, instead of normalizing their case.
- Adds an option to disable observability middleware, in the event you
  provide alternate observability middleware.

## [1.27.2] - 2017-01-23
### Fixed
- Removed buffer pooling from GRPC outbound requests which had possible data
  corruption issues.

## [1.27.1] - 2017-01-22
### Changed
- Regenerate thrift files.

## [1.27.0] - 2017-01-22
### Added
- Add support for Inbound and Outbound streaming RPCs using gRPC and Protobuf.
- Add support for creating peer choosers through config with PeerChooserSpec.
- Add the option of injecting a `"go.uber.org/net/metrics".Scope` into the
  dispatcher metrics configuration, in lieu of a Tally Scope.  Metrics scopes
  support in memory and Prometheus collection.

### Changed
- Detect buffer pooling bugs by detecting concurrent accesses in production
  and more thorough use-after-free detection in tests.

### Fixed
- Removed buffer pooling from GRPC inbound responses which had possible data
  corruption issues.
- TChannel inbound response errors are now properly mapped from YARPC errors.


## [1.26.2] - 2017-01-17
### Removed
- Removed buffer pooling from GRPC inbound responses which had possible data
  corruption issues.


## [1.26.1] - 2017-12-21
### Removed
- Reverts the integration for envelope-agnostic Thrift. This change
  introduced data corruption to request bodies due to a buffer pooling bug.


## [1.26.0] - 2017-12-13
### Added
- Support envelope-agnostic Thrift inbounds. Thrift inbounds will now accept
  Thrift requests with or without envelopes.

### Changed
- Wrap errors returned from lifecycle.Once functions in the yarpcerrors API.
- Wrap errors returned from tchannel outbounds in the yarpcerrors API.


## [1.25.1] - 2017-12-05
### Changed
- Revert Providing a better error message if outbound calls are made or inbound calls
  are received before Dispatcher start or after Dispatcher stop.


## [1.25.0] - 2017-12-04
### Added
- Make Dispatcher start/stop thread-safe.

### Changed
- Validate all oneway calls have a TTL.
- Add opentracing tags to denote the YARPC version and Golang version to the
  gRPC and HTTP transports.
- Provide a better error message if outbound calls are made or inbound calls
  are received before Dispatcher start or after Dispatcher stop.


## [1.24.1] - 2017-11-27

- Undeprecate ClientConfig function.


## [1.24.0] - 2017-11-22

- Introduces `api/peer.ListImplementation` with `peer/peerlist.List`, a
  building block that provides peer availability management for peer lists
  like round-robin, hash-ring.
- Adds /x/yarpctest infrastructure to create fake services and requests for
  tests.
- Adds a `peer/pendingheap` implementation that performs peer selection,
  sending requests to the available peer with the fewest pending requests.
- Adds `OutboundConfig` and `MustOutboundConfig` functions to the dispatcher
  to replace the ClientConfig function.


## [1.22.0] - 2017-11-14

- Thrift: Fx modules generated by the ThriftRW plugin now include a function
  to register procedures with YARPC.


## [1.21.1] - 2017-11-13

- Fix a bug in protoc-gen-yarpc-go where request or response types for
  methods in the same package but in a different file would result in an
  extraneous import.


## [1.21.0] - 2017-10-26

- Add a Logger option to the HTTP, GRPC, and TChannel transports to allow for
  internal logging.


## [1.20.1] - 2017-10-23

- http: Fix `http.Interceptor` ignoring `http.Mux`.


## [1.20.0] - 2017-10-16

- http: Add `http.Interceptor` option to inbounds, which allows intercepting
  incoming HTTP requests for that inbound.


## [1.19.2] - 2017-10-10

- transport/grpc: Fix deadlock where Peers can never be stopped if their
  corresponding Transport was not started.


## [1.19.1] - 2017-10-10

- transport/grpc: Add Chooser function to Outbound for testing.


## [1.19.0] - 2017-10-10

- Promote `transport/x/grpc` out of experimental status, moving it to
  `transport/grpc`.


## [1.18.1] - 2017-10-04

- Remove staticcheck from glide.yaml.


## [1.18.0] - 2017-09-26

- Add inbound/outbound direction tag to observability metrics.
- Remove x/retry to incubate internally.
- yarpcerrors: Undeprecate per error type creation and validation functions.


## [1.17.0] - 2017-09-20

- yarpcerrors: Make core API much simpler and use a Status struct
  to represent YARPC errors.
- transport/http: Add GrabHeaders option to propagate specific
  headers starting with x- to handlers.
- tranxport/x/grpc: Remove ContextWrapper.
- Export no-op backoff strategy in api/backoff.


## [1.16.0] - 2017-09-18

- ThriftRW Plugin: Added an option to strip TChannel-specific
  information from Contexts before making outgoing requests.
- x/retry: Fix bug where large TChannel responses would cause errors in retries.
- transport/http: Correct the Content-Type for Thrift responses, to
  `application/vnd.apache.thrift.binary`.
- transport/http: Correct the Content-Type for Proto responses, to
  `application/x-protobuf`.


## [1.15.0] - 2017-09-15

- yarpcerrors: Update the ErrorCode and ErrorMessage functions to return
  default values for non-YARPC errors.
- transport/http: Return appropriate Content-Type response headers based on
  the transport encoding.
- transport/x/grpc: Add options to specify the maximum message size sent and
  received over the wire.


## [1.14.0] - 2017-09-08

- Increased granularity of error observability metrics to expose yarpc
  error types.
- Wrapped peer list `Choose` errors in yarpc error codes.
- x/retry: Add granular metric counters for retry middleware.
- Removed experimental redis and cherami transports.


## [1.13.1] - 2017-08-03

- Rename structured logging field to avoid a type collision.


## [1.13.0] - 2017-08-01

- Added a `yarpc.ClientConfig` interface to provide access to ClientConfigs.
  All Dispatchers already implement this interface.
- Thrift: Fx modules generated by the ThriftRW plugin now rely on
  `yarpc.ClientConfig` instead of the Dispatcher.
- Promote `x/config` out of experimental status, moving it to `yarpcconfig`.


## [1.12.1] - 2017-07-26

- Fixed issue with github.com/apache/thrift by pinning to version 0.9.3
  due to breaking change https://issues.apache.org/jira/browse/THRIFT-4261.


## [1.12.0] - 2017-07-20

Experimental:

- x/debug: Added support for debug pages for introspection.


## [1.11.0] - 2017-07-18

- Fixed bug where outbound HTTP errors were not being properly wrapped in
  yarpc errors.

Experimental:

- x/retry: Added support for procedure-based retry policies.
- x/retry: Fixed bug in retry middleware where a failed request that did not
  read the request body would not be retried.
- x/grpc: Altered inbound GRPC code in order to support RouterMiddleware.


## [1.10.0] - 2017-17-11

- Thrift: UberFx-compatible modules are now generated for each service inside
  a subpackage. Disable this by passing a `-no-fx` flag to the plugin.
- Improves resilience of HTTP and TChannel by broadcasting peer availability
  to peer lists like round robin. A round robin or least pending peer list
  blocks requests until a peer becomes available.
- Add support for reading ShardKey, RoutingKey, and RoutingDelegate for
  inbound http and tchannel calls.
- Move encoding/x/protobuf to encoding/protobuf.
- Exposes the Lifecycle synchronization helper as pkg/lifecycle, for
  third-party implementations of transports, inbounds, outbounds, peer lists,
  and peer list bindings.

Experimental:

- x/retry: Added support for creating retry middleware directly from config.


## [1.9.0] - 2017-06-08

- Different encodings can now register handlers under the same procedure
  name.
- http: Added support for configuring the HTTP transport using x/config.
- tchannel: Added support for configuring the TChannel transport using
  x/config.
- Moved the RoundRobin Peer List out of the /x/ package.
- Fixed race conditions in hostport.Peer.
- Buffers for inbound Thrift requests are now pooled to reduce allocations.

Experimental:

- x/cherami: Renamed the `InboundConfig` and `OutboundConfig` structures to
  `InboundOptions` and `OutboundOptions`.
- x/cherami: Added support for configuring the Cherami transport using
  x/config.
- x/roundrobin: Added support for taking peer list updates before and after
  the peer list has been started.
- x/config: Fix bug where embedded struct fields could not be interpolated.
- x/config: Fix bug where Chooser and Updater fields could not be interpolated.
- x/grpc: Remove `NewInbound` and `NewSingleOutbound` in favor of methods on
  `Transport`.
- x/grpc: Use `rpc-caller`, `rpc-service`, `rpc-encoding`, `rpc-shard-key`,
  `rpc-routing-key`, `rpc-routing-delegate` headers.
- x/protobuf: Handle JSON-encoded protobuf requests and return JSON-encoded
  protobuf responses if the `rpc-encoding` header is set to `json`. Protobuf
  clients may use JSON by supplying the `protobuf.UseJSON` option.
- x/protobuf: Support instantiating clients with `yarpc.InjectClients`.
- x/protobuf: The wire representation of request metadata was changed. This
  will break existing users of this encoding.


## [1.8.0] - 2017-05-01

- Adds consistent structured logging and metrics to all RPCs. This feature
  may be enabled and configured through `yarpc.Config`.
- Adds an `http.AddHeader` option to HTTP outbounds to send certain HTTP
  headers for all requests.
- Options `thrift.Multiplexed` and `thrift.Enveloped` may now be provided for
  Thrift clients constructed by `yarpc.InjectClients` by adding a `thrift`
  tag to the corresponding struct field with the name of the option. See the
  Thrift package documentation for more details.
- Adds support for matching and constructing `UnrecognizedProcedureError`s
  indicating that the router was unable to find a handler for the request.
- Adds support for linking peer lists and peer updaters using the `peer.Bind`
  function.
- Adds an accessor to Dispatcher which provides access to the inbound
  middleware used by that Dispatcher.
- Fixes a bug where the TChannel inbounds would not write the response headers
  if the response body was empty.

Experimental:

- x/config: The service name is no longer part of the configuration and must
  be passed as an argument to the `LoadConfig*` or `NewDispatcher*` methods.
- x/config: Configuration structures may now annotate primitive fields with
  `config:",interpolate"` to support reading environment variables in them.
  See the `TransportSpec` documentation for more information.


## [1.7.1] - 2017-03-29

- Thrift: Fixed a bug where deserialization of large lists would return
  corrupted data at high throughputs.


## [1.7.0] - 2017-03-20

- x/config adds support for a pluggable configuration system that allows
  building `yarpc.Config` and `yarpc.Dispatcher` objects from YAML and
  arbitrary `map[string]interface{}` objects. Check the package documentation
  for more information.
-	tchannel: mask existing procedures with provided procedures.
- Adds a peer.Bind function that takes a peer.ChooserList and a binder
  (anything that binds a peer list to a peer provider and returns the
  Lifecycle of the binding), and returns a peer.Chooser that combines
  the lifecycle of the peer list and its bound peer provider.
  The peer chooser is suitable for passing to an outbound constructor,
  capturing the lifecycle of its dependencies.
- Adds a peer.ChooserList interface to the API, for convenience when passing
  instances with both capabilities (suitable for outbounds, suitable for peer
  list updaters).


## [1.6.0] - 2017-03-08

- Remove buffer size limit from Thrift encoding/decoding buffer pool.
- Increased efficiency of inbound/outbound requests by pooling buffers.
- Added MaxIdleConnsPerHost option to HTTP transports.  This option will
  configure the number of idle (keep-alive) outbound connections the transport
  will maintain per host.
- Fixed bug in Lifecycle Start/Stop where we would run the Stop functionality
  even if Start hadn't been called yet.
- Updated RoundRobin and PeerHeap implementations to block until the list has
  started or a timeout had been exceeded.


## [1.5.0] - 2017-03-03

- Increased efficiency of Thrift encoding/decoding by pooling buffers.
- x/yarpcmeta make it easy to expose the list of procedures and other
  introspection information of a dispatcher on itself.
- Redis: `Client` now has an `IsRunning` function to match the `Lifecycle`
  interface.
- TChannel: bug fix that allows a YARPC proxy to relay requests for any
  inbound service name. Requires upgrade of TChannel to version 1.4 or
  greater.


## [1.4.0] - 2017-02-14

- Relaxed version constraint for `jaeger-client-go` to `>= 1, < 3`.
- TChannel transport now supports procedures with a different service name
  than the default taken from the dispatcher. This brings the TChannel
	transport up to par with HTTP.


## [1.3.0] - 2017-02-06

- Added a `tchannel.NewTransport`. The new transport, a replacement for the
  temporary `tchannel.NewChannelTransport`, supports YARPC peer choosers.

    ```go
    transport, err := tchannel.NewTransport(tchannel.ServiceName("keyvalue"))
    chooser := peerheap.New(transport)
    outbound := transport.NewOutbound(chooser)
    ```

  The new transport hides the implementation of TChannel entirely to give us
  flexibility going forward to relieve TChannel of all RPC-related
  responsibilities, leaving only the wire protocol at its core.
  As a consequence, you cannot thread an existing Channel through this
  transport.

- All outbounds now support `Call` before `Start` and all peer choosers now
  support `Choose` before `Start`, within the context deadline.
  These would previously return an error indicating that the component was
  not yet started.  They now wait for the component to start, or for their
  deadline to expire.


## [1.2.0] - 2017-02-02

- Added heap based PeerList under `peer/x/peerheap`.
- Added `RouterMiddleware` parameter to `yarpc.Config`, which, if provided,
  will allow customizing routing to handlers.
- Added experimental `transports/x/cherami` for transporting RPCs through
  [Cherami](https://eng.uber.com/cherami/).
- Added ability to specify a ServiceName for outbounds on the
  transport.Outbounds object.  This will allow defining outbounds with a
  `key` that is different from the service name they will use for requests.
  If no ServiceName is specified, the ServiceName will fallback to the
  config.Outbounds map `key`.

  Before:

    ```go
    config.Outbounds['service'] := transport.Outbounds{
        Unary: httpTransport.NewSingleOutbound(...)
    }
    ...
    cc := dispatcher.ClientConfig('service')
    cc.Service() // returns 'service'
    ```

  After (optional):

    ```go
    config.Outbounds['service-key'] := transport.Outbounds{
        ServiceName: 'service'
        Unary: httpTransport.NewSingleOutbound(...)
    }
    ...
    cc := dispatcher.ClientConfig('service-key')
    cc.Service() // returns 'service'
    ```


## [1.1.0] - 2017-01-24

- Thrift: Mock clients compatible with gomock are now generated for each
  service inside a test subpackage. Disable this by passing a `-no-gomock`
  flag to the plugin.


## [1.0.1] - 2017-01-11

- Thrift: Fixed code generation for empty services.
- Thrift: Fixed code generation for Thrift services that inherit other Thrift
  services.


## [1.0.0] - 2016-12-30

- Stable release: No more breaking changes will be made in the 1.x release
  series.


## [1.0.0-rc5] - 2016-12-30

- **Breaking**: The ThriftRW plugin now generates code under the subpackages
  `${service}server` and `$[service}client` rather than
  `yarpc/${service}server` and `yarpc/${service}client`.

  Given a `kv.thrift` that defines a `KeyValue` service, previously the
  imports would be,

      import ".../kv/yarpc/keyvalueserver"
      import ".../kv/yarpc/keyvalueclient"

  The same packages will now be available at,

      import ".../kv/keyvalueserver"
      import ".../kv/keyvalueclient"

- **Breaking**: `NewChannelTransport` can now return an error upon
  construction.
- **Breaking**: `http.URLTemplate` has no effect on `http.NewSingleOutbound`.
- `http.Transport.NewOutbound` now accepts `http.OutboundOption`s.


## [1.0.0-rc4] - 2016-12-28

- **Breaking**: Removed the `yarpc.ReqMeta` and `yarpc.ResMeta` types. To
  migrate your handlers, simply drop the argument and the return value from
  your handler definition.

  Before:

    ```go
    func (h *myHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, ...) (..., yarpc.ResMeta, error) {
        // ...
    }
    ```

  After:

    ```go
    func (h *myHandler) Handle(ctx context.Context, ...) (..., error) {
        // ...
    }
    ```

  To access information previously available in the `yarpc.ReqMeta` or to
  write response headers, use the `yarpc.CallFromContext` function.

- **Breaking**: Removed the `yarpc.CallReqMeta` and `yarpc.CallResMeta`
  types. To migrate your call sites, drop the argument and remove the return
  value.

  Before:

    ```go
    res, resMeta, err := client.Call(ctx, reqMeta, ...)
    ```

  After:

    ```go
    res, err := client.Call(ctx, ...)
    ```

  Use `yarpc.CallOption`s to specify per-request options and
  `yarpc.ResponseHeaders` to receive response headers for the call.

- **Breaking**: Removed `yarpc.Headers` in favor of `map[string]string`.
- **Breaking**: `yarpc.Dispatcher` no longer implements the
  `transport.Router` interface.
- **Breaking**: Start and Stop for Inbound and Outbound are now expected to
  be idempotent.
- **Breaking**: Combine `ServiceProcedure` and `Registrant` into `Procedure`.
- **Breaking**: Rename `Registrar` to `RouteTable`.
- **Breaking**: Rename `Registry` to `Router`.
- **Breaking**: Rename `middleware.{Oneway,Unary}{Inbound,Outbound}Middleware`
  to `middleware.{Oneway,Unary}{Inbound,Outbound}`
- **Breaking**: Changed `peer.List.Update` to accept a `peer.ListUpdates`
  struct instead of a list of additions and removals
- **Breaking**: yarpc.NewDispatcher now returns a pointer to a
  yarpc.Dispatcher. Previously, yarpc.Dispatcher was an interface, now a
  concrete struct.

  This change will allow us to extend the Dispatcher after the 1.0.0 release
  without breaking tests depending on the rigidity of the Dispatcher
  interface.
- **Breaking**: `Peer.StartRequest` and `Peer.EndRequest` no longer accept a
  `dontNotify` argument.
- Added `yarpc.IsBadRequestError`, `yarpc.IsUnexpectedError` and
  `yarpc.IsTimeoutError` functions.
- Added a `transport.InboundBadRequestError` function to build errors which
  satisfy `transport.IsBadRequestError`.
- Added a `transport.ValidateRequest` function to validate
  `transport.Request`s.


## [1.0.0-rc3] - 2016-12-09

- Moved the `yarpc/internal/crossdock/` and `yarpc/internal/examples`
  folders to `yarpc/crossdock/` and `yarpc/examples` respectively.

- **Breaking**: Relocated the `go.uber.org/yarpc/transport` package to
  `go.uber.org/yarpc/api/transport`.  In the process the `middleware`
  logic from transport has been moved to `go.uber.org/yarpc/api/middleware`
  and the concrete implementation of the Registry has been moved from
  `transport.MapRegistry` to `yarpc.MapRegistry`.  This did **not** move the
  concrete implementations of http/tchannel from the `yarpc/transport/` directory.

- **Breaking**: Relocated the `go.uber.org/yarpc/peer` package to
  `go.uber.org/yarpc/api/peer`. This does not include the concrete
  implementations still in the `/yarpc/peer/` directory.

- **Breaking**: This version overhauls the code required for constructing
  inbounds and outbounds.

  Inbounds and Outbounds now share an underlying Transport, of which there
  should be one for each transport protocol, so one HTTP Transport for all
  HTTP inbounds and outbounds, and a TChannel transport for all TChannel
  inbounds and outbounds.

  Before:

    ```go
    ch, err := tchannelProper.NewChannel("example-service", nil)
    if err != nil {
        log.Fatalln(err)
    }
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            http.NewInbound(":80"),
            tchannel.NewInbound(ch, tchannel.ListenAddr(":4040")),
        },
        yarpc.Outbounds{
            http.NewOutbound("http://example-service/rpc/v1"),
            tchannel.NewOutbound(ch, tchannel.HostPort("127.0.0.1:4040")),
        },
    })
    ```

  After:

    ```go
    httpTransport := http.NewTransport()
    tchannelTransport := tchannel.NewChannelTransport(
          tchannel.ServiceName("example-service"),
          tchannel.ListenAddr(":4040"),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            httpTransport.NewInbound(":80"),
            tchannelTransport.NewInbound(),
        },
        yarpc.Outbounds{
            httpTransport.NewSingleOutbound("http://example-service/rpc/v1"),
            tchannelTransport.NewSingleOutbound("127.0.0.1:4040"),
        },
    })
    ```

  The dispatcher now collects all of the unique transport instances from
  inbounds and outbounds and manages their lifecycle independently.

  This version repurposed the name `NewOutbound` for outbounds with a peer
  chooser, whereas `NewSingleOutbound` is a convenience for creating an
  outbound addressing a specific single peer.
  You may need to rename existing usage. The compiler will complain that
  strings are not `peer.Chooser` instances.

  This version introduces support for peer choosers, peer lists, and peer
  list updaters for HTTP outbounds. This is made possible by the above
  change that introduces a concrete instance of a Transport for each
  protocol, which deduplicates peer instances across all inbounds and
  outbounds, making connection sharing and load balancing possible,
  eventually for all transport protocols.

  Note that we use `NewChannelTransport`, as opposed to `NewTransport`.
  We reserve this name for a future minor release that will provide
  parity with HTTP for outbounds with peer choosers.

  The new ChannelTransport constructor can still use a shared TChannel
  Channel instance, if that is required.

    ```go
    ch, err := tchannelProper.NewChannel("example-service", nil)
    if err != nil {
        log.Fatalln(err)
    }
    tchannelTransport := tchannel.NewChannelTransport(
        tchannel.WithChannel(ch),
          tchannel.ServiceName("example-service"),
          tchannel.ListenAddr(":4040"),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            tchannelTransport.NewInbound(),
        },
    })
    ```

- **Breaking**: the `transport.Inbound` and `transport.Outbound` interfaces
  now implement `Start()` without any arguments.

  The dispatcher no longer threads a dependencies object through the start
  method of every configured transport. The only existing dependency was an
  opentracing Tracer, which you can now thread through Transport constructor
  options instead.

  Before:

    ```go
    yarpc.NewDispatcher(yarpc.Config{
        yarpc.Inbounds{
            http.NewInbound(...),
        },
        yarpc.Outbounds{
            "callee": http.NewOutbound(...)
        },
        Tracer: opentracing.GlobalTracer(),
    })
    ```

  Now:

    ```go
    tracer := opentracing.GlobalTracer()
    httpTransport := http.NewTransport(
        http.Tracer(tracer),
    )
    tchannelTransport := tchannel.NewChannelTransport(
        tchannel.Tracer(tracer),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            httpTransport.NewInbound(":80"),
            tchannelTransport.NewInbound(),
        },
        yarpc.Outbounds{
            httpTransport.NewSingleOutbound("http://example-service/rpc/v1"),
            tchannelTransport.NewSingleOutbound("127.0.0.1:4040"),
        },
    })
    ```

  The `yarpc.Config` `Tracer` property is still accepted, but unused and
  deprecated.

  The dispatcher no longer provides a `transport.ServiceDetail` as an
  argument to `Start` on inbound transports.  The `transport.ServiceDetail`
  no longer exists.  You no longer need to provide the service name to start
  an inbound, only a registry.  Instead of passing the service detail to start,
  the dispatcher now calls `inbound.SetRegistry(transport.Registry)` before
  calling `Start()`.

  Custom transport protocols must change their interface accordingly to
  satisfy the `transport.Inbound` interface.  Uses that construct inbounds
  manually must either call `SetRegistry` or use the `WithRegistry` chained
  configuration method before calling `Start` without a `ServiceDetail`.

  Before:

    ```go
    inbound := tchannel.NewInbound(...)
    err := inbound.Start(
        transport.ServiceDetail{
            Name: "service",
            Registry: registry,
        },
        transport.NoDeps,
    )
    ```

  Now:

    ```go
    transport := tchannel.NewTransport()
    inbound := transport.NewInbound()
    inbound.SetRegistry(registry)
    err := inbound.Start()
    ```

  The `transport.Deps` struct and `transport.NoDeps` instance no longer exist.

- **Breaking**: TChannel inbound and outbound constructors now return
  pointers to Inbound and Outbound structs with private state satisfying the
  `transport.Inbound` and `transport.Outbound` interfaces.  These were
  previously transport specific Inbound and Outbound interfaces.
  This eliminates unnecessary polymorphism in some cases.

- Introduced OpenTracing helpers for transport authors.
- Created the `yarpc.Serialize` package for marshalling RPC messages at rest.
  Useful for transports that persist RPC messages.
- Tranports have access to `DispatchOnewayHandler` and `DispatchUnaryHandler`.
  These should be called by all `transport.Inbounds` instead of directly
  calling handlers.

## [1.0.0-rc2] - 2016-12-02

- **Breaking** Renamed `Agent` to `Transport`.
- **Breaking** Renamed `hostport.Peer`'s `AddSubscriber/RemoveSubscriber`
  to `Subscribe/Unsubscribe`.
- **Breaking** Updated `Peer.StartRequest` to take a `dontNotify` `peer.Subscriber` to exempt
  from updates.  Also added `Peer.EndRequest` function to replace the `finish` callback
  from `Peer.StartRequest`.
- **Breaking** Renamed `peer.List` to `peer.Chooser`, `peer.ChangeListener` to `peer.List`
  and `peer.Chooser.ChoosePeer` to `peer.Chooser.Choose`.
- Reduced complexity of `single` `peer.Chooser` to retain the passed in peer immediately.
- **Breaking** Moved `/peer/list/single.go` to `/peer/single/list.go`.
- **Breaking** Moved `/peer/x/list/roundrobin.go` to `/peer/x/roundrobin/list.go`.
- HTTP Oneway requests will now process http status codes and returns appropriate errors.
- **Breaking** Update `roundrobin.New` function to stop accepting an initial peer list.
  Use `list.Update` to initialize the peers in the list instead.
- **Breaking**: Rename `Channel` to `ClientConfig` for both the dispatcher
  method and the interface. `mydispatcher.Channel("myservice")` becomes
  `mydispatcher.ClientConfig("myservice")`. The `ClientConfig` object can
  then used to build a new Client as before:
  `NewMyThriftClient(mydispatcher.ClientConfig("myservice"))`.
- A comment is added atop YAML files generated by the recorder to help
  understanding where they come from.

## [1.0.0-rc1] - 2016-11-23

- **Breaking**: Rename the `Interceptor` and `Filter` types to
  `UnaryInboundMiddleware` and `UnaryOutboundMiddleware` respectively.
- **Breaking**: `yarpc.Config` now accepts middleware using the
  `InboundMiddleware` and `OutboundMiddleware` fields.

  Before:

      yarpc.Config{Interceptor: myInterceptor, Filter: myFilter}

  Now:

      yarpc.Config{
          InboundMiddleware: yarpc.InboundMiddleware{Unary: myInterceptor},
          OutboundMiddleware: yarpc.OutboundMiddleware{Unary: myFilter},
      }

- Add support for Oneway middleware via the `OnewayInboundMiddleware` and
  `OnewayOutboundMiddleware` interfaces.


## [0.5.0] - 2016-11-21

- **Breaking**: A detail of inbound transports has changed.
  Starting an inbound transport accepts a ServiceDetail, including
  the service name and a Registry. The Registry now must
  implement `Choose(context.Context, transport.Request) (HandlerSpec, error)`
  instead of `GetHandler(service, procedure string) (HandlerSpec, error)`.
  Note that in the prior release, `Handler` became `HandleSpec` to
  accommodate oneway handlers.
- Upgrade to ThriftRW 1.0.
- TChannel: `NewInbound` and `NewOutbound` now accept any object satisfying
  the `Channel` interface. This should work with existing `*tchannel.Channel`
  objects without any changes.
- Introduced `yarpc.Inbounds` to be used instead of `[]transport.Inbound`
  when configuring a Dispatcher.
- Add support for peer lists in HTTP outbounds.


## [0.4.0] - 2016-11-11

This release requires regeneration of ThriftRW code.

- **Breaking**: Procedure registration must now always be done directly
  against the `Dispatcher`. Encoding-specific functions `json.Register`,
  `raw.Register`, and `thrift.Register` have been deprecated in favor of
  the `Dispatcher.Register` method. Existing code may be migrated by running
  the following commands on your go files.

    ```
    gofmt -w -r 'raw.Register(d, h) -> d.Register(h)' $file.go
    gofmt -w -r 'json.Register(d, h) -> d.Register(h)' $file.go
    gofmt -w -r 'thrift.Register(d, h) -> d.Register(h)' $file.go
    ```

- Add `yarpc.InjectClients` to automatically instantiate and inject clients
  into structs that need them.
- Thrift: Add a `Protocol` option to change the Thrift protocol used by
  clients and servers.
- **Breaking**: Remove the ability to set Baggage Headers through yarpc, use
  opentracing baggage instead
- **Breaking**: Transport options have been removed completely. Encoding
  values differently based on the transport is no longer supported.
- **Breaking**: Thrift requests and responses are no longer enveloped by
  default. The `thrift.Enveloped` option may be used to turn enveloping on
  when instantiating Thrift clients or registering handlers.
- **Breaking**: Use of `golang.org/x/net/context` has been dropped in favor
  of the standard library's `context` package.
- Add support for providing peer lists to dynamically choose downstream
  peers in HTTP Outbounds
- Rename `Handler` interface to `UnaryHandler` and separate `Outbound`
  interface into `Outbound` and `UnaryOutbound`.
- Add `OnewayHandler` and `HandlerSpec` to support oneway handlers.
  Transport inbounds can choose which RPC types to accept
- The package `yarpctest.recorder` can be used to record/replay requests
  during testing. A command line flag (`--recorder=replay|append|overwrite`)
  is used to control the mode during the execution of the test.


## [0.3.1] - 2016-09-31

- Fix missing canonical import path to `go.uber.org/yarpc`.


## [0.3.0] - 2016-09-30

- **Breaking**: Rename project to `go.uber.org/yarpc`.
- **Breaking**: Switch to `go.uber.org/thriftrw ~0.3` from
  `github.com/thriftrw/thriftrw-go ~0.2`.
- Update opentracing-go to `>= 1, < 2`.


## [0.2.1] - 2016-09-28

- Loosen constraint on `opentracing-go` to `>= 0.9, < 2`.


## [0.2.0] - 2016-09-19

- Update thriftrw-go to `>= 0.2, < 0.3`.
- Implemented a ThriftRW plugin. This should now be used instead of the
  ThriftRW `--yarpc` flag. Check the documentation of the
  [thrift](https://godoc.org/github.com/yarpc/yarpc-go/encoding/thrift)
  package for instructions on how to use it.
- Adds support for [opentracing][]. Pass an opentracing instance as a
  `Tracer` property of the YARPC config struct and both TChannel and HTTP
  transports will submit spans and propagate baggage.
- This also modifies the public interface for transport inbounds and
  outbounds, which must now accept a transport.Deps struct. The deps struct
  carries the tracer and may eventually carry other dependencies.
- Panics from user handlers are recovered. The panic is logged (stderr), and
  an unexpected error is returned to the client about it.
- Thrift clients can now make requests to multiplexed Apache Thrift servers
  using the `thrift.Multiplexed` client option.

[opentracing]: http://opentracing.io/


## [0.1.1] - 2016-09-01

- Use `github.com/yarpc/yarpc-go` as the import path; revert use of
  `go.uber.org/yarpc` vanity path. There is an issue in Glide `0.11` which
  causes installing these packages to fail, and thriftrw `~0.1`'s yarpc
  template is still using `github.com/yarpc/yarpc-go`.


## 0.1.0 - 2016-08-31

- Initial release.

[Unreleased]: https://github.com/yarpc/yarpc-go/compare/v1.42.1...HEAD
[1.42.1]: https://github.com/yarpc/yarpc-go/compare/v1.42.0...v1.42.1
[1.42.0]: https://github.com/yarpc/yarpc-go/compare/v1.41.0...v1.42.0
[1.41.0]: https://github.com/yarpc/yarpc-go/compare/v1.40.0...v1.41.0
[1.40.0]: https://github.com/yarpc/yarpc-go/compare/v1.39.0...v1.40.0
[1.39.0]: https://github.com/yarpc/yarpc-go/compare/v1.38.0...v1.39.0
[1.38.0]: https://github.com/yarpc/yarpc-go/compare/v1.37.4...v1.38.0
[1.37.4]: https://github.com/yarpc/yarpc-go/compare/v1.37.3...v1.37.4
[1.37.3]: https://github.com/yarpc/yarpc-go/compare/v1.37.2...v1.37.3
[1.37.2]: https://github.com/yarpc/yarpc-go/compare/v1.37.1...v1.37.2
[1.37.1]: https://github.com/yarpc/yarpc-go/compare/v1.37.0...v1.37.1
[1.37.0]: https://github.com/yarpc/yarpc-go/compare/v1.36.2...v1.37.0
[1.36.2]: https://github.com/yarpc/yarpc-go/compare/v1.36.1...v1.36.2
[1.36.1]: https://github.com/yarpc/yarpc-go/compare/v1.36.0...v1.36.1
[1.36.0]: https://github.com/yarpc/yarpc-go/compare/v1.35.2...v1.36.0
[1.35.2]: https://github.com/yarpc/yarpc-go/compare/v1.35.1...v1.35.2
[1.35.1]: https://github.com/yarpc/yarpc-go/compare/v1.35.0...v1.35.1
[1.35.0]: https://github.com/yarpc/yarpc-go/compare/v1.34.0...v1.35.0
[1.34.0]: https://github.com/yarpc/yarpc-go/compare/v1.33.0...v1.34.0
[1.33.0]: https://github.com/yarpc/yarpc-go/compare/v1.32.4...v1.33.0
[1.32.4]: https://github.com/yarpc/yarpc-go/compare/v1.32.3...v1.32.4
[1.32.3]: https://github.com/yarpc/yarpc-go/compare/v1.32.2...v1.32.3
[1.32.2]: https://github.com/yarpc/yarpc-go/compare/v1.32.1...v1.32.2
[1.32.1]: https://github.com/yarpc/yarpc-go/compare/v1.32.0...v1.32.1
[1.32.0]: https://github.com/yarpc/yarpc-go/compare/v1.31.0...v1.32.0
[1.31.0]: https://github.com/yarpc/yarpc-go/compare/v1.30.0...v1.31.0
[1.30.0]: https://github.com/yarpc/yarpc-go/compare/v1.29.1...v1.30.0
[1.29.1]: https://github.com/yarpc/yarpc-go/compare/v1.29.0...v1.29.1
[1.29.0]: https://github.com/yarpc/yarpc-go/compare/v1.28.0...v1.29.0
[1.28.0]: https://github.com/yarpc/yarpc-go/compare/v1.27.2...v1.28.0
[1.27.2]: https://github.com/yarpc/yarpc-go/compare/v1.27.1...v1.27.2
[1.27.1]: https://github.com/yarpc/yarpc-go/compare/v1.27.0...v1.27.1
[1.27.0]: https://github.com/yarpc/yarpc-go/compare/v1.26.2...v1.27.0
[1.26.2]: https://github.com/yarpc/yarpc-go/compare/v1.26.1...v1.26.2
[1.26.1]: https://github.com/yarpc/yarpc-go/compare/v1.26.0...v1.26.1
[1.26.0]: https://github.com/yarpc/yarpc-go/compare/v1.25.1...v1.26.0
[1.25.1]: https://github.com/yarpc/yarpc-go/compare/v1.25.0...v1.25.1
[1.25.0]: https://github.com/yarpc/yarpc-go/compare/v1.24.1...v1.25.0
[1.24.1]: https://github.com/yarpc/yarpc-go/compare/v1.24.0...v1.24.1
[1.24.0]: https://github.com/yarpc/yarpc-go/compare/v1.22.0...v1.24.0
[1.22.0]: https://github.com/yarpc/yarpc-go/compare/v1.21.1...v1.22.0
[1.21.1]: https://github.com/yarpc/yarpc-go/compare/v1.21.0...v1.21.1
[1.21.0]: https://github.com/yarpc/yarpc-go/compare/v1.20.1...v1.21.0
[1.20.1]: https://github.com/yarpc/yarpc-go/compare/v1.20.0...v1.20.1
[1.20.0]: https://github.com/yarpc/yarpc-go/compare/v1.19.2...v1.20.0
[1.19.2]: https://github.com/yarpc/yarpc-go/compare/v1.19.1...v1.19.2
[1.19.1]: https://github.com/yarpc/yarpc-go/compare/v1.19.0...v1.19.1
[1.19.0]: https://github.com/yarpc/yarpc-go/compare/v1.18.1...v1.19.0
[1.18.1]: https://github.com/yarpc/yarpc-go/compare/v1.18.0...v1.18.1
[1.18.0]: https://github.com/yarpc/yarpc-go/compare/v1.17.0...v1.18.0
[1.17.0]: https://github.com/yarpc/yarpc-go/compare/v1.16.0...v1.17.0
[1.16.0]: https://github.com/yarpc/yarpc-go/compare/v1.15.0...v1.16.0
[1.15.0]: https://github.com/yarpc/yarpc-go/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/yarpc/yarpc-go/compare/v1.13.1...v1.14.0
[1.13.1]: https://github.com/yarpc/yarpc-go/compare/v1.13.0...v1.13.1
[1.13.0]: https://github.com/yarpc/yarpc-go/compare/v1.12.1...v1.13.0
[1.12.1]: https://github.com/yarpc/yarpc-go/compare/v1.12.0...v1.12.1
[1.12.0]: https://github.com/yarpc/yarpc-go/compare/v1.11.0...v1.12.0
[1.11.0]: https://github.com/yarpc/yarpc-go/compare/v1.10.0...v1.11.0
[1.10.0]: https://github.com/yarpc/yarpc-go/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/yarpc/yarpc-go/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/yarpc/yarpc-go/compare/v1.7.1...v1.8.0
[1.7.1]: https://github.com/yarpc/yarpc-go/compare/v1.7.0...v1.7.1
[1.7.0]: https://github.com/yarpc/yarpc-go/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/yarpc/yarpc-go/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/yarpc/yarpc-go/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/yarpc/yarpc-go/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/yarpc/yarpc-go/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/yarpc/yarpc-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/yarpc/yarpc-go/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/yarpc/yarpc-go/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/yarpc/yarpc-go/compare/v1.0.0-rc5...v1.0.0
[1.0.0-rc5]: https://github.com/yarpc/yarpc-go/compare/v1.0.0-rc4...v1.0.0-rc5
[1.0.0-rc4]: https://github.com/yarpc/yarpc-go/compare/v1.0.0-rc3...v1.0.0-rc4
[1.0.0-rc3]: https://github.com/yarpc/yarpc-go/compare/v1.0.0-rc2...v1.0.0-rc3
[1.0.0-rc2]: https://github.com/yarpc/yarpc-go/compare/v1.0.0-rc1...v1.0.0-rc2
[1.0.0-rc1]: https://github.com/yarpc/yarpc-go/compare/v0.5.0...v1.0.0-rc1
[0.5.0]: https://github.com/yarpc/yarpc-go/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/yarpc/yarpc-go/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/yarpc/yarpc-go/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/yarpc/yarpc-go/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/yarpc/yarpc-go/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/yarpc/yarpc-go/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/yarpc/yarpc-go/compare/v0.1.0...v0.1.1
