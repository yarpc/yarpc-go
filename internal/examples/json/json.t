Transport is required.

  $ [[ -n "$TRANSPORT" ]] || (echo 'Please provide a $TRANSPORT' && exit 1)

The trap ensures background processes are killed on exit.

  $ trap 'kill $(jobs -p)' EXIT

Test code:

  $ $TESTDIR/server/server &
  $ sleep 0.250
  $ $TESTDIR/client/client -outbound=$TRANSPORT << INPUT
  > get foo
  > set foo bar
  > get foo
  > set baz qux
  > get baz
  > INPUT
  sending request "get" to service "keyvalue" (encoding "json")
  received a request to "get" from client "keyvalue-client" (encoding "json")
  foo = 
  sending request "set" to service "keyvalue" (encoding "json")
  received a request to "set" from client "keyvalue-client" (encoding "json")
  sending request "get" to service "keyvalue" (encoding "json")
  received a request to "get" from client "keyvalue-client" (encoding "json")
  foo = bar
  sending request "set" to service "keyvalue" (encoding "json")
  received a request to "set" from client "keyvalue-client" (encoding "json")
  sending request "get" to service "keyvalue" (encoding "json")
  received a request to "get" from client "keyvalue-client" (encoding "json")
  baz = qux
