Transport is required.

  $ [[ -n "$TRANSPORT" ]] || (echo 'Please provide an $TRANSPORT' && exit 1)

The trap ensures background processes are killed on exit.

  $ trap 'kill $(jobs -p)' EXIT

Test code:

  $ $TESTDIR/keyvalue/server/server &
  $ sleep 0.250
  $ $TESTDIR/keyvalue/client/client -outbound=$TRANSPORT << INPUT
  > get foo
  > set foo bar
  > get foo
  > set baz qux
  > get baz
  > INPUT
  get "foo" failed: "foo" does not exist
  foo = bar
  baz = qux
