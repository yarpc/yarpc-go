TEMPLATE_DATA_DIR = internal/templatedata
TEMPLATE_DIR = internal/template
TESTS_DIR = internal/tests

TEMPLATES = \
	$(TEMPLATE_DIR)/base.tmpl \
	$(TEMPLATE_DIR)/client.tmpl \
	$(TEMPLATE_DIR)/client_impl.tmpl \
	$(TEMPLATE_DIR)/client_stream.tmpl \
	$(TEMPLATE_DIR)/fx.tmpl \
	$(TEMPLATE_DIR)/server.tmpl \
	$(TEMPLATE_DIR)/server_impl.tmpl \
	$(TEMPLATE_DIR)/server_stream.tmpl

.PHONY: generate
generate:
ifeq ($(shell which prototool),)
	@go get -u github.com/uber/prototool/internal/cmd/prototool
endif
	@go install .
	@prototool generate $(TESTS_DIR)

# A simple utility used to update our test fixtures when
# we have made a change to the generated templates.
.PHONY: fixtures
fixtures:
	@FIXTURES=true go test . --run TestGenerate

.PHONY: templates
templates: $(TEMPLATES)
ifeq ($(shell which go-bindata),)
	@go get -u github.com/kevinburke/go-bindata/go-bindata
endif
	@go-bindata -pkg templatedata -o ./$(TEMPLATE_DATA_DIR)/bindata.go ./$(TEMPLATE_DIR)
