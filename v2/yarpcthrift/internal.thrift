enum ExceptionType {
  UNKNOWN = 0
  UNKNOWN_METHOD = 1
  INVALID_MESSAGE_TYPE = 2
  WRONG_METHOD_NAME = 3
  BAD_SEQUENCE_ID = 4
  MISSING_RESULT = 5
  INTERNAL_ERROR = 6
  PROTOCOL_ERROR = 7
  INVALID_TRANSFORM = 8
  INVALID_PROTOCOL = 9
  UNSUPPORTED_CLIENT_TYPE = 10
}

/**
 * TApplicationException is a Thrift-level exception.
 *
 * Thrift envelopes with the type Exception contain an exception of this
 * shape.
 */
exception TApplicationException {
  1: optional string message
  2: optional ExceptionType type
}
