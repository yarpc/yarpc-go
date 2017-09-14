TChannel Semantics
==================

YARPC TChannel requests that are using the Thrift or JSON encodings must use
the [`thrift`] and [`json`] arg schemes respectively. All other requests must
use the `raw` encoding.

Headers
-------

Headers are **case-insensitive** key-value pairs of strings *without
duplicates* (`map[string]string`) and must be in the `arg2` of the TChannel
request.

They must be encoded as,

-   JSON dictionary if the encoding is JSON.

-   Binary payload using the following format for all other encodings.

        nh:2 (k~2 v~2){nh}

    That is, a 16-bit count followed by that many key-value pairs of strings
    prefixed with 16-bit lengths.

  [`thrift`]: https://github.com/uber/tchannel/blob/master/docs/thrift.md
  [`json`]: https://github.com/uber/tchannel/blob/master/docs/json.md
