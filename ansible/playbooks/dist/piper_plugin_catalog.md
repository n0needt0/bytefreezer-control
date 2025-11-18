# ByteFreezer Piper Transformation Plugin Catalog

This document provides a comprehensive reference for all available transformation plugins in the ByteFreezer Piper data processing pipeline. This catalog is used by the AI assistant to generate pipeline configurations based on natural language descriptions.

## Plugin Categories

### 1. Field Manipulation Plugins

#### add_field
**Purpose**: Add new fields to records with support for variable interpolation

**Parameters**:
- `field` (string, required): Name of the field to add
- `value` (string, required): Value to set (supports template variables)

**Template Variables**:
- `${tenant_id}`: Current tenant identifier
- `${dataset_id}`: Current dataset identifier
- `${timestamp}`: Current timestamp
- `${line_number}`: Record line number

**Example**:
```json
{
  "type": "add_field",
  "config": {
    "field": "source",
    "value": "bytefreezer-${tenant_id}"
  }
}
```

---

#### remove_field
**Purpose**: Remove one or multiple fields from records

**Parameters**:
- `field` (string): Single field name to remove
- `fields` (array of strings): Multiple field names to remove

**Example**:
```json
{
  "type": "remove_field",
  "config": {
    "fields": ["temp_field", "debug_info", "internal_id"]
  }
}
```

---

#### rename_field
**Purpose**: Rename a field in the record

**Parameters**:
- `from` (string, required): Original field name
- `to` (string, required): New field name

**Example**:
```json
{
  "type": "rename_field",
  "config": {
    "from": "src_ip",
    "to": "source_ip"
  }
}
```

---

#### mutate
**Purpose**: Advanced field manipulation with multiple operations

**Operations**:
- `split`: Split string into array
- `join`: Join array into string
- `gsub`: Regex-based find and replace
- `convert`: Type conversion (string, int, float, bool)
- `lowercase`: Convert to lowercase
- `uppercase`: Convert to uppercase
- `strip`: Trim whitespace
- `merge`: Merge fields
- `update`: Update field value
- `replace`: Replace field value
- `rename`: Rename field
- `remove`: Remove field
- `copy`: Copy field to new location

**Type Conversions**: string, integer/int, float, boolean/bool

**Example**:
```json
{
  "type": "mutate",
  "config": {
    "convert": {
      "status_code": "integer",
      "response_time": "float"
    },
    "lowercase": ["user_agent"],
    "split": {
      "path": "/"
    }
  }
}
```

---

### 2. Pattern Matching & Parsing Plugins

#### grok
**Purpose**: Parse unstructured log data using Grok patterns

**Parameters**:
- `pattern` (string) or `patterns` (array): Grok patterns to match
- `source_field` (string, default: "message"): Field to parse
- `break_on_match` (bool, default: true): Stop after first match
- `keep_empty_captures` (bool): Keep empty capture groups
- `named_captures_only` (bool, default: true): Only use named captures
- `overwrite_keys` (bool, default: true): Overwrite existing keys
- `target_field` (string): Store parsed data in specific field
- `custom_patterns` (map): Custom Grok pattern definitions

**Common Patterns**:
- `%{COMBINEDAPACHELOG}`: Apache combined log format
- `%{SYSLOGBASE}`: Syslog format
- `%{IP}`: IP address
- `%{TIMESTAMP_ISO8601}`: ISO8601 timestamp
- `%{WORD}`, `%{NUMBER}`, `%{DATA}`: Basic patterns

**Example**:
```json
{
  "type": "grok",
  "config": {
    "source_field": "message",
    "pattern": "%{IP:client_ip} - - \\[%{HTTPDATE:timestamp}\\] \"%{WORD:method} %{URIPATHPARAM:request} HTTP/%{NUMBER:http_version}\" %{NUMBER:status_code} %{NUMBER:bytes}",
    "break_on_match": true
  }
}
```

---

#### regex_replace
**Purpose**: Find and replace using regular expressions

**Parameters**:
- `source_field` (string, default: "message"): Field to process
- `target_field` (string, default: source_field): Output field
- `pattern` (string, required): Regex pattern to match
- `replacement` (string): Replacement string
- `global` (bool, default: true): Replace all occurrences

**Example**:
```json
{
  "type": "regex_replace",
  "config": {
    "source_field": "message",
    "pattern": "\\b\\d{3}-\\d{2}-\\d{4}\\b",
    "replacement": "XXX-XX-XXXX",
    "global": true
  }
}
```

---

#### kv
**Purpose**: Parse key-value pair strings

**Parameters**:
- `source_field` (string, default: "message"): Field containing key-value pairs
- `field_split` (string, default: " "): Separator between pairs
- `value_split` (string, default: "="): Separator between key and value
- `target_field` (string): Store parsed data in specific field
- `prefix` (string): Add prefix to parsed keys
- `include_keys` (array): Only include specified keys
- `exclude_keys` (array): Exclude specified keys
- `trim_key` (string): Characters to trim from keys
- `trim_value` (string): Characters to trim from values
- `allow_duplicate` (bool, default: true): Allow duplicate keys
- `default_values` (map): Default values for missing keys
- `remove_field` (bool): Remove source field after parsing

**Example**:
```json
{
  "type": "kv",
  "config": {
    "source_field": "params",
    "field_split": "&",
    "value_split": "=",
    "target_field": "url_params"
  }
}
```

---

#### date_parse
**Purpose**: Parse date strings with format specifications

**Parameters**:
- `source_field` (string, default: "timestamp"): Field containing date
- `target_field` (string, default: "@timestamp"): Output field
- `format` or `formats` (array): Date format patterns
- `timezone` (string, default: "UTC"): Timezone for parsing
- `locale` (string, default: "en"): Locale for parsing

**Supported Formats**:
- Unix timestamps (seconds, ms, µs, ns) - automatic detection
- RFC3339: `2006-01-02T15:04:05Z07:00`
- Custom Go time formats

**Example**:
```json
{
  "type": "date_parse",
  "config": {
    "source_field": "timestamp",
    "target_field": "@timestamp",
    "formats": ["02/Jan/2006:15:04:05 -0700", "2006-01-02T15:04:05Z"],
    "timezone": "UTC"
  }
}
```

---

### 3. Data Enrichment Plugins

#### geoip
**Purpose**: Geographic information lookup for IP addresses

**Parameters**:
- `source_field` (string, default: "ip_address"): Field containing IP
- `target_field` (string, default: "geoip"): Output field
- `database` (string): Path to GeoLite2 .mmdb file
- `fields` (array): Fields to extract

**Available Fields**:
- `country`: Country name
- `country_code`: ISO country code
- `region`: Region/state name
- `region_code`: Region code
- `city`: City name
- `postal_code`: Postal/ZIP code
- `latitude`: Latitude coordinate
- `longitude`: Longitude coordinate
- `timezone`: Timezone

**Example**:
```json
{
  "type": "geoip",
  "config": {
    "source_field": "client_ip",
    "target_field": "geoip",
    "database": "/var/lib/geoip/GeoLite2-City.mmdb",
    "fields": ["country", "city", "latitude", "longitude"]
  }
}
```

---

#### dns
**Purpose**: DNS lookups for forward/reverse resolution

**Parameters**:
- `resolve` (array, required): Fields to resolve
- `action` (string): "replace" or "append"
- `target_field` (string): Store resolved value
- `nameserver` (array): Custom DNS servers
- `timeout` (int, default: 2): Timeout in seconds
- `cache_size` (int, default: 1000): DNS cache size
- `cache_ttl` (int, default: 3600): Cache TTL in seconds

**Example**:
```json
{
  "type": "dns",
  "config": {
    "resolve": ["server_ip"],
    "action": "append",
    "target_field": "hostname",
    "timeout": 2,
    "cache_size": 1000
  }
}
```

---

#### useragent
**Purpose**: Parse user agent strings into structured data

**Parameters**:
- `source_field` (string, default: "user_agent"): Field containing UA string
- `target_field` (string, default: "ua"): Output field

**Extracted Fields**:
- Browser name and version
- OS name and version
- Device type, brand, and model

**Example**:
```json
{
  "type": "useragent",
  "config": {
    "source_field": "user_agent",
    "target_field": "ua"
  }
}
```

---

#### fingerprint
**Purpose**: Generate hashes/fingerprints for deduplication

**Parameters**:
- `source_fields` (array, required): Fields to include in fingerprint
- `target_field` (string, default: "fingerprint"): Output field
- `method` (string, default: "SHA256"): Hash algorithm (MD5, SHA1, SHA256, SHA512)
- `key_separator` (string, default: "|"): Separator for concatenating values
- `base64encode` (bool): Base64 encode the hash
- `include_keys` (bool): Include field names in hash

**Example**:
```json
{
  "type": "fingerprint",
  "config": {
    "source_fields": ["user_id", "timestamp", "action"],
    "target_field": "event_id",
    "method": "SHA256",
    "base64encode": false
  }
}
```

---

### 4. JSON Processing Plugins

#### json_validate
**Purpose**: Validate JSON content in fields

**Parameters**:
- `source_field` (string, default: "message"): Field to validate
- `fail_on_invalid` (bool, default: true): Skip record if invalid JSON

**Example**:
```json
{
  "type": "json_validate",
  "config": {
    "source_field": "payload",
    "fail_on_invalid": true
  }
}
```

---

#### json_flatten
**Purpose**: Flatten nested JSON objects

**Parameters**:
- `source_field` (string, default: "message"): Field to flatten
- `target_field` (string, default: "@flatten"): Output field
- `separator` (string, default: "."): Key separator for nested fields

**Example**:
```json
{
  "type": "json_flatten",
  "config": {
    "source_field": "nested_data",
    "target_field": "flat",
    "separator": "_"
  }
}
```

---

#### uppercase_keys
**Purpose**: Convert all object keys to uppercase

**Parameters**:
- `source_field` (string, default: "@flatten"): Field to process
- `recursive` (bool, default: true): Process nested objects

**Example**:
```json
{
  "type": "uppercase_keys",
  "config": {
    "source_field": "data",
    "recursive": true
  }
}
```

---

### 5. Filtering & Sampling Plugins

#### include
**Purpose**: Keep only events matching specified conditions (drops everything else)

**Parameters**:
- `field` (string): Field name to check
- `equals` (any): Exact match value
- `contains` (string): Substring match
- `matches` (regex): Regex pattern match
- `any_field` (array): Match if any field exists
- `any_equals` (any): Match if any field equals value
- `any_matches` (regex): Match if any field matches pattern

**Logic**: Uses OR logic for `any_field` conditions

**Example**:
```json
{
  "type": "include",
  "config": {
    "field": "event_type",
    "equals": "file_operation"
  }
}
```

---

#### exclude
**Purpose**: Drop events matching specified conditions (keeps everything else)

**Parameters**:
- `field` (string): Field name to check
- `equals` (any): Exact match value
- `contains` (string): Substring match
- `matches` (regex): Regex pattern match
- `any_field` (array): Match if any field exists
- `any_equals` (any): Match if any field equals value
- `any_matches` (regex): Match if any field matches pattern

**Example**:
```json
{
  "type": "exclude",
  "config": {
    "field": "status_code",
    "equals": 200
  }
}
```

---

#### drop
**Purpose**: Conditionally drop events with field-based or percentage-based conditions

**Parameters**:
- `if_field` (string): Field to check
- `equals` (any): Drop if field equals value
- `not_equals` (any): Drop if field not equals value
- `contains` (string): Drop if field contains string
- `matches` (regex): Drop if field matches pattern
- `percentage` (float, 0-100): Random drop percentage
- `always_drop` (bool): Drop all events
- `unless_field` (string): Don't drop if field exists
- `unless_equals` (any): Don't drop if field equals value
- `unless_matches` (regex): Don't drop if field matches pattern

**Example**:
```json
{
  "type": "drop",
  "config": {
    "if_field": "level",
    "equals": "debug",
    "unless_field": "important"
  }
}
```

---

#### sample
**Purpose**: Random sampling of events (keep percentage)

**Parameters**:
- `percentage` (float, 0-100) or `rate` (float, 0.0-1.0, required): Sampling rate
- `seed` (int): Random seed for deterministic sampling

**Example**:
```json
{
  "type": "sample",
  "config": {
    "percentage": 10
  }
}
```

---

#### conditional
**Purpose**: Conditional filtering based on field evaluation

**Parameters**:
- `field` (string, required): Field to evaluate
- `operator` (string, required): Comparison operator
- `value` (any, required): Value to compare against
- `action` (string): "keep" or "drop"

**Operators**:
- `eq`: Equal to
- `ne`: Not equal to
- `gt`: Greater than
- `lt`: Less than
- `gte`: Greater than or equal
- `lte`: Less than or equal
- `contains`: Contains substring
- `not_contains`: Does not contain substring
- `exists`: Field exists
- `not_exists`: Field does not exist

**Example**:
```json
{
  "type": "conditional",
  "config": {
    "field": "response_time",
    "operator": "gt",
    "value": 1000,
    "action": "keep"
  }
}
```

---

### 6. Structural Transformation Plugins

#### split
**Purpose**: Split single event into multiple events based on array field

**Parameters**:
- `field` (string, required): Field containing array to split
- `terminator` (string): Separator for string splitting
- `target` (string): Field name for split output

**Metadata Added**:
- `_split_field`: Original field name
- `_split_items`: Item value
- `_split_count`: Total items

**Example**:
```json
{
  "type": "split",
  "config": {
    "field": "tags",
    "target": "tag"
  }
}
```

---

## Common Use Cases

### Use Case 1: Web Server Log Processing
```json
{
  "filters": [
    {
      "type": "grok",
      "config": {
        "pattern": "%{COMBINEDAPACHELOG}"
      }
    },
    {
      "type": "date_parse",
      "config": {
        "source_field": "timestamp",
        "formats": ["02/Jan/2006:15:04:05 -0700"]
      }
    },
    {
      "type": "geoip",
      "config": {
        "source_field": "clientip",
        "target_field": "geo"
      }
    },
    {
      "type": "useragent",
      "config": {
        "source_field": "agent"
      }
    }
  ]
}
```

### Use Case 2: PII Removal
```json
{
  "filters": [
    {
      "type": "regex_replace",
      "config": {
        "pattern": "\\b\\d{3}-\\d{2}-\\d{4}\\b",
        "replacement": "[SSN-REDACTED]"
      }
    },
    {
      "type": "regex_replace",
      "config": {
        "pattern": "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b",
        "replacement": "[EMAIL-REDACTED]"
      }
    },
    {
      "type": "remove_field",
      "config": {
        "fields": ["credit_card", "password", "api_key"]
      }
    }
  ]
}
```

### Use Case 3: eBPF Event Filtering
```json
{
  "filters": [
    {
      "type": "include",
      "config": {
        "field": "event_type",
        "equals": "file"
      }
    },
    {
      "type": "exclude",
      "config": {
        "field": "path",
        "matches": "^/(proc|sys|dev)/"
      }
    },
    {
      "type": "geoip",
      "config": {
        "source_field": "remote_ip",
        "fields": ["country", "city"]
      }
    },
    {
      "type": "add_field",
      "config": {
        "field": "processed_by",
        "value": "bytefreezer-${tenant_id}"
      }
    }
  ]
}
```

---

## Pipeline Configuration Best Practices

1. **Order Matters**: Filters execute sequentially - place parsing before enrichment
2. **Filter Early**: Use include/exclude early to reduce processing load
3. **Validate First**: Use json_validate before json_flatten
4. **Type Conversion**: Use mutate convert after grok parsing
5. **Field Cleanup**: Remove temporary fields with remove_field at the end
6. **Error Handling**: Failed filters skip the record by default
7. **Performance**: Use specific field patterns instead of wildcards
8. **Testing**: Always test on sample data before activation

---

## Configuration Validation Rules

1. **Required Parameters**: Must be provided for filter to work
2. **Type Safety**: Values must match expected types (string, int, bool, array)
3. **Mutual Exclusivity**: Some parameters cannot be used together (e.g., `field` vs `fields`)
4. **Dependencies**: Some filters require external resources (GeoIP database, DNS servers)
5. **Regex Syntax**: Regular expressions must be valid Go regex
6. **Field Names**: Field names are case-sensitive

---

## Error Handling Behavior

- **Invalid Configuration**: Pipeline validation fails, activation prevented
- **Runtime Errors**: Record skipped, error logged, processing continues
- **Missing Fields**: Operation skipped, record passed through
- **Type Mismatches**: Conversion attempted, record skipped on failure
- **External Services**: Timeout results in field not being enriched

---

## Performance Considerations

1. **Grok Patterns**: Complex patterns can be CPU-intensive
2. **DNS Lookups**: Enable caching, set reasonable timeouts
3. **GeoIP Lookups**: Memory-mapped database file, fast lookups
4. **Regex Operations**: Compiled once, cached for reuse
5. **Filtering**: Early filtering reduces downstream processing
6. **Sampling**: Use sample plugin for high-volume testing

---

## Version Information

- **Plugin API Version**: 1.0
- **Configuration Schema**: JSON-based filter definitions
- **Supported Data Format**: NDJSON (newline-delimited JSON)
- **Filter Execution**: Sequential chain with skip semantics

---

## Additional Resources

- Test API: POST `/api/v1/transformations/test` - Test filters without S3
- Validate API: POST `/api/v1/transformations/validate` - Test on fresh S3 data
- Schema API: GET `/api/v1/transformations/{tenant}/{dataset}/schema` - Get data schema
- Activate API: POST `/api/v1/transformations/activate` - Activate pipeline
