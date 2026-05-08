package sc

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

func convertToStringMap(i any) any {
	switch x := i.(type) {
	// Not sure about this: Json unmarshal does not support it, and we unmarshal Yaml via Json
	// case map[any]any:
	// 	m2 := map[any]any{}
	// 	for k, v := range x {
	// 		m2[k] = convertToStringMap(v)
	// 	}
	// 	return m2
	case map[string]any:
		m2 := map[string]any{}
		for k, v := range x {
			m2[k] = convertToStringMap(v)
		}
		return m2
	case []any:
		for i, v := range x {
			x[i] = convertToStringMap(v)
		}
		return i
	default:
		return i
	}
}

func yamlUnmarshal(in []byte, out any) error {
	var body any
	if err := yaml.Unmarshal(in, &body); err != nil {
		return err
	}

	body = convertToStringMap(body)

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, out)
}

func JsonOrYamlUnmarshal(scriptType ScriptType, in []byte, out any) error {
	switch scriptType {
	case ScriptJson:
		if err := json.Unmarshal(in, out); err != nil {
			return fmt.Errorf("cannot unmarshal json: %s", err.Error())
		}
	case ScriptYaml:
		if err := yamlUnmarshal(in, out); err != nil {
			return fmt.Errorf("cannot unmarshal yaml: %s", err.Error())
		}
	default:
		return errors.New("cannot unmarshal yaml or json, unknown format")
	}

	return nil
}
