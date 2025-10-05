# ByteFreezer Control - Ansible Playbooks

Collection of Ansible playbooks for ByteFreezer Control deployment and management.

## 📁 **Available Playbooks**

### **install.yml**
Installs ByteFreezer Control from GitHub releases
- Downloads and installs specified version
- Creates systemd service
- Configures logging and directories

### **local_install.yml**  
Installs ByteFreezer Control from local binary
- Uses binary from `ansible/playbooks/dist/`
- Requires running `./build_local.sh` first
- Ideal for development and testing

### **docker_install.yml**
Installs ByteFreezer Control from Docker image
- Extracts binary from container
- Always gets latest image
- Good for staging environments

### **remove.yml**
Uninstalls ByteFreezer Control service
- Stops and disables service
- Removes binary and configuration
- Optional data cleanup

### **migrate.yml**
Database migration management
- Schema versioning and upgrades
- Fresh install vs upgrade modes
- Data preservation options
- Rollback capabilities

## 📁 **Configuration**

### **group_vars/all.yml**
Contains all configuration variables including:
- Server settings (API port, logging)
- Service paths and directories  
- Environment-specific overrides
- Template variables

### **templates/**
- **config.yaml.j2** - Main configuration template
- **bytefreezer-control.service.j2** - Systemd service template  
- **logrotate.j2** - Log rotation configuration

## 🗄️ **Database Migration Options**

ByteFreezer Control includes a robust migration system for database schema management:

### **Migration Modes**
- **`upgrade`** (default) - Apply new migrations only, preserve data
- **`fresh_install`** - Drop and recreate all tables (new deployment)
- **`reset_data`** - Clear data but preserve schema (testing/development)
- **`rollback`** - Rollback to specific version

### **Migration Variables**
Configure in AWX surveys or via `-e` flags:
- **`migration_mode`** - Controls migration behavior
- **`migration_target_version`** - Target version (0 = latest)
- **`migration_dry_run`** - Preview changes without applying

### **Migration Examples**
```bash
# Standard upgrade (preserves data)
ansible-playbook playbooks/migrate.yml -e migration_mode=upgrade

# Fresh installation  
ansible-playbook playbooks/migrate.yml -e migration_mode=fresh_install

# Reset data for testing
ansible-playbook playbooks/migrate.yml -e migration_mode=reset_data

# Rollback to version 1
ansible-playbook playbooks/migrate.yml -e migration_mode=rollback -e migration_target_version=1

# Preview changes
ansible-playbook playbooks/migrate.yml -e migration_dry_run=true
```

### **Deployment Scenarios**
- **New installation:** migration_mode=fresh_install
- **Code update only:** migration_mode=upgrade (default)
- **Reset for testing:** migration_mode=reset_data
- **Fix schema issues:** migration_mode=rollback