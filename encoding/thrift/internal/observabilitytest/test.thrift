exception ExceptionWithCode {
    1: required string val
} (
    rpc.code = "DATA_LOSS" // server error
)

exception ExceptionWithoutCode {
    1: required string val
}

service TestService  {
    string Call(1: required string key) throws (
      2: ExceptionWithoutCode exNoCode,
      1: ExceptionWithCode exCode,
    )
}
