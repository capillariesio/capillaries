package sc

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func convertToStringMap(i any) any {
	switch x := i.(type) {
	case map[any]any:
		m2 := map[string]any{}
		for k, v := range x {
			m2[k.(string)] = convertToStringMap(v)
		}
		return m2
	case []any:
		for i, v := range x {
			x[i] = convertToStringMap(v)
		}
	}
	return i
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
	if scriptType == ScriptJson {
		if err := json.Unmarshal(in, out); err != nil {
			return fmt.Errorf("cannot unmarshal json: %s", err.Error())
		}
	} else if scriptType == ScriptYaml {
		if err := yamlUnmarshal(in, out); err != nil {
			return fmt.Errorf("cannot unmarshal yaml: %s", err.Error())
		}
	} else {
		return fmt.Errorf("cannot unmarshal yaml or json, unknown format")
	}

	return nil
}
