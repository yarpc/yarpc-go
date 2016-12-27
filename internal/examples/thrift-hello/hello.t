The trap ensures background processes are killed on exit.

  $ trap 'kill $(jobs -p)' EXIT

Test code:

  $ $TESTDIR/hello/hello &
  $ sleep 0.250
  EchoResponse{Message: Hello world, Count: 2}
