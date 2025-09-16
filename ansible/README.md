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