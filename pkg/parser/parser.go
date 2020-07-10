package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInternal  = errors.New("internal error")
	ErrTransport = errors.New("transport error")
)

type Parser struct {
	// GraphqlEndpoint specifies an alternative GraphQL endpoint to use. If empty, DefaultGraphqlEndpoint is used.
	GraphqlEndpoint string
}

const (
	DefaultGraphqlEndpoint = "https://api.cirrus-ci.com/graphql"
	defaultTimeout         = time.Second * 5
)

type Result struct {
	Errors []string
}

const gqlQuery = `mutation Validate($clientMutationId: String, $config: String) {
  validate(input: {clientMutationId: $clientMutationId, config: $config}) {
    errors
  }
}
`

type gqlValidate struct {
	Errors []string
}

type gqlData struct {
	Validate gqlValidate
}

type gqlError struct {
	Message string
}

type gqlResponse struct {
	Data   *gqlData
	Errors []gqlError
}

func (p *Parser) graphqlEndpoint() string {
	if p.GraphqlEndpoint == "" {
		return DefaultGraphqlEndpoint
	}

	return p.GraphqlEndpoint
}

func (p *Parser) Parse(config string) (*Result, error) {
	// Prepare GraphQL query
	query := struct {
		Query     string            `json:"query"`
		Variables map[string]string `json:"variables"`
	}{
		Query: gqlQuery,
		Variables: map[string]string{
			"clientMutationId": uuid.New().String(),
			"config":           config,
		},
	}
	buf, err := json.Marshal(&query)
	if err != nil {
		return nil, fmt.Errorf("%w, %v", ErrInternal, err)
	}
	requestBody := bytes.NewBuffer(buf)

	// Send config for validation via Cirrus CI API
	client := http.Client{Timeout: defaultTimeout}
	resp, err := client.Post(p.graphqlEndpoint(), "text/json", requestBody)
	if err != nil {
		return nil, fmt.Errorf("%w, %v", ErrTransport, err)
	}
	defer resp.Body.Close()

	// GraphQL's convention is to always return HTTP 200,
	// in all other cases something has really gone wrong.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: got HTTP %d (%s)", ErrTransport, resp.StatusCode, resp.Status)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransport, err)
	}

	// Parse and validate JSON result
	var parsed gqlResponse
	err = json.Unmarshal(responseBody, &parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransport, err)
	}

	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrInternal, parsed.Errors[0].Message)
	}

	if parsed.Data == nil {
		return nil, fmt.Errorf("%w: empty GraphQL response without errors", ErrInternal)
	}

	return &Result{Errors: parsed.Data.Validate.Errors}, nil
}

func (p *Parser) ParseFromFile(path string) (*Result, error) {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.Parse(string(config))
}
