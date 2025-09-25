# Container Management User Guide

## Overview

The Docker Container Lifecycle Management System provides a comprehensive web interface and API for managing Docker containers with advanced features including real-time monitoring, automated updates, and web-based terminal access.

## Getting Started

### Accessing the System

1. **Web Interface**: Navigate to `http://localhost` in your web browser
2. **Default Login**:
   - Email: `admin@example.com`
   - Password: `admin123`
   - **⚠️ Important**: Change the default password immediately after first login

### Dashboard Overview

The main dashboard provides:
- **Container Status Summary**: Quick overview of running, stopped, and total containers
- **System Resource Usage**: CPU, memory, and disk usage metrics
- **Recent Activity**: Latest container operations and alerts
- **Quick Actions**: Start, stop, restart containers directly from the dashboard

## Container Management

### Adding a New Container

1. **Navigate to Containers**: Click "Containers" in the main navigation
2. **Add Container**: Click the "Add Container" button
3. **Configure Container**:
   - **Name**: Unique container name
   - **Image**: Docker image (e.g., `nginx:latest`, `postgres:13`)
   - **Ports**: Map container ports to host ports
   - **Environment Variables**: Set environment variables
   - **Volumes**: Mount host directories or named volumes
   - **Resource Limits**: Set CPU and memory limits

4. **Advanced Configuration**:
   - **Health Check**: Configure container health monitoring
   - **Restart Policy**: Set automatic restart behavior
   - **Network Settings**: Configure network mode and aliases
   - **Registry Settings**: Configure private registry authentication

### Container Operations

#### Basic Operations

**Start Container**:
- Click the "Start" button on the container row
- Or navigate to container details and click "Start"

**Stop Container**:
- Click the "Stop" button with optional timeout
- Graceful shutdown with SIGTERM, then SIGKILL after timeout

**Restart Container**:
- Combines stop and start operations
- Preserves container configuration

**Remove Container**:
- Click "Remove" (only available when stopped)
- Option to remove associated volumes

#### Bulk Operations

Select multiple containers using checkboxes to perform bulk operations:
- Start multiple containers
- Stop multiple containers
- Remove multiple containers
- Update multiple containers

### Container Configuration

#### Environment Variables

Add environment variables for your container:
```
NODE_ENV=production
DATABASE_URL=postgresql://user:pass@localhost/db
API_KEY=your-secret-key
```

#### Port Mapping

Map container ports to host ports:
- **Container Port**: Port inside the container
- **Host Port**: Port on the host machine
- **Protocol**: TCP or UDP

Example: Map container port 80 to host port 8080 for web access.

#### Volume Mounts

**Bind Mounts**: Mount host directories into containers
- Host Path: `/data/nginx`
- Container Path: `/usr/share/nginx/html`
- Mode: `ro` (read-only) or `rw` (read-write)

**Named Volumes**: Use Docker volumes for persistent data
- Volume Name: `postgres-data`
- Container Path: `/var/lib/postgresql/data`

#### Health Checks

Configure health checks to monitor container health:
- **Command**: Health check command (e.g., `curl -f http://localhost/health`)
- **Interval**: How often to run the check (default: 30s)
- **Timeout**: Maximum time to wait for response (default: 30s)
- **Retries**: Number of consecutive failures before marking unhealthy (default: 3)

## Real-time Monitoring

### Container Metrics

Access real-time container performance data:

1. **Navigate to Container Details**: Click on a container name
2. **Monitoring Tab**: View real-time metrics including:
   - **CPU Usage**: Current CPU utilization percentage
   - **Memory Usage**: RAM usage and limits
   - **Network I/O**: Network traffic in/out
   - **Disk I/O**: Disk read/write operations

### Performance Charts

The monitoring interface provides interactive charts:
- **Time-series Data**: Historical performance over time
- **Zoom Controls**: Focus on specific time periods
- **Export Options**: Download metrics data as CSV

### Alerts and Notifications

Set up alerts for:
- High CPU usage (>80%)
- High memory usage (>90%)
- Container health check failures
- Container stops/crashes

Configure notification methods:
- **Email**: Receive alerts via email
- **Slack**: Integration with Slack channels
- **Webhooks**: Custom webhook endpoints

## Web Terminal Access

### Accessing Container Shell

1. **Navigate to Container Details**
2. **Terminal Tab**: Click to open web terminal
3. **Shell Selection**: Choose shell (`/bin/bash`, `/bin/sh`, `/bin/zsh`)
4. **Interactive Session**: Full terminal with command history

### Terminal Features

- **Copy/Paste**: Standard Ctrl+C/Ctrl+V support
- **Resize**: Terminal automatically resizes with window
- **Multiple Sessions**: Open multiple terminal sessions per container
- **Command History**: Navigate command history with arrow keys
- **File Editing**: Use nano, vim, or other editors

### Security Considerations

- Terminal access respects container user permissions
- Sessions automatically timeout after inactivity
- All terminal sessions are logged for audit purposes
- Access controlled by user permissions

## Automated Updates

### Update Policies

Configure automatic update behavior:

**Auto Update**: Automatically update containers when new versions are available
- **Rolling Updates**: Update containers one at a time with zero downtime
- **Blue-Green Deployment**: Switch between two identical environments
- **Canary Deployment**: Gradually roll out updates to subset of instances

**Manual Update**: Require manual approval for all updates
- Receive notifications when updates are available
- Review changes before applying
- Schedule updates during maintenance windows

**Scheduled Updates**: Update containers on a schedule
- Define maintenance windows
- Automatically apply updates during specified times
- Skip updates if health checks fail

### Update Process

1. **Check for Updates**: System automatically checks for new image versions
2. **Notification**: Receive notification when updates are available
3. **Review Changes**: View changelog and version differences
4. **Apply Update**: Choose update strategy and apply
5. **Health Check**: Automatic health verification after update
6. **Rollback**: Automatic rollback if health checks fail

### Rollback Capabilities

If an update fails:
- **Automatic Rollback**: Revert to previous version if health checks fail
- **Manual Rollback**: Manually revert to any previous version
- **Rollback History**: View and restore from update history

## Registry Management

### Adding Container Registries

Support for multiple registry types:

**Docker Hub**:
- Public images: No authentication required
- Private images: Username and password or access token

**Private Registries**:
- Harbor, GitLab Container Registry, AWS ECR, Azure ACR, Google GCR
- Username/password authentication
- Token-based authentication
- Certificate-based authentication

**Configuration Steps**:
1. Navigate to **Settings > Registries**
2. Click **Add Registry**
3. Enter registry details:
   - **Name**: Human-readable name
   - **URL**: Registry URL
   - **Type**: Registry type (Docker Hub, Harbor, etc.)
   - **Credentials**: Username, password, or token
4. **Test Connection**: Verify registry access
5. **Save**: Registry is now available for container deployments

### Registry Features

- **Connection Testing**: Verify registry connectivity
- **Image Search**: Search available images in registries
- **Authentication Management**: Secure credential storage
- **Usage Statistics**: Monitor registry usage and performance

## User Management and Permissions

### User Roles

**Administrator**:
- Full system access
- User management
- System configuration
- All container operations

**Operator**:
- Container management
- Monitoring access
- Update operations
- Limited system settings

**Viewer**:
- Read-only access
- View containers and metrics
- No modification permissions

### Permission Management

Configure user permissions:
1. Navigate to **Settings > Users**
2. Select user or create new user
3. Assign role and specific permissions
4. Set container access restrictions

## Best Practices

### Container Naming

- Use descriptive names: `nginx-web-server`, `postgres-database`
- Include environment: `api-production`, `frontend-staging`
- Avoid special characters except hyphens and underscores

### Resource Management

- **Set Resource Limits**: Always set CPU and memory limits
- **Monitor Usage**: Regularly review resource utilization
- **Health Checks**: Configure appropriate health checks
- **Log Rotation**: Ensure logs don't consume excessive disk space

### Security

- **Regular Updates**: Keep container images updated
- **Least Privilege**: Run containers with minimal required permissions
- **Network Security**: Use appropriate network configurations
- **Secrets Management**: Use environment variables for sensitive data

### Backup and Recovery

- **Volume Backups**: Regularly backup persistent data volumes
- **Configuration Backup**: Export container configurations
- **Update History**: Maintain update history for rollback capability
- **Disaster Recovery**: Document recovery procedures

## Troubleshooting

### Common Issues

**Container Won't Start**:
- Check container logs in the Logs tab
- Verify image exists and is accessible
- Check port conflicts
- Ensure sufficient system resources

**High Resource Usage**:
- Review container metrics
- Check for memory leaks in applications
- Optimize container resource limits
- Consider horizontal scaling

**Update Failures**:
- Check container health after update
- Review update logs
- Verify image compatibility
- Use rollback if necessary

**Network Issues**:
- Verify port mappings
- Check firewall settings
- Ensure network connectivity
- Review DNS configuration

### Getting Help

1. **Documentation**: Check this user guide and API documentation
2. **System Logs**: Review application logs in the web interface
3. **Health Status**: Check system health on the dashboard
4. **Support**: Contact system administrator or support team

### Log Analysis

Access container logs through the web interface:
- **Real-time Logs**: Stream container logs in real-time
- **Historical Logs**: Search and filter log history
- **Log Export**: Download logs for external analysis
- **Log Rotation**: Configure log retention policies

## Advanced Features

### Custom Health Checks

Create sophisticated health checks:
```bash
# HTTP health check
curl -f http://localhost:8080/health || exit 1

# Database connection check
pg_isready -h localhost -p 5432 || exit 1

# Application-specific check
/app/scripts/health-check.sh
```

### Environment Templates

Create reusable configuration templates:
- **Development Template**: Standard development container setup
- **Production Template**: Production-ready configuration with monitoring
- **Database Template**: Common database container configurations

### Integration Hooks

Configure webhooks for external integrations:
- **CI/CD Integration**: Trigger deployments on container updates
- **Monitoring Integration**: Send metrics to external monitoring systems
- **Alerting Integration**: Forward alerts to external alerting systems

## FAQ

**Q: Can I manage containers across multiple Docker hosts?**
A: Yes, configure multiple Docker endpoints in system settings.

**Q: How do I backup container data?**
A: Use volume mounts and backup the host directories, or create volume snapshots.

**Q: Can I schedule container operations?**
A: Yes, use the scheduling feature to start/stop containers at specific times.

**Q: How do I migrate containers between hosts?**
A: Export container configuration and import on the target host.

**Q: What happens if the management system goes down?**
A: Containers continue running normally; restart the management system to regain control.

**Q: How do I monitor system performance?**
A: Use the system monitoring dashboard and configure alerts for key metrics.