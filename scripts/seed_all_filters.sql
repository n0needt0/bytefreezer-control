-- Seed all available piper filters into control_piper_filter_catalog
-- This should be run whenever new filters are added to piper

-- Clear existing entries (optional - remove if you want to keep manual additions)
-- TRUNCATE TABLE control_piper_filter_catalog;

INSERT INTO control_piper_filter_catalog (filter_type, display_name, category, purpose, parameters, examples, version)
VALUES
    ('add_field', 'Add Field', 'field_manipulation', 'Add new fields to records',
     '[{"name":"field","type":"string","required":true,"description":"Field name to add"},{"name":"value","type":"any","required":true,"description":"Value to assign"}]'::jsonb,
     '[{"description":"Add static field","config":{"field":"environment","value":"production"}}]'::jsonb, '1.0.0'),

    ('remove_field', 'Remove Field', 'field_manipulation', 'Remove fields from records',
     '[{"name":"fields","type":"array","required":true,"description":"List of field names to remove"}]'::jsonb,
     '[{"description":"Remove sensitive fields","config":{"fields":["password","ssn"]}}]'::jsonb, '1.0.0'),

    ('rename_field', 'Rename Field', 'field_manipulation', 'Rename fields in records',
     '[{"name":"from","type":"string","required":true,"description":"Original field name"},{"name":"to","type":"string","required":true,"description":"New field name"}]'::jsonb,
     '[{"description":"Rename user_id to userId","config":{"from":"user_id","to":"userId"}}]'::jsonb, '1.0.0'),

    ('regex_replace', 'Regex Replace', 'transformation', 'Replace text using regular expressions',
     '[{"name":"field","type":"string","required":true,"description":"Field to apply regex on"},{"name":"pattern","type":"string","required":true,"description":"Regex pattern to match"},{"name":"replacement","type":"string","required":true,"description":"Replacement string"}]'::jsonb,
     '[{"description":"Redact email addresses","config":{"field":"message","pattern":"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}","replacement":"[EMAIL]"}}]'::jsonb, '1.0.0'),

    ('date_parse', 'Date Parse', 'datetime', 'Parse date strings into timestamp',
     '[{"name":"field","type":"string","required":true,"description":"Field containing date string"},{"name":"format","type":"string","required":true,"description":"Date format layout"},{"name":"target","type":"string","required":false,"description":"Target field name (defaults to @timestamp)"}]'::jsonb,
     '[{"description":"Parse custom date format","config":{"field":"created_at","format":"2006-01-02 15:04:05","target":"@timestamp"}}]'::jsonb, '1.0.0'),

    ('conditional', 'Conditional', 'logic', 'Apply filters conditionally based on field values',
     '[{"name":"condition","type":"string","required":true,"description":"Condition expression"},{"name":"filters","type":"array","required":true,"description":"Filters to apply if condition is true"}]'::jsonb,
     '[{"description":"Apply filter if status is error","config":{"condition":"status == \"error\"","filters":[{"type":"add_field","config":{"field":"severity","value":"high"}}]}}]'::jsonb, '1.0.0'),

    ('geoip', 'GeoIP Lookup', 'enrichment', 'Enrich IP addresses with geographic information',
     '[{"name":"source_field","type":"string","required":true,"description":"Field containing IP address"},{"name":"target_field","type":"string","required":false,"default":"geoip","description":"Field to store geo data"},{"name":"fields","type":"array","required":false,"description":"Specific geo fields to extract"}]'::jsonb,
     '[{"description":"Basic GeoIP lookup","config":{"source_field":"client_ip","target_field":"geo"}},{"description":"Extract specific fields","config":{"source_field":"ip","fields":["country_name","city_name","latitude","longitude"]}}]'::jsonb, '1.0.0'),

    ('json_validate', 'JSON Validate', 'validation', 'Validate JSON structure and optionally drop invalid records',
     '[{"name":"field","type":"string","required":false,"description":"Field to validate (validates entire record if not specified)"},{"name":"drop_invalid","type":"boolean","required":false,"default":false,"description":"Drop records with invalid JSON"}]'::jsonb,
     '[{"description":"Validate entire record","config":{}},{"description":"Validate specific field and drop invalid","config":{"field":"payload","drop_invalid":true}}]'::jsonb, '1.0.0'),

    ('json_flatten', 'JSON Flatten', 'transformation', 'Flatten nested JSON objects into flat key-value pairs',
     '[{"name":"separator","type":"string","required":false,"default":".","description":"Character to use when joining nested keys"}]'::jsonb,
     '[{"description":"Flatten with dot separator","config":{"separator":"."}},{"description":"Flatten with underscore","config":{"separator":"_"}}]'::jsonb, '1.0.0'),

    ('uppercase_keys', 'Uppercase Keys', 'utility', 'Convert all object keys to uppercase',
     '[{"name":"source_field","type":"array or string","required":false,"description":"Array of fields to operate on, single field name, \"*\" for all fields, or empty/omitted for entire record"},{"name":"recursive","type":"boolean","required":false,"default":true,"description":"Recursively uppercase nested objects"}]'::jsonb,
     '[{"description":"Uppercase all keys in entire record","config":{"recursive":true}},{"description":"Uppercase keys in specific field","config":{"source_field":"metadata","recursive":true}}]'::jsonb, '1.0.0'),

    ('grok', 'Grok Parse', 'parsing', 'Parse unstructured text using grok patterns',
     '[{"name":"source_field","type":"string","required":true,"description":"Field to parse"},{"name":"pattern","type":"string","required":true,"description":"Grok pattern"},{"name":"target_field","type":"string","required":false,"description":"Target field for parsed data"}]'::jsonb,
     '[{"description":"Parse Apache logs","config":{"source_field":"message","pattern":"%{COMBINEDAPACHELOG}"}}]'::jsonb, '1.0.0'),

    ('mutate', 'Mutate', 'transformation', 'Perform various mutations on fields (convert, split, merge, etc)',
     '[{"name":"convert","type":"object","required":false,"description":"Convert field types"},{"name":"split","type":"object","required":false,"description":"Split string fields"},{"name":"merge","type":"object","required":false,"description":"Merge multiple fields"}]'::jsonb,
     '[{"description":"Convert field types","config":{"convert":{"user_id":"integer","active":"boolean"}}}]'::jsonb, '1.0.0'),

    ('drop', 'Drop', 'filtering', 'Drop records that match certain conditions',
     '[{"name":"condition","type":"string","required":false,"description":"Condition for dropping (drops all if not specified)"}]'::jsonb,
     '[{"description":"Drop debug logs","config":{"condition":"level == \"debug\""}}]'::jsonb, '1.0.0'),

    ('kv', 'Key-Value Parse', 'parsing', 'Parse key-value pairs from strings',
     '[{"name":"source_field","type":"string","required":true,"description":"Field containing key-value pairs"},{"name":"field_split","type":"string","required":false,"default":" ","description":"Separator between key-value pairs"},{"name":"value_split","type":"string","required":false,"default":"=","description":"Separator between key and value"}]'::jsonb,
     '[{"description":"Parse space-separated KV pairs","config":{"source_field":"message","field_split":" ","value_split":"="}}]'::jsonb, '1.0.0'),

    ('split', 'Split', 'transformation', 'Split a field into array based on separator',
     '[{"name":"field","type":"string","required":true,"description":"Field to split"},{"name":"separator","type":"string","required":true,"description":"Separator to split on"},{"name":"target","type":"string","required":false,"description":"Target field name"}]'::jsonb,
     '[{"description":"Split comma-separated values","config":{"field":"tags","separator":","}}]'::jsonb, '1.0.0'),

    ('useragent', 'User Agent Parse', 'enrichment', 'Parse user agent strings into structured data',
     '[{"name":"source_field","type":"string","required":true,"description":"Field containing user agent string"},{"name":"target_field","type":"string","required":false,"default":"user_agent","description":"Field to store parsed data"}]'::jsonb,
     '[{"description":"Parse user agent","config":{"source_field":"user_agent_string","target_field":"ua"}}]'::jsonb, '1.0.0'),

    ('dns', 'DNS Lookup', 'enrichment', 'Perform DNS lookups on IP addresses',
     '[{"name":"source_field","type":"string","required":true,"description":"Field containing IP or hostname"},{"name":"target_field","type":"string","required":false,"description":"Field to store DNS result"},{"name":"lookup_type","type":"string","required":false,"default":"reverse","description":"Type of lookup: forward or reverse"}]'::jsonb,
     '[{"description":"Reverse DNS lookup","config":{"source_field":"ip","target_field":"hostname","lookup_type":"reverse"}}]'::jsonb, '1.0.0'),

    ('fingerprint', 'Fingerprint', 'hashing', 'Generate hash fingerprint for records',
     '[{"name":"fields","type":"array","required":false,"description":"Fields to include in fingerprint (all fields if not specified)"},{"name":"target","type":"string","required":false,"default":"fingerprint","description":"Field to store fingerprint"},{"name":"method","type":"string","required":false,"default":"SHA256","description":"Hash method"}]'::jsonb,
     '[{"description":"Fingerprint entire record","config":{"target":"fingerprint","method":"SHA256"}},{"description":"Fingerprint specific fields","config":{"fields":["user_id","timestamp"],"target":"event_id"}}]'::jsonb, '1.0.0'),

    ('include', 'Include', 'filtering', 'Include only records matching conditions',
     '[{"name":"field","type":"string","required":true,"description":"Field to check"},{"name":"match","type":"any","required":true,"description":"Value or pattern to match"}]'::jsonb,
     '[{"description":"Include only error logs","config":{"field":"level","match":"error"}}]'::jsonb, '1.0.0'),

    ('exclude', 'Exclude', 'filtering', 'Exclude records matching conditions',
     '[{"name":"field","type":"string","required":true,"description":"Field to check"},{"name":"match","type":"any","required":true,"description":"Value or pattern to match"}]'::jsonb,
     '[{"description":"Exclude debug logs","config":{"field":"level","match":"debug"}}]'::jsonb, '1.0.0'),

    ('sample', 'Sample', 'filtering', 'Sample a percentage of records',
     '[{"name":"percentage","type":"number","required":true,"description":"Percentage of records to keep (0-100)"}]'::jsonb,
     '[{"description":"Keep 10% of records","config":{"percentage":10}}]'::jsonb, '1.0.0'),

    ('passthrough', 'Passthrough', 'utility', 'Pass records through without modification (for testing)',
     '[]'::jsonb,
     '[{"description":"Passthrough filter","config":{}}]'::jsonb, '1.0.0')

ON CONFLICT (filter_type) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    category = EXCLUDED.category,
    purpose = EXCLUDED.purpose,
    parameters = EXCLUDED.parameters,
    examples = EXCLUDED.examples,
    version = EXCLUDED.version,
    updated_at = NOW();
