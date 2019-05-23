package validator

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

// Version represents the Kubernetes version.
var Version string

// SchemaLocation represents the URL of schema location.
var SchemaLocation string

// DefaultSchemaLocation is the default value for SchemaLocation.
var DefaultSchemaLocation = "https://kubernetesjsonschema.dev"

// ValidFormat is for some formats in Kubernetes loaded to gojsonschema.
type ValidFormat struct{}

// IsFormat fulfills gojsonschema.FormatChecker interface.
func (f ValidFormat) IsFormat(input interface{}) bool {
	return true
}

// ValidationResult contains Kubernetes resource schema validation result.
type ValidationResult struct {
	FileName   string
	Kind       string
	APIVersion string
	Errors     []gojsonschema.ResultError
}

func getLineBreak(body []byte) string {
	lineBreak := "\n"
	if windowsLineEnding := bytes.Contains(body, []byte("\r\n")); windowsLineEnding && runtime.GOOS == "windows" {
		lineBreak = "\r\n"
	}
	return fmt.Sprintf("%s---%s", lineBreak, lineBreak)
}

func getSchema(kind string, apiVersion string) string {
	// set a default Version
	if Version == "" {
		Version = "master"
	}
	// add prefix `v` into schema to match the tag in Kubernetes repo
	normalisedVersion := Version
	if Version != "master" {
		normalisedVersion = "v" + normalisedVersion
	}

	// environment variable -> override in SchemaLocation -> default value
	baseURLFromEnv := viper.GetString("schema_location")
	var baseURL string
	if baseURLFromEnv != "" {
		baseURL = baseURLFromEnv
	} else if SchemaLocation != "" {
		baseURL = SchemaLocation
	} else {
		baseURL = DefaultSchemaLocation
	}

	strictSuffix := "-strict"
	var kindSuffix string
	// apiVersion "v1"   "apps/v1"   "apiextensions.k8s.io/v1beta1"  "storage.k8s.io/v1"
	// kindSuffix "-v1"  "-apps-v1"  "-apiextensions-v1beta1"        "-storage-v1"
	// schema example https://kubernetesjsonschema.dev/master-standalone-strict/service-v1.json
	// schema example https://kubernetesjsonschema.dev/master-standalone-strict/storageclass-storage-v1.json
	groupParts := strings.Split(apiVersion, "/")
	versionParts := strings.Split(groupParts[0], ".")

	if len(groupParts) == 1 {
		kindSuffix = "-" + strings.ToLower(versionParts[0])
	} else {
		kindSuffix = fmt.Sprintf("-%s-%s", strings.ToLower(versionParts[0]), strings.ToLower(groupParts[1]))
	}

	return fmt.Sprintf("%s/%s-standalone%s/%s%s.json", baseURL, normalisedVersion, strictSuffix,
		strings.ToLower(kind), kindSuffix)
}

func getKind(body interface{}) (string, error) {
	bodyMap, _ := body.(map[string]interface{})
	if _, ok := bodyMap["kind"]; !ok {
		return "", errors.New("Missing kind key")
	}
	if bodyMap["kind"] == nil {
		return "", errors.New("Missing kind value")
	}
	return bodyMap["kind"].(string), nil
}

func getAPIVersion(body interface{}) (string, error) {
	bodyMap, _ := body.(map[string]interface{})
	if _, ok := bodyMap["apiVersion"]; !ok {
		return "", errors.New("Missing apiVersion key")
	}
	if bodyMap["apiVersion"] == nil {
		return "", errors.New("Missing apiVersion value")
	}
	return bodyMap["apiVersion"].(string), nil
}

// validateResource validates a single Kubernetes resource against the Kubernetes json schema.
func validateResource(data []byte, fileName string) (ValidationResult, error) {
	var spec interface{}
	result := ValidationResult{FileName: fileName}
	err := yaml.Unmarshal(data, &spec)
	if err != nil {
		return result, errors.New("Failed to decode YAML from " + fileName)
	}

	body := convertToStringKeys(spec)
	if body == nil {
		return result, nil
	}

	bodyMap, _ := body.(map[string]interface{})
	if len(bodyMap) == 0 {
		return result, nil
	}

	documentLoader := gojsonschema.NewGoLoader(body)

	kind, err := getKind(body)
	if err != nil {
		return result, err
	}
	result.Kind = kind

	apiVersion, err := getAPIVersion(body)
	if err != nil {
		return result, err
	}
	result.APIVersion = apiVersion

	schema := getSchema(kind, apiVersion)
	schemaLoader := gojsonschema.NewReferenceLoader(schema)

	// add types into gojsonschema so the gojsonschema loading works
	gojsonschema.FormatCheckers.Add("int64", ValidFormat{})
	gojsonschema.FormatCheckers.Add("byte", ValidFormat{})
	gojsonschema.FormatCheckers.Add("int32", ValidFormat{})
	gojsonschema.FormatCheckers.Add("int-or-string", ValidFormat{})

	results, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return result, fmt.Errorf("Failed to load schema from %s: %s", schema, err)
	}

	if results.Valid() {
		return result, nil
	}

	result.Errors = results.Errors()
	return result, nil
}

// Validate individual resources in a Kubernetes YAML file against the Kubernetes json schema.
func Validate(config []byte, fileName string) ([]ValidationResult, error) {
	results := make([]ValidationResult, 0)

	if len(config) == 0 {
		results = append(results, ValidationResult{FileName: fileName})
		return results, nil
	}

	bits := bytes.Split(config, []byte(getLineBreak(config)))

	var errors *multierror.Error
	for _, element := range bits {
		if len(element) > 0 {
			result, err := validateResource(element, fileName)
			results = append(results, result)
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		} else {
			results = append(results, ValidationResult{FileName: fileName})
		}
	}
	return results, errors.ErrorOrNil()
}

func convertToStringKeys(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[k.(string)] = convertToStringKeys(v)
		}
		return m
	case []interface{}:
		for idx, v := range x {
			x[idx] = convertToStringKeys(v)
		}
	}
	return i
}
