The trap ensures background processes are killed on exit.

  $ trap 'kill $(jobs -p)' EXIT

Test code:

  $ $TESTDIR/hello/hello &
  $ sleep 0.250
  EchoResponse{Message: Hi There, Count: 2} {map[from:self]}
