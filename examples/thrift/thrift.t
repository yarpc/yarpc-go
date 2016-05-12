Transport is required.

  $ [[ -n "$TRANSPORT" ]] || (echo 'Please provide an $TRANSPORT' && exit 1)

The trap ensures background processes are killed on exit.

  $ trap 'kill $(jobs -p)' EXIT

Test code:

  $ $TESTDIR/keyvalue/server/server &
  $ sleep 0.250
  $ $TESTDIR/keyvalue/client/client -outbound=$TRANSPORT << INPUT
  > get foo
  > get foo
  > set foo bar
  > get foo
  > get foo
  > set baz qux
  > get baz
  > get foo
  > get baz
  > INPUT
  cache miss
  get "foo" failed: ResourceDoesNotExist{Key: foo}
  cache hit
  get "foo" failed: ResourceDoesNotExist{Key: foo}
  cache invalidate
  cache miss
  foo = bar
  cache hit
  foo = bar
  cache invalidate
  cache miss
  baz = qux
  cache miss
  foo = bar
  cache hit
  baz = qux
