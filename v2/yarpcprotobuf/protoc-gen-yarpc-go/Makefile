TEMPLATE_DATA_DIR = internal/templatedata
TESTS_DIR = internal/tests

TEMPLATES = \
	$(TEMPLATE_DATA_DIR)/base.tmpl \
	$(TEMPLATE_DATA_DIR)/client.tmpl \
	$(TEMPLATE_DATA_DIR)/client_impl.tmpl \
	$(TEMPLATE_DATA_DIR)/client_stream.tmpl \
	$(TEMPLATE_DATA_DIR)/fx.tmpl \
	$(TEMPLATE_DATA_DIR)/server.tmpl \
	$(TEMPLATE_DATA_DIR)/server_impl.tmpl \
	$(TEMPLATE_DATA_DIR)/server_stream.tmpl

.PHONY: generate
generate:
ifeq ($(shell which prototool),)
	@go get -u github.com/uber/prototool/internal/cmd/prototool
endif
	@prototool generate $(TESTS_DIR)

.PHONY: templates
templates: $(TEMPLATES)
ifeq ($(shell which go-bindata),)
	@go get -u github.com/jteeuwen/go-bindata/go-bindata
endif
	@go-bindata -pkg templatedata -o ./$(TEMPLATE_DATA_DIR)/bindata.go ./$(TEMPLATE_DATA_DIR)
