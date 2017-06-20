# Errors

## Mappings

### gRPC

| Code | gRPC Code |
| :---: | :---: |
| ok | OK |
| cancelled | Canceled |
| unknown | Unknown |
| invalid-argument | InvalidArgument |
| deadline-exceeded | DeadlineExceeded |
| not-found | NotFound |
| already-exists | AlreadyExists |
| permission-denied | PermissionDenied |
| resource-exhausted | ResourceExhausted |
| failed-precondition | FailedPrecondition |
| aborted | Aborted |
| out-of-range | OutOfRange |
| unimplemented | Unimplemented |
| internal | Internal |
| unavailable | Unavailable |
| data-loss | DataLoss |
| unauthenticated | Unauthenticated |


### TChannel

| Code | TChannel Code |
| :---: | :---: |
| cancelled | cancelled |
| unknown | unexpected-error |
| invalid-argument | bad-request |
| deadline-exceeded | timeout |
| resource-exhausted | busy |
| unimplemented | bad-request |
| internal | unexpected-error |
| unavailable | declined |
| data-loss | protocol-error |


### HTTP

| Code | HTTP Status Code |
| :---: | :---: |
| ok | 200 |
| cancelled | 499 |
| unknown | 500 |
| invalid-argument | 400 |
| deadline-exceeded | 504 |
| not-found | 404 |
| already-exists | 409 |
| permission-denied | 403 |
| resource-exhausted | 429 |
| failed-precondition | 400 |
| aborted | 409 |
| out-of-range | 400 |
| unimplemented | 501 |
| internal | 500 |
| unavailable | 503 |
| data-loss | 500 |
| unauthenticated | 401 |

