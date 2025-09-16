# ByteFreezer Control - AWX/Tower Integration

Complete AWX/Ansible Tower automation suite for ByteFreezer Control deployment and management.

## 🎯 **AWX Components**

### **📋 Job Templates**
- **Local Install** - Deploy from local binary
- **Docker Install** - Deploy from Docker image  
- **GitHub Release Install** - Deploy from GitHub releases
- **Remove Service** - Clean removal with data protection
- **Kubernetes Deploy** - K8s cluster deployment
- **Kubernetes Remove** - K8s service removal

### **🔄 Workflow Templates**
- **Full Production Deployment** - Complete prod workflow with validation/rollback
- **Staging Deployment** - Staging workflow with integration tests
- **Development Auto-Deploy** - Continuous deployment for dev
- **Kubernetes Multi-Environment** - Multi-stage K8s deployment with approval gates
- **Emergency Rollback** - Emergency rollback across all environments

### **📊 Smart Inventories**
- **Production Servers** - Production environment hosts
- **Staging Servers** - Staging environment hosts  
- **Development Servers** - Development environment hosts
- **All Environments** - Cross-environment management
- **Kubernetes Clusters** - K8s deployment targets

## 🚀 **Quick Setup in AWX**

### **1. Import Project**
```bash
# In AWX UI:
# Projects → Add
Name: "ByteFreezer Control"
SCM Type: "Git"
SCM URL: "https://github.com/n0needt0/bytefreezer-control.git"
SCM Branch: "main"
Playbook Directory: "ansible"
```

### **2. Import Job Templates**
```bash
# Method 1: AWX CLI
awx job_templates create --conf.file awx/job_templates.yml

# Method 2: Manual import via UI
# Templates → Job Templates → Add
# Copy configuration from awx/job_templates.yml
```

### **3. Import Workflow Templates**
```bash
# AWX CLI
awx workflow_job_templates create --conf.file awx/workflow_templates.yml

# Or import manually via UI
# Templates → Workflow Job Templates → Add
```

### **4. Configure Inventories**
```bash
# Import smart inventories
awx inventories create --conf.file awx/inventory_sources.yml

# Or create manually using host_filter patterns from inventory_sources.yml
```

## 🎮 **Survey Variables**

### **Common Survey Variables**
All job templates support these survey inputs:

**Environment Selection:**
- `target_environment`: `development` | `staging` | `production`
- `enable_debug_logging`: `true` | `false`

**Deployment Options:**
- `bytefreezer_control_version`: Version to deploy (e.g., `v1.0.0`, `latest`)
- `force_reinstall`: `true` | `false` - Force reinstall over existing

**Removal Options:**
- `remove_data_dir`: `true` | `false` - Remove data directory
- `remove_storage`: `true` | `false` - Remove K8s persistent storage
- `remove_namespace`: `true` | `false` - Remove K8s namespace

**Kubernetes Options:**
- `storage_enabled`: `true` | `false` - Enable persistent storage

## 🏗️ **Workflow Examples**

### **Production Deployment Workflow**
```yaml
Survey Inputs:
- Deployment Version: "v1.2.0"
- Deployment Method: "github_release"
- Enable Rollback on Failure: true

Workflow Steps:
1. Pre-deployment validation → Health checks
2. Deploy v1.2.0 to production servers
3. Post-deployment validation → Health checks
4. On success: Notify success
5. On failure: Auto-rollback + Notify
```

### **Kubernetes Multi-Environment Workflow**
```yaml
Survey Inputs:
- Container Version: "v1.2.0"
- Deploy to Staging: true
- Deploy to Production: true (requires approval)
- Enable Persistent Storage: true

Workflow Steps:
1. Deploy to Dev K8s → Auto-proceed
2. Deploy to Staging K8s → Auto-proceed  
3. Approval Gate → Manual approval required
4. Deploy to Prod K8s → On approval
5. Rollback on failure → Automatic
```

### **Emergency Rollback Workflow**
```yaml
Survey Inputs:
- Target Environment: "production"
- Rollback Version: "v1.1.0"
- Confirm Emergency Rollback: "CONFIRM"

Workflow Steps:
1. Validate confirmation → Check "CONFIRM" input
2. Stop services → Graceful shutdown
3. Rollback deployment → Install v1.1.0
4. Verify rollback → Health checks
5. Notify results → Success/failure alerts
```

## 🔐 **Credentials Configuration**

### **Required Credentials**
Configure these credential types in AWX:

**Linux Servers SSH:**
```yaml
Name: "Linux Servers SSH"
Type: "Machine"
Username: "admin"
SSH Private Key: "[SSH private key content]"
Privilege Escalation Method: "sudo"
```

**Kubernetes Admin:**
```yaml
Name: "Kubernetes Admin"  
Type: "Kubernetes/OpenShift API Bearer Token"
Host: "https://k8s-api.example.com:6443"
Bearer Token: "[K8s service account token]"
Verify SSL: true
```

**GitHub Token (for releases):**
```yaml
Name: "GitHub API Token"
Type: "Personal Access Token"
Token: "[GitHub token with repo access]"
```

### **Vault Integration**
For sensitive variables, use AWX Vault:

```yaml
# In job template extra vars:
config:
  database:
    password: "{{ vault_db_password }}"
  auth:
    jwt_secret: "{{ vault_jwt_secret }}"
```

## 📊 **Smart Inventory Filters**

### **Host Filter Examples**
Use these filters for smart inventories:

**Production Servers:**
```
awx_project:"bytefreezer-control" and group_names:"production"
```

**All ByteFreezer Services:**
```
awx_project:"bytefreezer-control" or awx_project:"bytefreezer-proxy" or awx_project:"bytefreezer-receiver"
```

**Environment-Specific:**
```
target_environment:"staging" and awx_service_type:"control-plane"
```

**Kubernetes Hosts:**
```
ansible_connection:"local" and kubernetes_admin:"true"
```

## 🚨 **Monitoring and Alerting**

### **Job Notifications**
Configure notifications for workflow events:

**Slack Integration:**
```yaml
Notification Type: "Slack"
Channel: "#bytefreezer-deployments"
Events: ["started", "success", "failed", "approval"]
```

**Email Alerts:**
```yaml
Notification Type: "Email"
Recipients: ["ops@company.com", "dev@company.com"]  
Events: ["failed", "approval"]
```

### **Custom Notification Templates**
```yaml
# Success notification
Subject: "✅ ByteFreezer Control {{ job_status }} - {{ awx_job_template_name }}"
Body: |
  Deployment completed successfully!
  
  Environment: {{ target_environment }}
  Version: {{ bytefreezer_control_version }}
  Job: {{ awx_job_url }}
  
# Failure notification  
Subject: "❌ ByteFreezer Control {{ job_status }} - {{ awx_job_template_name }}"
Body: |
  Deployment failed!
  
  Environment: {{ target_environment }}
  Error: {{ job_explanation }}
  Logs: {{ awx_job_url }}
```

## 🔧 **Maintenance and Troubleshooting**

### **Common AWX Issues**

**Survey Variables Not Updating:**
```bash
# Clear project cache
awx projects update <project_id>

# Restart AWX services
kubectl rollout restart deployment/awx-web -n awx
```

**Smart Inventory Empty:**
```bash
# Check host filters
awx inventories list --name "ByteFreezer Control - Production"

# Verify host facts
awx hosts list --inventory <inventory_id>
```

**Workflow Approval Stuck:**
```bash
# Check pending approvals
awx workflow_approvals list --status pending

# Approve manually
awx workflow_approvals approve <approval_id>
```

### **Best Practices**

**Environment Isolation:**
- Use separate inventories per environment
- Environment-specific credentials
- Approval gates for production

**Version Management:**
- Pin versions for production deployments
- Use `latest` only for development
- Maintain rollback capability

**Security:**
- Use Vault for secrets
- Limit credential access by team
- Enable audit logging

## 🎯 **Integration Examples**

### **CI/CD Pipeline Integration**
```bash
# Trigger AWX job from CI/CD
curl -X POST \
  -H "Authorization: Bearer $AWX_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "extra_vars": {
      "bytefreezer_control_version": "'$BUILD_VERSION'",
      "target_environment": "staging"
    }
  }' \
  $AWX_URL/api/v2/job_templates/$JOB_TEMPLATE_ID/launch/
```

### **Monitoring Integration**
```bash
# Prometheus alert → AWX webhook → Emergency rollback
- alert: ByteFreezerControlDown
  expr: up{job="bytefreezer-control"} == 0
  for: 5m
  annotations:
    webhook_url: "$AWX_URL/api/v2/job_templates/$EMERGENCY_ROLLBACK_ID/launch/"
```

This AWX integration provides enterprise-grade automation for ByteFreezer Control with comprehensive deployment workflows, approval processes, and monitoring capabilities! 🚀