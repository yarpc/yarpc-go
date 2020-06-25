exception ExceptionWithCode {
    1: required string val
} (
    yarpc.code = "invalid-argument"
)

exception ExceptionWithoutCode {
    1: required string val
}

service TestService  {
    string Call(1: required string key) throws (
      1: ExceptionWithCode exCode,
      2: ExceptionWithoutCode exNoCode,
    )
}
