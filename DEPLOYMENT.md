# Deployment Guide

This guide covers deployment options for the World of Wisdom project.

## Prerequisites

- Docker and Docker Compose installed on target servers
- SSH access to VPS instances
- Git installed on deployment targets

## GitHub Actions CI/CD

### Setup GitHub Secrets

Configure the following secrets in your GitHub repository settings:

#### Staging Environment
- `STAGING_SSH_KEY`: Private SSH key for staging server
- `STAGING_USER`: SSH username for staging server
- `STAGING_HOST`: Staging server hostname/IP

#### Production Environment (VPS)
- `VPS_1_SSH_KEY`: Private SSH key for VPS 1
- `VPS_1_USER`: SSH username for VPS 1
- `VPS_1_HOST`: VPS 1 hostname
- `VPS_1_PUBLIC_IP`: VPS 1 public IP address

- `VPS_2_SSH_KEY`: Private SSH key for VPS 2
- `VPS_2_USER`: SSH username for VPS 2
- `VPS_2_HOST`: VPS 2 hostname
- `VPS_2_PUBLIC_IP`: VPS 2 public IP address

### Workflow Triggers

- **Push to `main`**: Deploys to production VPS instances
- **Push to `dev`**: Deploys to staging environment
- **Pull Request**: Runs tests and security scans only
- **Manual trigger**: Can be triggered manually via GitHub Actions UI

### Pipeline Stages

1. **Test**: Runs Go tests, race detector, formatting checks, and vet
2. **Build**: Builds and pushes Docker images to GitHub Container Registry
3. **Deploy**: Deploys to appropriate environment based on branch
4. **Security**: Runs Trivy vulnerability scans on PRs
5. **Performance**: Runs load tests against staging
6. **Notify**: Sends deployment notifications

## Manual VPS Deployment

### Using the Deployment Script

The `scripts/deploy-vps.sh` script provides automated VPS deployment:

```bash
# Basic deployment
./scripts/deploy-vps.sh 198.51.100.42

# Deployment with custom SSH key and user
./scripts/deploy-vps.sh -u deployer -k ~/.ssh/vps_key 198.51.100.42

# Deploy microservices architecture
./scripts/deploy-vps.sh --microservices -b dev 198.51.100.42

# Deploy with cleanup of old resources
./scripts/deploy-vps.sh --cleanup 198.51.100.42
```

### Script Options

- `-u, --user USER`: SSH user (default: root)
- `-p, --port PORT`: SSH port (default: 22)
- `-b, --branch BRANCH`: Git branch to deploy (default: main)
- `-k, --key SSH_KEY`: SSH private key file path
- `-c, --cleanup`: Clean up old Docker resources
- `--no-backup`: Skip database backup
- `--microservices`: Deploy with microservices architecture

### Manual Steps

If you prefer manual deployment:

1. **Prepare VPS**:
   ```bash
   # Install Docker
   curl -fsSL https://get.docker.com -o get-docker.sh
   sh get-docker.sh
   
   # Install Docker Compose
   curl -L "https://github.com/docker/compose/releases/download/v2.23.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   chmod +x /usr/local/bin/docker-compose
   ```

2. **Clone Repository**:
   ```bash
   git clone https://github.com/username/world-of-wisdom.git /opt/world-of-wisdom
   cd /opt/world-of-wisdom
   ```

3. **Configure Environment**:
   ```bash
   # Create .env file for VPS configuration
   cat > .env << EOF
   PUBLIC_IP=YOUR_VPS_PUBLIC_IP
   HOST_IP=0.0.0.0
   POSTGRES_PASSWORD=wisdom123
   ALGORITHM=argon2
   DIFFICULTY=2
   EOF
   ```

4. **Deploy Application**:
   ```bash
   # Standard deployment
   docker-compose up -d
   
   # Microservices deployment
   docker-compose -f docker-compose.yml -f docker-compose.microservices.yml up -d
   ```

5. **Configure Firewall**:
   ```bash
   # Allow required ports
   ufw allow 22      # SSH
   ufw allow 3000    # Web UI
   ufw allow 8080    # TCP Server
   ufw allow 8082    # API Server
   ufw --force enable
   ```

## Service Ports

| Service | Port | Description |
|---------|------|-------------|
| Web UI | 3000 | React frontend |
| TCP Server | 8080 | Main PoW server |
| WebServer | 8081 | WebSocket API |
| API Server | 8082 | REST API |
| Service Registry | 8083 | Service discovery |
| Gateway | 8084 | Load balancer |
| Monitor | 8085 | Health monitoring |
| PostgreSQL | 5432 | Database (internal) |
| Redis | 6379 | Cache (internal) |

## Monitoring and Maintenance

### Health Checks

- Web UI: `http://YOUR_IP:3000`
- API Health: `http://YOUR_IP:8082/health`
- Service Registry: `http://YOUR_IP:8083/health`
- System Overview: `http://YOUR_IP:8085/system`

### Viewing Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f server

# Follow logs with timestamps
docker-compose logs -f -t
```

### Updating Deployment

```bash
# Pull latest changes
git pull origin main

# Update and restart services
docker-compose pull
docker-compose up -d --remove-orphans
```

### Backup and Recovery

```bash
# Backup volumes
docker run --rm -v world-of-wisdom_postgres_data:/source -v $(pwd)/backup:/backup alpine tar czf /backup/postgres_$(date +%Y%m%d).tar.gz -C /source .

# Restore volumes
docker run --rm -v world-of-wisdom_postgres_data:/target -v $(pwd)/backup:/backup alpine tar xzf /backup/postgres_YYYYMMDD.tar.gz -C /target
```

## Architecture Comparison

### Standard Deployment
- Single compose file: `docker-compose.yml`
- 5 services: server, webserver, apiserver, web, databases
- Simpler setup, suitable for single VPS

### Microservices Deployment
- Extended with: `docker-compose.microservices.yml`
- 8+ services: adds service-registry, gateway, monitor
- Better for multi-VPS deployments with load balancing
- Includes health monitoring and service discovery

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3000, 8080-8085 are available
2. **Memory issues**: Ensure VPS has at least 2GB RAM for full deployment
3. **Permission errors**: Run Docker commands as root or add user to docker group
4. **Network issues**: Check firewall settings and security groups

### Debug Commands

```bash
# Check service status
docker-compose ps

# Check resource usage
docker stats

# Check networks
docker network ls

# Check volumes
docker volume ls

# Test TCP server connection
telnet YOUR_IP 8080
```

### Recovery Steps

```bash
# Complete reset
docker-compose down -v
docker system prune -f
git pull origin main
docker-compose build --no-cache
docker-compose up -d
```

## Security Considerations

- Use non-root users where possible
- Configure firewall to limit exposed ports
- Use strong passwords for database services
- Regularly update base images and dependencies
- Monitor logs for suspicious activity
- Use HTTPS in production (add reverse proxy like nginx)

## Performance Tuning

- Adjust `DIFFICULTY` based on expected load
- Scale client containers for testing: `docker-compose up --scale client1=5`
- Monitor resource usage with `docker stats`
- Use `--restart unless-stopped` for production services
- Consider using Docker secrets for sensitive data

## Support

For issues with deployment:
1. Check the GitHub Actions logs for automated deployments
2. Review service logs with `docker-compose logs`
3. Verify all health check endpoints are responding
4. Ensure all required environment variables are set