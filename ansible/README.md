# ByteFreezer Control - AWX/Ansible Deployment

Complete AWX/Ansible Tower automation for deploying ByteFreezer Control service across different environments with enterprise-grade workflows and survey integration.

## 🚀 **Quick Start**

### **Local Installation (Binary)**
```bash
# Build the binary first
./build_local.sh

# Deploy locally
cd ansible
ansible-playbook -i inventory.yml playbooks/local_install.yml --limit localhost
```

### **Docker Installation**
```bash
cd ansible
ansible-playbook -i inventory.yml playbooks/docker_install.yml --limit localhost
```

### **GitHub Release Installation**
```bash
cd ansible
ansible-playbook -i inventory.yml playbooks/install.yml --limit localhost
```

### **Kubernetes Deployment**
```bash
cd ansible
ansible-playbook -i inventory.yml playbooks/kubernetes/deploy.yml
```

## 📁 **Structure**

```
ansible/
├── inventory.yml              # AWX-compatible host definitions
├── awx/                       # AWX/Tower specific configurations
│   ├── job_templates.yml      # AWX job template definitions
│   ├── workflow_templates.yml # AWX workflow configurations
│   ├── inventory_sources.yml  # Smart inventory sources
│   └── README.md             # AWX setup and usage guide
├── playbooks/
│   ├── group_vars/all.yml    # AWX survey variables and configuration
│   ├── local_install.yml     # Install from local binary (AWX compatible)
│   ├── docker_install.yml    # Install from Docker image (AWX compatible)
│   ├── install.yml           # Install from GitHub release (AWX compatible)
│   ├── remove.yml            # Uninstall service (AWX compatible)
│   ├── templates/            # Configuration templates
│   │   ├── config.yaml.j2
│   │   ├── bytefreezer-control.service.j2
│   │   └── logrotate.j2
│   └── kubernetes/           # Kubernetes deployment
│       ├── group_vars/all.yml
│       ├── deploy.yml        # K8s deploy (AWX compatible)
│       └── remove.yml        # K8s remove (AWX compatible)
└── README.md                 # This file
```

## ⚙️ **Configuration**

### **Environment Variables**
Override default configuration using host variables in `inventory.yml`:

```yaml
hosts:
  my-server:
    ansible_host: 192.168.1.100
    bytefreezer_control_version: "v1.0.0"
    config:
      server:
        api_port: 8082
      database:
        host: "postgres.internal"
        password: "{{ vault_password }}"
```

### **Service Configuration**
Key configuration options in `group_vars/all.yml`:

- **API Port**: `config.server.api_port` (default: 8082)
- **Database**: Full PostgreSQL configuration
- **Ecosystem Services**: URLs for receiver, proxy, SOC, packer
- **Authentication**: JWT secrets and admin users
- **OpenTelemetry**: Observability configuration
- **Rate Limiting**: Request throttling settings

## 🎯 **Deployment Scenarios**

### **1. Production Deployment**
```bash
# Deploy to production servers
ansible-playbook -i inventory.yml playbooks/install.yml --limit production

# Update configuration only
ansible-playbook -i inventory.yml playbooks/local_install.yml --limit production --tags config
```

### **2. Staging Environment**
```bash
# Deploy latest version to staging
ansible-playbook -i inventory.yml playbooks/docker_install.yml --limit staging
```

### **3. Development Setup**
```bash
# Local development with debug logging
ansible-playbook -i inventory.yml playbooks/local_install.yml --limit development
```

### **4. Kubernetes Cluster**
```bash
# Deploy to K8s with persistent storage
ansible-playbook playbooks/kubernetes/deploy.yml -e storage.enabled=true

# Remove from K8s (preserve data)
ansible-playbook playbooks/kubernetes/remove.yml
```

## 🔧 **Service Management**

### **Status and Control**
```bash
# Check service status
sudo systemctl status bytefreezer-control

# View logs
sudo journalctl -u bytefreezer-control -f

# Restart service
sudo systemctl restart bytefreezer-control
```

### **Configuration Updates**
```bash
# Update configuration and restart
ansible-playbook -i inventory.yml playbooks/local_install.yml --tags config

# Validate configuration
/usr/local/bin/bytefreezer-control --validate-config
```

## 🛡️ **Security**

### **Secrets Management**
Use Ansible Vault for sensitive data:

```bash
# Create vault file
ansible-vault create secrets.yml

# Edit vault
ansible-vault edit secrets.yml

# Deploy with vault
ansible-playbook -i inventory.yml playbooks/install.yml --ask-vault-pass
```

### **Vault Variables**
```yaml
# secrets.yml
vault_db_password_prod: "super-secure-password"
vault_jwt_secret: "jwt-signing-secret"
```

## 📊 **Monitoring and Health**

### **Health Endpoints**
- **Health Check**: `http://server:8082/api/v2/health`
- **Configuration**: `http://server:8082/api/v2/config`
- **Service Status**: `http://server:8082/api/v2/services/status`

### **Log Files**
- **Application**: `/var/log/bytefreezer-control/bytefreezer-control.log`
- **Errors**: `/var/log/bytefreezer-control/bytefreezer-control-error.log`
- **System**: `journalctl -u bytefreezer-control`

### **OpenTelemetry Integration**
```yaml
config:
  otel:
    enabled: true
    endpoint: "http://otel-collector:4317"
    service_name: "bytefreezer-control"
```

## 🔄 **Maintenance**

### **Updates**
```bash
# Update to specific version
ansible-playbook -i inventory.yml playbooks/install.yml -e bytefreezer_control_version=v1.1.0

# Update from Docker (latest)
ansible-playbook -i inventory.yml playbooks/docker_install.yml -e docker.force_pull=true
```

### **Backup**
```bash
# Backup configuration
cp -r /etc/bytefreezer-control /backup/config-$(date +%Y%m%d)

# Backup data (if using local database)
pg_dump bytefreezer_control > /backup/control-db-$(date +%Y%m%d).sql
```

### **Removal**
```bash
# Remove service (preserve data)
ansible-playbook -i inventory.yml playbooks/remove.yml

# Remove everything including data
ansible-playbook -i inventory.yml playbooks/remove.yml -e remove.data_dir=true
```

## 🎭 **Troubleshooting**

### **Common Issues**

**Service Won't Start**
```bash
# Check configuration
/usr/local/bin/bytefreezer-control --validate-config

# Check logs
sudo journalctl -u bytefreezer-control --no-pager
```

**Database Connection Issues**
```bash
# Test database connectivity
psql -h postgres.internal -U bytefreezer -d bytefreezer_control

# Check network policies
telnet postgres.internal 5432
```

**Ecosystem Service Communication**
```bash
# Test service endpoints
curl http://receiver.internal:8080/api/v2/health
curl http://proxy.internal:8088/api/v2/health
```

## 🏗️ **Development**

### **Local Testing**
```bash
# Syntax check
ansible-playbook playbooks/local_install.yml --syntax-check

# Dry run
ansible-playbook -i inventory.yml playbooks/local_install.yml --check --diff

# Run with verbose output
ansible-playbook -i inventory.yml playbooks/local_install.yml -vvv
```

### **Custom Configurations**
Create environment-specific variable files:
```bash
# group_vars/production.yml
config:
  logging:
    level: "warn"
  rate_limit:
    requests_per_minute: 1000
```

## 🎭 **AWX Integration**

### **Job Templates with Surveys**
All playbooks are AWX-compatible with survey variables:

- `target_environment`: Choose development/staging/production
- `force_reinstall`: Override existing installations
- `enable_debug_logging`: Toggle debug mode
- `bytefreezer_control_version`: Specify deployment version

### **Workflow Templates**
Enterprise workflows available:
- **Full Production Deployment** - Multi-stage with approval gates
- **Staging Deployment** - Automated testing integration
- **Emergency Rollback** - Cross-environment rollback capability
- **Kubernetes Multi-Environment** - K8s deployment with approvals

### **Smart Inventories**
Dynamic inventory management:
- Environment-based host filtering
- Service-type grouping
- Automatic host discovery

**Setup:** See `ansible/awx/README.md` for complete AWX integration guide.

This AWX-compatible setup provides enterprise-grade deployment automation for ByteFreezer Control with survey-driven workflows, approval processes, smart inventories, and comprehensive monitoring capabilities! 🚀