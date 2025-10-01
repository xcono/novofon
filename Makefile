.PHONY: all

all: go

go:
	docker run --rm -v "./api:/local" openapitools/openapi-generator-cli generate \
    -i /local/contact.yaml \
    -g go \
    -o /local/clients/go/contact
