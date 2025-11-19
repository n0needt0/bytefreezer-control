package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "/etc/bytefreezer/control/config.yaml", "Path to config file")
	version := flag.String("version", "1.0.0", "Piper version")
	flag.Parse()

	// Load configuration
	log.Info("Loading configuration...")
	configData, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	type Config struct {
		Database struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Database string `yaml:"database"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
			SSLMode  string `yaml:"ssl_mode"`
		} `yaml:"database"`
	}

	var conf Config
	if err := yaml.Unmarshal(configData, &conf); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Create storage backend
	log.Info("Connecting to database...")

	uri := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		conf.Database.Host,
		conf.Database.Port,
		conf.Database.Username,
		conf.Database.Password,
		conf.Database.Database,
		conf.Database.SSLMode,
	)

	storageConfig := storage.Config{
		Type: "postgresql",
		URI:  uri,
	}

	store, err := storage.NewPostgreSQLStorage(storageConfig)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Health check
	ctx := context.Background()
	if err := store.HealthCheck(ctx); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	log.Info("Database connection established")

	// Seed filters
	log.Info("Seeding filter catalog...")
	if err := seedFilters(ctx, store, *version); err != nil {
		log.Fatalf("Failed to seed filters: %v", err)
	}

	log.Info("Filter catalog seeded successfully!")
}

func seedFilters(ctx context.Context, store storage.Storage, version string) error {
	filters := []storage.PiperFilter{
		// Event Filtering Filters
		{
			FilterType:  "include",
			DisplayName: "Include Filter",
			Category:    "filtering",
			Purpose:     "Keep only events matching conditions (drop all others)",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: false, Description: "Field name to check"},
				{Name: "equals", Type: "any", Required: false, Description: "Value to match exactly"},
				{Name: "contains", Type: "string", Required: false, Description: "Substring to match"},
				{Name: "matches", Type: "string", Required: false, Description: "Regex pattern to match"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Keep only error events",
					Config: map[string]interface{}{
						"field":  "level",
						"equals": "error",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "exclude",
			DisplayName: "Exclude Filter",
			Category:    "filtering",
			Purpose:     "Drop events matching conditions",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: false, Description: "Field name to check"},
				{Name: "equals", Type: "any", Required: false, Description: "Value to match exactly"},
				{Name: "contains", Type: "string", Required: false, Description: "Substring to match"},
				{Name: "matches", Type: "string", Required: false, Description: "Regex pattern to match"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Drop debug events",
					Config: map[string]interface{}{
						"field":  "level",
						"equals": "debug",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "sample",
			DisplayName: "Sample Filter",
			Category:    "filtering",
			Purpose:     "Keep only a percentage of events (random sampling)",
			Parameters: []storage.FilterParameter{
				{Name: "percentage", Type: "float", Required: true, Description: "Percentage of events to keep (0-100)"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Keep 10% of events",
					Config: map[string]interface{}{
						"percentage": 10,
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "drop",
			DisplayName: "Drop Filter",
			Category:    "filtering",
			Purpose:     "Advanced conditional dropping",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: false, Description: "Field name to check"},
				{Name: "equals", Type: "any", Required: false, Description: "Value to match"},
				{Name: "contains", Type: "string", Required: false, Description: "Substring to match"},
				{Name: "matches", Type: "string", Required: false, Description: "Regex pattern to match"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Drop health check requests",
					Config: map[string]interface{}{
						"field":   "path",
						"matches": "^/health",
					},
				},
			},
			Version: version,
		},

		// Parsing Filters
		{
			FilterType:  "grok",
			DisplayName: "Grok Filter",
			Category:    "parsing",
			Purpose:     "Parse unstructured text using grok patterns",
			Parameters: []storage.FilterParameter{
				{Name: "pattern", Type: "string", Required: true, Description: "Grok pattern"},
				{Name: "source_field", Type: "string", Required: false, Default: "message", Description: "Field to parse"},
				{Name: "target_field", Type: "string", Required: false, Description: "Field for parsed data"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Parse Apache logs",
					Config: map[string]interface{}{
						"pattern": "%{COMMONAPACHELOG}",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "kv",
			DisplayName: "Key-Value Filter",
			Category:    "parsing",
			Purpose:     "Parse key-value pairs from strings",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: false, Default: "message", Description: "Field to parse"},
				{Name: "field_split", Type: "string", Required: false, Default: " ", Description: "Separator between pairs"},
				{Name: "value_split", Type: "string", Required: false, Default: "=", Description: "Separator between key and value"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Parse query string parameters",
					Config: map[string]interface{}{
						"source_field": "params",
						"field_split":  "&",
						"value_split":  "=",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "date_parse",
			DisplayName: "Date Parse Filter",
			Category:    "parsing",
			Purpose:     "Parse timestamp strings into structured dates",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: true, Description: "Field containing timestamp"},
				{Name: "target_field", Type: "string", Required: false, Default: "@timestamp", Description: "Field to store parsed date"},
				{Name: "formats", Type: "array", Required: false, Description: "Array of date formats to try"},
				{Name: "timezone", Type: "string", Required: false, Default: "UTC", Description: "Timezone for parsing"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Parse ISO8601 timestamp",
					Config: map[string]interface{}{
						"source_field": "timestamp",
						"formats":      []string{"2006-01-02T15:04:05Z07:00"},
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "split",
			DisplayName: "Split Filter",
			Category:    "parsing",
			Purpose:     "Split one event into multiple events",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: true, Description: "Field containing array to split"},
				{Name: "separator", Type: "string", Required: false, Description: "String separator if field is string"},
				{Name: "target", Type: "string", Required: false, Description: "Field name for split value in new events"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Split comma-separated tags",
					Config: map[string]interface{}{
						"field":     "tags",
						"separator": ",",
						"target":    "tag",
					},
				},
			},
			Version: version,
		},

		// Field Manipulation Filters
		{
			FilterType:  "add_field",
			DisplayName: "Add Field Filter",
			Category:    "transformation",
			Purpose:     "Add new fields to records",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: true, Description: "Field name to add"},
				{Name: "value", Type: "any", Required: true, Description: "Value to set"},
				{Name: "overwrite", Type: "boolean", Required: false, Default: false, Description: "Overwrite if field exists"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Add environment field",
					Config: map[string]interface{}{
						"field": "environment",
						"value": "production",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "remove_field",
			DisplayName: "Remove Field Filter",
			Category:    "transformation",
			Purpose:     "Remove fields from records",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: false, Description: "Single field name to remove"},
				{Name: "fields", Type: "array", Required: false, Description: "Multiple fields to remove"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Remove sensitive fields",
					Config: map[string]interface{}{
						"fields": []string{"password", "ssn", "credit_card"},
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "rename_field",
			DisplayName: "Rename Field Filter",
			Category:    "transformation",
			Purpose:     "Rename fields in records",
			Parameters: []storage.FilterParameter{
				{Name: "from", Type: "string", Required: true, Description: "Original field name"},
				{Name: "to", Type: "string", Required: true, Description: "New field name"},
				{Name: "overwrite", Type: "boolean", Required: false, Default: false, Description: "Overwrite if target exists"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Rename msg to message",
					Config: map[string]interface{}{
						"from": "msg",
						"to":   "message",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "mutate",
			DisplayName: "Mutate Filter",
			Category:    "transformation",
			Purpose:     "Multiple field operations (add, remove, rename, convert)",
			Parameters: []storage.FilterParameter{
				{Name: "add", Type: "object", Required: false, Description: "Fields to add (field: value)"},
				{Name: "remove", Type: "array", Required: false, Description: "Fields to remove"},
				{Name: "rename", Type: "object", Required: false, Description: "Fields to rename (from: to)"},
				{Name: "uppercase", Type: "array", Required: false, Description: "Fields to uppercase"},
				{Name: "lowercase", Type: "array", Required: false, Description: "Fields to lowercase"},
				{Name: "convert", Type: "object", Required: false, Description: "Fields to convert types (field: type)"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Multiple transformations",
					Config: map[string]interface{}{
						"add":    map[string]interface{}{"environment": "production"},
						"remove": []string{"temp", "debug"},
						"rename": map[string]interface{}{"msg": "message"},
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "regex_replace",
			DisplayName: "Regex Replace Filter",
			Category:    "transformation",
			Purpose:     "Find and replace using regular expressions",
			Parameters: []storage.FilterParameter{
				{Name: "field", Type: "string", Required: true, Description: "Field to operate on"},
				{Name: "pattern", Type: "string", Required: true, Description: "Regex pattern to match"},
				{Name: "replacement", Type: "string", Required: true, Description: "Replacement string"},
				{Name: "target", Type: "string", Required: false, Description: "Target field for result"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Mask SSN numbers",
					Config: map[string]interface{}{
						"field":       "message",
						"pattern":     "\\d{3}-\\d{2}-\\d{4}",
						"replacement": "XXX-XX-XXXX",
					},
				},
			},
			Version: version,
		},

		// Enrichment Filters
		{
			FilterType:  "geoip",
			DisplayName: "GeoIP Filter",
			Category:    "enrichment",
			Purpose:     "Add geographic information from IP addresses",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: true, Description: "Field containing IP address"},
				{Name: "target_field", Type: "string", Required: false, Default: "geoip", Description: "Field for geo data"},
				{Name: "database_path", Type: "string", Required: false, Description: "Path to MaxMind database"},
				{Name: "fields", Type: "array", Required: false, Description: "Specific fields to extract"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "GeoIP lookup on client IP",
					Config: map[string]interface{}{
						"source_field": "client_ip",
						"target_field": "geo",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "useragent",
			DisplayName: "User Agent Filter",
			Category:    "enrichment",
			Purpose:     "Parse user agent strings",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: true, Description: "Field containing user agent"},
				{Name: "target_field", Type: "string", Required: false, Default: "user_agent", Description: "Field for parsed data"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Parse user agent string",
					Config: map[string]interface{}{
						"source_field": "user_agent",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "dns",
			DisplayName: "DNS Filter",
			Category:    "enrichment",
			Purpose:     "Perform DNS lookups",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: true, Description: "Field containing IP or hostname"},
				{Name: "target_field", Type: "string", Required: false, Description: "Field for result"},
				{Name: "action", Type: "string", Required: false, Default: "reverse", Description: "reverse or resolve", Options: []string{"reverse", "resolve"}},
				{Name: "timeout", Type: "int", Required: false, Description: "Timeout in seconds"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Reverse DNS lookup",
					Config: map[string]interface{}{
						"source_field": "client_ip",
						"target_field": "hostname",
						"action":       "reverse",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "fingerprint",
			DisplayName: "Fingerprint Filter",
			Category:    "enrichment",
			Purpose:     "Generate hash fingerprints for events",
			Parameters: []storage.FilterParameter{
				{Name: "source", Type: "array", Required: false, Description: "Fields to include in hash"},
				{Name: "target_field", Type: "string", Required: false, Default: "fingerprint", Description: "Field for fingerprint"},
				{Name: "method", Type: "string", Required: false, Default: "SHA256", Description: "Hash method", Options: []string{"MD5", "SHA1", "SHA256"}},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Generate SHA256 fingerprint",
					Config: map[string]interface{}{
						"source": []string{"message", "timestamp", "host"},
						"method": "SHA256",
					},
				},
			},
			Version: version,
		},

		// Utility Filters
		{
			FilterType:  "conditional",
			DisplayName: "Conditional Filter",
			Category:    "utility",
			Purpose:     "Apply filters based on conditions",
			Parameters: []storage.FilterParameter{
				{Name: "condition", Type: "string", Required: true, Description: "Condition expression"},
				{Name: "if_true", Type: "array", Required: false, Description: "Filters to apply if condition is true"},
				{Name: "if_false", Type: "array", Required: false, Description: "Filters to apply if condition is false"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Add alert field for errors",
					Config: map[string]interface{}{
						"condition": "{{.level}} == 'error'",
						"if_true": []map[string]interface{}{
							{
								"type": "add_field",
								"config": map[string]interface{}{
									"field": "alert",
									"value": "true",
								},
							},
						},
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "json_validate",
			DisplayName: "JSON Validate Filter",
			Category:    "utility",
			Purpose:     "Validate JSON structure and schema",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: false, Description: "Field to validate (default: entire record)"},
				{Name: "schema", Type: "object", Required: false, Description: "JSON schema for validation"},
				{Name: "drop_invalid", Type: "boolean", Required: false, Default: false, Description: "Drop invalid events"},
				{Name: "tag_on_failure", Type: "string", Required: false, Description: "Tag to add on validation failure"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Validate and drop invalid JSON",
					Config: map[string]interface{}{
						"source_field": "payload",
						"drop_invalid": true,
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "json_flatten",
			DisplayName: "JSON Flatten Filter",
			Category:    "utility",
			Purpose:     "Flatten nested JSON objects",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "string", Required: false, Description: "Field to flatten (default: entire record)"},
				{Name: "separator", Type: "string", Required: false, Default: ".", Description: "Separator for nested keys"},
				{Name: "depth", Type: "int", Required: false, Default: 0, Description: "Maximum depth (0 = unlimited)"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Flatten with dot separator",
					Config: map[string]interface{}{
						"separator": ".",
					},
				},
			},
			Version: version,
		},
		{
			FilterType:  "uppercase_keys",
			DisplayName: "Uppercase Keys Filter",
			Category:    "utility",
			Purpose:     "Convert all object keys to uppercase",
			Parameters: []storage.FilterParameter{
				{Name: "source_field", Type: "array or string", Required: false, Description: "Array of fields to operate on, single field name, \"*\" for all fields, or empty/omitted for entire record"},
				{Name: "recursive", Type: "boolean", Required: false, Default: true, Description: "Recursively uppercase nested objects"},
			},
			Examples: []storage.FilterExample{
				{
					Description: "Uppercase all keys in entire record",
					Config: map[string]interface{}{
						"recursive": true,
					},
				},
				{
					Description: "Uppercase keys in specific field",
					Config: map[string]interface{}{
						"source_field": "metadata",
						"recursive":    true,
					},
				},
				{
					Description: "Uppercase keys in multiple fields",
					Config: map[string]interface{}{
						"source_field": []string{"metadata", "context"},
						"recursive":    true,
					},
				},
				{
					Description: "Apply to all fields (same as omitting source_field)",
					Config: map[string]interface{}{
						"source_field": "*",
						"recursive":    true,
					},
				},
			},
			Version: version,
		},
	}

	// Upsert each filter
	for _, filter := range filters {
		log.Infof("Seeding filter: %s", filter.FilterType)
		if err := store.UpsertPiperFilter(ctx, &filter); err != nil {
			return fmt.Errorf("failed to seed filter %s: %w", filter.FilterType, err)
		}
	}

	log.Infof("Successfully seeded %d filters", len(filters))
	return nil
}
