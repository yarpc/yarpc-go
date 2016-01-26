project := yarpc-go


.PHONY: install
install:
	go build ./...


.PHONY: test
test:
	go test ./...


.PHONY: trigger-xlang-tests
trigger-xlang-tests:
	curl  -H 'Content-Type: application/json' -H  'Authorization: Bearer 1efb2bf4f2accef6081e7bad7e3b5e2a6f0312fb80bc72f2986787b35d470e32' -X POST -d '{"applicationId": "56a7ea58bb1e57431b01ec5e", "branch":"master", "message":"triggered from yarpc-go"}' https://app.wercker.com/api/v3/builds
