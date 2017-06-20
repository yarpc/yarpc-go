# Errors

## Mappings

### gRPC

| Code | gRPC Code |
| :---: | :---: |
{{range $code, $grpcCode := .CodeToGRPCCode}}| {{$code.String}} | {{$grpcCode.String}} |
{{end}}

### TChannel

| Code | TChannel Code |
| :---: | :---: |
{{range $code, $tchannelCode := .CodeToTChannelCode}}| {{$code.String}} | {{$tchannelCode.MetricsKey}} |
{{end}}

### HTTP

| Code | HTTP Status Code |
| :---: | :---: |
{{range $code, $httpStatusCode := .CodeToHTTPStatusCode}}| {{$code.String}} | {{$httpStatusCode}} |
{{end}}
