# ydump

Command-line utility for dumping YARPC requests and Thrift encoded structs. 

## Installation

```
go get go.uber.org/yarpc/x/ydump
```

## Usage

### Encoding

```
$ echo "X: abcde" | ydump -t examples/foo.thrift -symbol Foo -serialize=true | base64
CwABAAAABWFiY2RlAA==
$ echo "X: abcde" | ydump -t examples/foo.thrift -symbol Foo -serialize=true | hexdump -C
00000000  0b 00 01 00 00 00 05 61  62 63 64 65 00           |.......abcde.|
0000000d
```

### Decoding

```
$ echo "CwABAAAABWFiY2RlAA==" | base64 -D | ydump -t examples/foo.thrift -symbol Foo
X: abcde
```