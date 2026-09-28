package domains

import _ "embed"

// The raw contents of the pipeline values yaml.
//
// This is used by soc-integrations to help control and validate the
// pipeline values delivered to the builds-service.
//
//go:embed resources/com/circleci/expr/domains/pipeline-values.yml
var PipelineValues []byte
