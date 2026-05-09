package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type ContractValidator struct{}

func NewContractValidator() *ContractValidator {
	return &ContractValidator{}
}

func (v *ContractValidator) Compile(schemaRaw string) (*jsonschema.Schema, error) {
	schema, err := compileContractSchema(schemaRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidContractSchema, err)
	}

	return schema, nil
}

func (v *ContractValidator) ValidateInput(schemaRaw string, value json.RawMessage) error {
	schema, err := v.Compile(schemaRaw)
	if err != nil {
		return err
	}
	if err := validateContractValue(schema, value); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrContractInputInvalid, err)
	}

	return nil
}

func (v *ContractValidator) ValidateOutput(schemaRaw string, value json.RawMessage) error {
	schema, err := v.Compile(schemaRaw)
	if err != nil {
		return err
	}
	if err := validateContractValue(schema, value); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrContractOutputInvalid, err)
	}

	return nil
}

func compileContractSchema(schemaRaw string) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaRaw))
	if err != nil {
		return nil, err
	}
	if err := compiler.AddResource("schema.json", doc); err != nil {
		return nil, err
	}

	return compiler.Compile("schema.json")
}

func validateContractValue(schema *jsonschema.Schema, value json.RawMessage) error {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(value)))
	if err != nil {
		return err
	}

	return schema.Validate(doc)
}
