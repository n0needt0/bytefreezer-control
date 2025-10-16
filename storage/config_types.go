package storage

import "time"

// TenantConfig represents the rich configuration document for a tenant
type TenantConfig struct {
	Organization   OrganizationConfig `json:"organization"`
	Subscription   SubscriptionConfig `json:"subscription"`
	Notifications  NotificationConfig `json:"notifications"`
	Security       SecurityConfig     `json:"security"`
	Billing        BillingConfig      `json:"billing"`
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

// OrganizationConfig contains organizational information
type OrganizationConfig struct {
	Name        string   `json:"name"`
	Industry    string   `json:"industry,omitempty"`
	Size        string   `json:"size,omitempty"` // small, medium, large, enterprise
	Departments []string `json:"departments,omitempty"`
	Timezone    string   `json:"timezone,omitempty"`
	Country     string   `json:"country,omitempty"`
}

// SubscriptionConfig contains subscription and limits
type SubscriptionConfig struct {
	Tier              string    `json:"tier"`                // basic, pro, enterprise
	MaxDatasets       int       `json:"max_datasets"`
	StorageQuotaGB    int       `json:"storage_quota_gb"`
	ProcessingQuotaMB int       `json:"processing_quota_mb"`
	APIRateLimit      int       `json:"api_rate_limit"`
	Features          []string  `json:"features,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
}

// NotificationConfig contains notification preferences
type NotificationConfig struct {
	Email EmailNotificationConfig `json:"email"`
	SMS   SMSNotificationConfig   `json:"sms"`
	Slack SlackNotificationConfig `json:"slack"`
	Webhook WebhookNotificationConfig `json:"webhook"`
}

type EmailNotificationConfig struct {
	Enabled     bool     `json:"enabled"`
	Addresses   []string `json:"addresses,omitempty"`
	Events      []string `json:"events,omitempty"` // dataset_complete, error, quota_warning
	DailyDigest bool     `json:"daily_digest"`
}

type SMSNotificationConfig struct {
	Enabled bool     `json:"enabled"`
	Numbers []string `json:"numbers,omitempty"`
	Events  []string `json:"events,omitempty"` // critical_errors only
}

type SlackNotificationConfig struct {
	Enabled bool   `json:"enabled"`
	Webhook string `json:"webhook,omitempty"`
	Channel string `json:"channel,omitempty"`
	Events  []string `json:"events,omitempty"`
}

type WebhookNotificationConfig struct {
	Enabled  bool              `json:"enabled"`
	URL      string            `json:"url,omitempty"`
	Secret   string            `json:"secret,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Events   []string          `json:"events,omitempty"`
	Retries  int               `json:"retries"`
	Timeout  int               `json:"timeout_seconds"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	IPWhitelist    []string          `json:"ip_whitelist,omitempty"`
	APIKeyRotation APIKeyRotationConfig `json:"api_key_rotation"`
	DataRetention  DataRetentionConfig  `json:"data_retention"`
	Encryption     EncryptionConfig     `json:"encryption"`
}

type APIKeyRotationConfig struct {
	Enabled        bool `json:"enabled"`
	IntervalDays   int  `json:"interval_days"`
	WarningDays    int  `json:"warning_days"`
	AutoRotate     bool `json:"auto_rotate"`
}

type DataRetentionConfig struct {
	LogRetentionDays    int  `json:"log_retention_days"`
	MetricsRetentionDays int `json:"metrics_retention_days"`
	AutoCleanup         bool `json:"auto_cleanup"`
}

type EncryptionConfig struct {
	AtRest    bool   `json:"at_rest"`
	InTransit bool   `json:"in_transit"`
	Algorithm string `json:"algorithm,omitempty"`
}

// BillingConfig contains billing and usage tracking
type BillingConfig struct {
	Address     BillingAddress `json:"address"`
	PaymentMethod string       `json:"payment_method,omitempty"` // card, invoice, wire
	Currency    string         `json:"currency"`
	TaxID       string         `json:"tax_id,omitempty"`
	Contact     BillingContact `json:"contact"`
}

type BillingAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type BillingContact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

// DatasetConfig represents the rich configuration document for a dataset
type DatasetConfig struct {
	Source      SourceConfig      `json:"source"`
	Processing  ProcessingConfig  `json:"processing"`
	Destination DestinationConfig `json:"destination"`
	Schedule    ScheduleConfig    `json:"schedule"`
	Monitoring  MonitoringConfig  `json:"monitoring"`
	Transform   TransformConfig   `json:"transform"`
	Parquet        map[string]interface{} `json:"parquet,omitempty"` // Parquet-specific configuration (metadata_level, partition_layout, etc.)
	Custom         map[string]interface{} `json:"custom,omitempty"` // Root-level custom config
	CustomPipeline map[string]interface{} `json:"custom_pipeline,omitempty"`
}

// SourceConfig defines where data comes from
type SourceConfig struct {
	Type       string                 `json:"type"` // api, file, database, stream
	Connection ConnectionConfig       `json:"connection"`
	Format     string                 `json:"format,omitempty"` // json, csv, parquet, avro
	Schema     SchemaConfig           `json:"schema,omitempty"`
	Validation ValidationConfig       `json:"validation"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

type ConnectionConfig struct {
	URL         string            `json:"url,omitempty"`
	Bucket      string            `json:"bucket,omitempty"`
	Region      string            `json:"region,omitempty"`
	Endpoint    string            `json:"endpoint,omitempty"`
	SSL         bool              `json:"ssl,omitempty"`
	Prefix      string            `json:"prefix,omitempty"`
	Credentials CredentialConfig  `json:"credentials,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     int               `json:"timeout_seconds"`
	Retries     int               `json:"retries"`
}

type CredentialConfig struct {
	Type      string `json:"type"` // bearer, basic, api_key, oauth, s3
	Token     string `json:"token,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	AccessKey string `json:"access_key,omitempty"` // S3 access key
	SecretKey string `json:"secret_key,omitempty"` // S3 secret key
}

type SchemaConfig struct {
	Strict   bool                   `json:"strict"`
	Fields   []SchemaField          `json:"fields,omitempty"`
	Custom   map[string]interface{} `json:"custom,omitempty"`
}

type SchemaField struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // string, integer, float, boolean, timestamp
	Required bool   `json:"required"`
	Default  interface{} `json:"default,omitempty"`
}

type ValidationConfig struct {
	Enabled      bool     `json:"enabled"`
	Rules        []string `json:"rules,omitempty"`
	FailOnError  bool     `json:"fail_on_error"`
	SampleRate   float64  `json:"sample_rate"` // 0.0 to 1.0
}

// ProcessingConfig defines how data is processed
type ProcessingConfig struct {
	Enabled    bool              `json:"enabled"`
	Transforms []TransformStep   `json:"transforms,omitempty"`
	Filters    []FilterStep      `json:"filters,omitempty"`
	Enrichment EnrichmentConfig  `json:"enrichment"`
	Batching   BatchingConfig    `json:"batching"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

type TransformStep struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"` // map, reduce, filter, aggregate
	Config map[string]interface{} `json:"config"`
	Order  int                    `json:"order"`
}

type FilterStep struct {
	Name      string                 `json:"name"`
	Condition string                 `json:"condition"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

type EnrichmentConfig struct {
	Enabled   bool                   `json:"enabled"`
	Sources   []string               `json:"sources,omitempty"` // ip_geo, user_agent, etc.
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

type BatchingConfig struct {
	Enabled   bool `json:"enabled"`
	Size      int  `json:"size"`       // records per batch
	TimeoutMS int  `json:"timeout_ms"` // max wait time
}

// DestinationConfig defines where processed data goes
type DestinationConfig struct {
	Type       string                 `json:"type"` // s3, database, api, file
	Connection ConnectionConfig       `json:"connection"`
	Format     string                 `json:"format,omitempty"`
	Partitioning PartitioningConfig   `json:"partitioning,omitempty"`
	Compression string                `json:"compression,omitempty"` // gzip, snappy, lz4
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

type PartitioningConfig struct {
	Enabled bool     `json:"enabled"`
	Fields  []string `json:"fields,omitempty"` // date, tenant_id, etc.
	Strategy string  `json:"strategy,omitempty"` // daily, hourly, by_size
}

// ScheduleConfig defines when processing runs
type ScheduleConfig struct {
	Enabled   bool   `json:"enabled"`
	Type      string `json:"type"` // cron, interval, event_driven
	Cron      string `json:"cron,omitempty"`
	Interval  int    `json:"interval_seconds,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Retries   int    `json:"retries"`
	Timeout   int    `json:"timeout_seconds"`
}

// MonitoringConfig defines monitoring and alerting
type MonitoringConfig struct {
	Enabled     bool                `json:"enabled"`
	Metrics     MetricsConfig       `json:"metrics"`
	Alerting    AlertingConfig      `json:"alerting"`
	Logging     LoggingConfig       `json:"logging"`
	HealthCheck HealthCheckConfig   `json:"health_check"`
}

type MetricsConfig struct {
	Enabled     bool     `json:"enabled"`
	Collectors  []string `json:"collectors,omitempty"` // prometheus, statsd, custom
	SampleRate  float64  `json:"sample_rate"`
	Retention   int      `json:"retention_days"`
}

type AlertingConfig struct {
	Enabled    bool          `json:"enabled"`
	Thresholds []Threshold   `json:"thresholds,omitempty"`
}

type Threshold struct {
	Metric    string  `json:"metric"`    // error_rate, latency, throughput
	Operator  string  `json:"operator"`  // gt, lt, eq
	Value     float64 `json:"value"`
	Severity  string  `json:"severity"`  // info, warning, critical
}

type LoggingConfig struct {
	Level       string `json:"level"`        // debug, info, warn, error
	Structured  bool   `json:"structured"`
	Retention   int    `json:"retention_days"`
	SampleRate  float64 `json:"sample_rate"`
}

type HealthCheckConfig struct {
	Enabled      bool `json:"enabled"`
	IntervalSec  int  `json:"interval_seconds"`
	TimeoutSec   int  `json:"timeout_seconds"`
	FailureThreshold int `json:"failure_threshold"`
}

// TransformConfig defines data transformation rules
type TransformConfig struct {
	Enabled bool                   `json:"enabled"`
	Rules   []TransformRule        `json:"rules,omitempty"`
	Custom  map[string]interface{} `json:"custom,omitempty"`
}

type TransformRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // field_map, value_map, regex, function
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
}

// Helper functions for default configurations

// getDefaultTenantConfig returns a sensible default configuration for new tenants
func getDefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Organization: OrganizationConfig{
			Name:     "New Organization",
			Size:     "small",
			Timezone: "UTC",
		},
		Subscription: SubscriptionConfig{
			Tier:              "basic",
			MaxDatasets:       10,
			StorageQuotaGB:    5,
			ProcessingQuotaMB: 100,
			APIRateLimit:      1000,
			Features:          []string{"basic_processing", "api_access"},
		},
		Notifications: NotificationConfig{
			Email: EmailNotificationConfig{
				Enabled:     true,
				Events:      []string{"dataset_complete", "error"},
				DailyDigest: false,
			},
			SMS: SMSNotificationConfig{
				Enabled: false,
			},
			Slack: SlackNotificationConfig{
				Enabled: false,
			},
			Webhook: WebhookNotificationConfig{
				Enabled: false,
			},
		},
		Security: SecurityConfig{
			IPWhitelist: []string{},
			APIKeyRotation: APIKeyRotationConfig{
				Enabled:      false,
				IntervalDays: 90,
				WarningDays:  7,
				AutoRotate:   false,
			},
			DataRetention: DataRetentionConfig{
				LogRetentionDays:     30,
				MetricsRetentionDays: 90,
				AutoCleanup:          true,
			},
			Encryption: EncryptionConfig{
				AtRest:    true,
				InTransit: true,
				Algorithm: "aes256",
			},
		},
		Billing: BillingConfig{
			Address: BillingAddress{
				Country: "US",
			},
			PaymentMethod: "invoice",
			Currency:      "USD",
			Contact: BillingContact{
				Name:  "Billing Contact",
				Email: "billing@example.com",
			},
		},
		CustomSettings: make(map[string]interface{}),
	}
}

// getDefaultDatasetConfig returns a sensible default configuration for new datasets
func getDefaultDatasetConfig() DatasetConfig {
	return DatasetConfig{
		Source: SourceConfig{
			Type: "api",
			Connection: ConnectionConfig{
				Timeout: 30,
				Retries: 3,
			},
			Format: "json",
			Validation: ValidationConfig{
				Enabled:     true,
				FailOnError: false,
				SampleRate:  0.1,
			},
		},
		Processing: ProcessingConfig{
			Enabled: true,
			Batching: BatchingConfig{
				Enabled:   true,
				Size:      100,
				TimeoutMS: 5000,
			},
			Enrichment: EnrichmentConfig{
				Enabled: false,
			},
		},
		Destination: DestinationConfig{
			Type:   "s3",
			Format: "json",
			Partitioning: PartitioningConfig{
				Enabled:  true,
				Fields:   []string{"date"},
				Strategy: "daily",
			},
		},
		Schedule: ScheduleConfig{
			Enabled:  false,
			Type:     "interval",
			Interval: 3600, // 1 hour
			Timezone: "UTC",
			Retries:  3,
			Timeout:  300,
		},
		Monitoring: MonitoringConfig{
			Enabled: true,
			Metrics: MetricsConfig{
				Enabled:    true,
				SampleRate: 1.0,
				Retention:  30,
			},
			Alerting: AlertingConfig{
				Enabled: false,
			},
			Logging: LoggingConfig{
				Level:      "info",
				Structured: true,
				Retention:  7,
				SampleRate: 1.0,
			},
			HealthCheck: HealthCheckConfig{
				Enabled:          true,
				IntervalSec:      60,
				TimeoutSec:       10,
				FailureThreshold: 3,
			},
		},
		Transform: TransformConfig{
			Enabled: false,
		},
		CustomPipeline: make(map[string]interface{}),
	}
}