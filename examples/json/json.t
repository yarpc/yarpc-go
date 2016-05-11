Transport is required.

  $ [[ -n "$TRANSPORT" ]] || (echo 'Please provide an $TRANSPORT' && exit 1)

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
  sending a request to "get"
  received a request to "get"
  foo = 
  sending a request to "set"
  received a request to "set"
  sending a request to "get"
  received a request to "get"
  foo = bar
  sending a request to "set"
  received a request to "set"
  sending a request to "get"
  received a request to "get"
  baz = qux
