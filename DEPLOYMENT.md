# Deployment Guide - Espelho de São Miguel

This guide provides comprehensive deployment instructions for Espelho de São Miguel Nostr relay aggregator in production environments.

## Prerequisites

- Linux server (Ubuntu 20.04+ recommended)
- Docker and Docker Compose installed
- Domain name pointing to your server
- Basic knowledge of Linux administration

## Quick Deployment

### Option 1: Docker Compose (Recommended)

1. **Download and extract the release:**
   ```bash
   wget https://github.com/girino/saint-michaels-mirror/releases/latest/download/saint-michaels-mirror-latest-complete.tar.gz
   tar -xzf saint-michaels-mirror-latest-complete.tar.gz
   cd saint-michaels-mirror-*/
   ```

2. **Configure your relay:**
   ```bash
   cp example.env .env
   nano .env  # Edit with your configuration
   ```

3. **Deploy:**
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```

### Option 2: Standalone Binary

1. **Download and extract the release:**
   ```bash
   wget https://github.com/girino/saint-michaels-mirror/releases/latest/download/saint-michaels-mirror-latest-complete.tar.gz
   tar -xzf saint-michaels-mirror-latest-complete.tar.gz
   cd saint-michaels-mirror-*/
   ```

2. **Set environment variables:**
   ```bash
   export RELAY_NAME="Your Relay Name"
   export RELAY_DESCRIPTION="A Nostr relay aggregator"
   export PUBLISH_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
   export QUERY_REMOTES="wss://relay1.example.com,wss://relay2.example.com"
   ```

3. **Run:**
   ```bash
   chmod +x saint-michaels-mirror-linux-amd64
   ./saint-michaels-mirror-linux-amd64 --addr=:3337
   ```

## Production Setup with Nginx

### 1. Install Nginx

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nginx

# CentOS/RHEL
sudo yum install nginx
# or
sudo dnf install nginx
```

### 2. Configure Nginx

1. **Copy the example configuration:**
   ```bash
   sudo cp nginx.conf.example /etc/nginx/sites-available/your-relay
   sudo ln -s /etc/nginx/sites-available/your-relay /etc/nginx/sites-enabled/
   ```

2. **Edit the configuration:**
   ```bash
   sudo nano /etc/nginx/sites-available/your-relay
   ```
   
   Replace `your-relay-domain.com` with your actual domain name.

3. **Test and reload nginx:**
   ```bash
   sudo nginx -t
   sudo systemctl reload nginx
   ```

### 3. Set up SSL with Certbot

1. **Install Certbot:**
   ```bash
   # Ubuntu/Debian
   sudo apt install certbot python3-certbot-nginx
   
   # CentOS/RHEL
   sudo yum install certbot python3-certbot-nginx
   ```

2. **Obtain SSL certificate:**
   ```bash
   sudo certbot --nginx -d your-relay-domain.com
   ```

3. **Verify auto-renewal:**
   ```bash
   sudo certbot renew --dry-run
   ```

### 4. Configure Firewall

```bash
# Ubuntu (ufw)
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

## Systemd Service (for Standalone Binary)

### 1. Create Service File

```bash
sudo nano /etc/systemd/system/saint-michaels-mirror.service
```

### 2. Service Configuration

```ini
[Unit]
Description=Espelho de São Miguel Nostr Relay
After=network.target

[Service]
Type=simple
User=relay
Group=relay
WorkingDirectory=/opt/saint-michaels-mirror
ExecStart=/opt/saint-michaels-mirror/saint-michaels-mirror-linux-amd64 --addr=:3337
Restart=always
RestartSec=5
EnvironmentFile=/opt/saint-michaels-mirror/.env

[Install]
WantedBy=multi-user.target
```

### 3. Enable and Start Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable saint-michaels-mirror
sudo systemctl start saint-michaels-mirror
```

## Monitoring and Maintenance

### Health Checks

Monitor your relay health:

```bash
# Check service status
sudo systemctl status saint-michaels-mirror

# Check Docker containers
docker ps
docker logs saint-michaels-mirror

# Check relay health endpoint
curl http://localhost:3337/api/v1/health
```

### Log Management

```bash
# View service logs
sudo journalctl -u saint-michaels-mirror -f

# View Docker logs
docker logs -f saint-michaels-mirror

# Rotate logs
sudo logrotate -f /etc/logrotate.conf
```

### Backup Strategy

1. **Configuration backup:**
   ```bash
   tar -czf relay-config-backup-$(date +%Y%m%d).tar.gz .env docker-compose.prod.yml
   ```

2. **No persistent database:**
   Espelho de São Miguel is a relay aggregator that forwards events to remote relays. It does not store events locally, so no database backup is needed.

## Performance Tuning

### Nginx Optimization

Add to your nginx configuration:

```nginx
# Worker processes
worker_processes auto;

# Events
events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

# HTTP settings
http {
    # Buffer sizes
    client_body_buffer_size 128k;
    client_header_buffer_size 1k;
    large_client_header_buffers 4 4k;
    
    # Timeouts
    keepalive_timeout 65;
    keepalive_requests 100;
}
```

### System Optimization

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Kernel parameters
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65536" >> /etc/sysctl.conf
sysctl -p
```

## Troubleshooting

### Common Issues

1. **WebSocket connections failing:**
   - Check nginx proxy settings
   - Verify WebSocket headers are set correctly
   - Check firewall settings

2. **High memory usage:**
   - Monitor goroutine counts in `/stats`
   - Check for memory leaks
   - Consider restarting the service periodically

3. **Connection timeouts:**
   - Increase nginx proxy timeouts
   - Check network connectivity to remote relays
   - Verify remote relay availability

### Debug Commands

```bash
# Check relay statistics
curl http://localhost:3337/api/v1/stats | jq

# Check relay health
curl http://localhost:3337/api/v1/health | jq

# Test WebSocket connection
wscat -c ws://localhost:3337

# Test with nak (fiatjaf's Nostr client)
nak req -k 1 -limit 5 ws://localhost:3337

# Monitor network connections
netstat -tulpn | grep :3337
ss -tulpn | grep :3337
```

## Security Considerations

1. **Keep the system updated:**
   ```bash
   sudo apt update && sudo apt upgrade
   ```

2. **Use strong passwords and SSH keys**

3. **Monitor logs for suspicious activity**

4. **Regular security scans:**
   ```bash
   # Using the built-in security workflow
   # Check GitHub Actions security scan results
   ```

5. **Backup and disaster recovery plan**

## Support

- **Repository**: https://gitworkshop.dev/npub18lav8fkgt8424rxamvk8qq4xuy9n8mltjtgztv2w44hc5tt9vets0hcfsz/relay.ngit.dev/saint-michaels-mirror
- **Clone URL**: nostr://npub18lav8fkgt8424rxamvk8qq4xuy9n8mltjtgztv2w44hc5tt9vets0hcfsz/relay.ngit.dev/saint-michaels-mirror
- **Releases**: [GitHub Releases](https://github.com/girino/saint-michaels-mirror/releases) (binaries and Docker images)
- **Community**: Join Nostr community discussions

---

**Espelho de São Miguel** - The mirror that returns light as truth.
