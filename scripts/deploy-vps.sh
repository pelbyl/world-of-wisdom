#!/bin/bash

# VPS Deployment Script for World of Wisdom
# This script deploys the application to VPS instances with public IP addresses

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="world-of-wisdom"
DEPLOY_DIR="/opt/${PROJECT_NAME}"
COMPOSE_FILE="docker-compose.yml"

# Default values
VPS_USER="root"
VPS_PORT="22"
BRANCH="main"
CLEANUP=false
BACKUP=true

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] VPS_IP

Deploy World of Wisdom to VPS with public IP address.

OPTIONS:
    -u, --user USER         SSH user (default: root)
    -p, --port PORT         SSH port (default: 22)
    -b, --branch BRANCH     Git branch to deploy (default: main)
    -k, --key SSH_KEY       SSH private key file path
    -c, --cleanup           Clean up old Docker resources
    --no-backup            Skip database backup
    --microservices        Deploy with microservices architecture
    -h, --help             Show this help message

EXAMPLES:
    # Deploy to VPS with basic setup
    $0 198.51.100.42

    # Deploy with custom user and SSH key
    $0 -u deployer -k ~/.ssh/vps_key 198.51.100.42

    # Deploy microservices architecture
    $0 --microservices -b dev 198.51.100.42

ENVIRONMENT VARIABLES:
    VPS_SSH_KEY            Path to SSH private key
    PUBLIC_IP              Override detected public IP
    
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--user)
            VPS_USER="$2"
            shift 2
            ;;
        -p|--port)
            VPS_PORT="$2"
            shift 2
            ;;
        -b|--branch)
            BRANCH="$2"
            shift 2
            ;;
        -k|--key)
            SSH_KEY="$2"
            shift 2
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        --no-backup)
            BACKUP=false
            shift
            ;;
        --microservices)
            COMPOSE_FILE="docker-compose.yml -f docker-compose.microservices.yml"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            if [[ -z "$VPS_IP" ]]; then
                VPS_IP="$1"
            else
                log_error "Unknown option: $1"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate required arguments
if [[ -z "$VPS_IP" ]]; then
    log_error "VPS IP address is required"
    usage
    exit 1
fi

# Set SSH key from environment if not provided
if [[ -z "$SSH_KEY" && -n "$VPS_SSH_KEY" ]]; then
    SSH_KEY="$VPS_SSH_KEY"
fi

# Build SSH command
SSH_CMD="ssh"
if [[ -n "$SSH_KEY" ]]; then
    SSH_CMD="$SSH_CMD -i $SSH_KEY"
fi
SSH_CMD="$SSH_CMD -p $VPS_PORT -o StrictHostKeyChecking=no $VPS_USER@$VPS_IP"

log_info "Starting deployment to VPS: $VPS_IP"
log_info "User: $VPS_USER, Port: $VPS_PORT, Branch: $BRANCH"

# Test SSH connection
log_info "Testing SSH connection..."
if ! $SSH_CMD "echo 'SSH connection successful'"; then
    log_error "Failed to connect to VPS. Please check your SSH configuration."
    exit 1
fi
log_success "SSH connection established"

# Prepare deployment directory
log_info "Preparing deployment directory..."
$SSH_CMD "
    # Create deployment directory if it doesn't exist
    mkdir -p $DEPLOY_DIR
    cd $DEPLOY_DIR
    
    # Install dependencies if needed
    if ! command -v docker &> /dev/null; then
        echo 'Installing Docker...'
        curl -fsSL https://get.docker.com -o get-docker.sh
        sh get-docker.sh
        rm get-docker.sh
        systemctl enable docker
        systemctl start docker
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo 'Installing Docker Compose...'
        curl -L \"https://github.com/docker/compose/releases/download/v2.23.0/docker-compose-\$(uname -s)-\$(uname -m)\" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
    fi
    
    if ! command -v git &> /dev/null; then
        echo 'Installing Git...'
        apt-get update && apt-get install -y git
    fi
"

# Clone or update repository
log_info "Updating application code..."
$SSH_CMD "
    cd $DEPLOY_DIR
    
    if [ -d '.git' ]; then
        echo 'Updating existing repository...'
        git fetch origin
        git checkout $BRANCH
        git pull origin $BRANCH
    else
        echo 'Cloning repository...'
        # Replace with your actual repository URL
        git clone https://github.com/username/world-of-wisdom.git .
        git checkout $BRANCH
    fi
"

# Backup existing data if requested
if [ "$BACKUP" = true ]; then
    log_info "Creating backup..."
    $SSH_CMD "
        cd $DEPLOY_DIR
        
        # Create backup directory
        BACKUP_DIR=\"backups/\$(date +%Y%m%d_%H%M%S)\"
        mkdir -p \$BACKUP_DIR
        
        # Backup volumes if they exist
        if docker volume ls | grep -q 'postgres_data'; then
            echo 'Backing up PostgreSQL data...'
            docker run --rm -v world-of-wisdom_postgres_data:/source -v \$(pwd)/\$BACKUP_DIR:/backup alpine tar czf /backup/postgres_data.tar.gz -C /source .
        fi
        
        if docker volume ls | grep -q 'redis_data'; then
            echo 'Backing up Redis data...'
            docker run --rm -v world-of-wisdom_redis_data:/source -v \$(pwd)/\$BACKUP_DIR:/backup alpine tar czf /backup/redis_data.tar.gz -C /source .
        fi
        
        echo 'Backup completed in \$BACKUP_DIR'
    "
fi

# Set environment variables for public IP deployment
log_info "Configuring environment for public IP access..."
PUBLIC_IP_TO_USE="${PUBLIC_IP:-$VPS_IP}"

$SSH_CMD "
    cd $DEPLOY_DIR
    
    # Create environment file for public IP configuration
    cat > .env.vps << EOF
# VPS Configuration
PUBLIC_IP=$PUBLIC_IP_TO_USE
HOST_IP=0.0.0.0

# Database Configuration
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=wisdom
POSTGRES_PASSWORD=wisdom123
POSTGRES_DB=wisdom

REDIS_HOST=redis
REDIS_PORT=6379

# Application Configuration
ALGORITHM=argon2
DIFFICULTY=2
ADAPTIVE_MODE=true

# Service Registry
SERVICE_REGISTRY_URL=http://service-registry:8083

# External Access
CORS_ORIGINS=http://$PUBLIC_IP_TO_USE:3000,http://localhost:3000
EOF
"

# Cleanup old resources if requested
if [ "$CLEANUP" = true ]; then
    log_info "Cleaning up old Docker resources..."
    $SSH_CMD "
        cd $DEPLOY_DIR
        docker-compose -f $COMPOSE_FILE down -v --remove-orphans || true
        docker system prune -f
        docker volume prune -f
    "
fi

# Build and deploy
log_info "Building and deploying application..."
$SSH_CMD "
    cd $DEPLOY_DIR
    
    # Load environment variables
    export \$(cat .env.vps | xargs)
    
    # Build images
    echo 'Building Docker images...'
    docker-compose -f $COMPOSE_FILE build --no-cache
    
    # Start services
    echo 'Starting services...'
    docker-compose -f $COMPOSE_FILE up -d --remove-orphans
    
    # Wait for services to be ready
    echo 'Waiting for services to start...'
    sleep 30
"

# Configure firewall
log_info "Configuring firewall..."
$SSH_CMD "
    # Configure UFW if available
    if command -v ufw &> /dev/null; then
        ufw allow 22/tcp      # SSH
        ufw allow 3000/tcp    # Web UI
        ufw allow 8080/tcp    # TCP Server
        ufw allow 8081/tcp    # WebServer
        ufw allow 8082/tcp    # API Server
        ufw allow 8083/tcp    # Service Registry
        ufw allow 8084/tcp    # Gateway
        ufw allow 8085/tcp    # Monitor
        ufw --force enable
    else
        echo 'UFW not available, please configure firewall manually'
    fi
"

# Health checks
log_info "Performing health checks..."
sleep 10

HEALTH_FAILED=false

# Check web interface
if curl -f -s "http://$VPS_IP:3000" > /dev/null; then
    log_success "Web UI is accessible at http://$VPS_IP:3000"
else
    log_warning "Web UI health check failed"
    HEALTH_FAILED=true
fi

# Check API server
if curl -f -s "http://$VPS_IP:8082/health" > /dev/null; then
    log_success "API Server is healthy"
else
    log_warning "API Server health check failed"
    HEALTH_FAILED=true
fi

# Check TCP server port
if timeout 5 bash -c "</dev/tcp/$VPS_IP/8080"; then
    log_success "TCP Server is listening on port 8080"
else
    log_warning "TCP Server health check failed"
    HEALTH_FAILED=true
fi

# Display deployment summary
log_info "Deployment Summary"
echo "===================="
echo "VPS IP: $VPS_IP"
echo "Branch: $BRANCH"
echo "Architecture: $(if [[ "$COMPOSE_FILE" == *"microservices"* ]]; then echo "Microservices"; else echo "Monolithic"; fi)"
echo ""
echo "🌐 Access Points:"
echo "   Web UI:        http://$VPS_IP:3000"
echo "   TCP Server:    $VPS_IP:8080"
echo "   API Server:    http://$VPS_IP:8082"
echo "   API Docs:      http://$VPS_IP:8082/swagger/index.html"

if [[ "$COMPOSE_FILE" == *"microservices"* ]]; then
    echo "   Registry:      http://$VPS_IP:8083"
    echo "   Gateway:       http://$VPS_IP:8084"
    echo "   Monitor:       http://$VPS_IP:8085"
fi

echo ""
echo "📋 Management Commands:"
echo "   View logs:     ssh $VPS_USER@$VPS_IP 'cd $DEPLOY_DIR && docker-compose logs -f'"
echo "   Restart:       ssh $VPS_USER@$VPS_IP 'cd $DEPLOY_DIR && docker-compose restart'"
echo "   Stop:          ssh $VPS_USER@$VPS_IP 'cd $DEPLOY_DIR && docker-compose down'"

if [ "$HEALTH_FAILED" = true ]; then
    log_warning "Some health checks failed. Please check the logs for issues."
    $SSH_CMD "cd $DEPLOY_DIR && docker-compose ps"
    exit 1
else
    log_success "Deployment completed successfully!"
fi