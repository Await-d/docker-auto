-- Docker Auto Update System Database Initialization
-- PostgreSQL Database Schema and Initial Data

-- Set timezone
SET timezone = 'Asia/Shanghai';

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar TEXT,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('admin', 'user', 'viewer')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    last_login TIMESTAMP WITH TIME ZONE,
    login_count INTEGER DEFAULT 0,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    must_change_password BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create containers table
CREATE TABLE IF NOT EXISTS containers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    container_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255) NOT NULL,
    tag VARCHAR(100) DEFAULT 'latest',
    status VARCHAR(50) NOT NULL,
    state VARCHAR(50) NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    ports JSONB,
    environment JSONB,
    volumes JSONB,
    networks JSONB,
    labels JSONB,
    mounts JSONB,
    restart_policy JSONB,
    health_check JSONB,
    update_available BOOLEAN DEFAULT FALSE,
    auto_update_enabled BOOLEAN DEFAULT FALSE,
    update_policy VARCHAR(50) DEFAULT 'manual' CHECK (update_policy IN ('manual', 'auto', 'scheduled')),
    check_interval INTEGER DEFAULT 3600,
    last_checked TIMESTAMP WITH TIME ZONE,
    last_updated TIMESTAMP WITH TIME ZONE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create images table
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id VARCHAR(255) UNIQUE NOT NULL,
    repository VARCHAR(255) NOT NULL,
    tag VARCHAR(100) NOT NULL,
    digest VARCHAR(255),
    size BIGINT,
    architecture VARCHAR(50),
    os VARCHAR(50),
    created TIMESTAMP WITH TIME ZONE,
    labels JSONB,
    last_scan TIMESTAMP WITH TIME ZONE,
    vulnerabilities JSONB,
    security_score INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository, tag)
);

-- Create update_policies table
CREATE TABLE IF NOT EXISTS update_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    update_strategy VARCHAR(50) DEFAULT 'recreate' CHECK (update_strategy IN ('recreate', 'rolling', 'blue_green')),
    schedule_type VARCHAR(20) DEFAULT 'manual' CHECK (schedule_type IN ('manual', 'immediate', 'scheduled', 'cron')),
    schedule_value VARCHAR(100),
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    auto_rollback BOOLEAN DEFAULT TRUE,
    rollback_timeout INTEGER DEFAULT 300,
    health_check_enabled BOOLEAN DEFAULT TRUE,
    health_check_timeout INTEGER DEFAULT 60,
    notification_enabled BOOLEAN DEFAULT TRUE,
    notification_channels JSONB DEFAULT '["email"]'::jsonb,
    backup_enabled BOOLEAN DEFAULT FALSE,
    maintenance_window_enabled BOOLEAN DEFAULT FALSE,
    maintenance_window_start TIME,
    maintenance_window_end TIME,
    maintenance_window_days INTEGER[] DEFAULT ARRAY[1,2,3,4,5],
    max_parallel_updates INTEGER DEFAULT 1,
    update_delay_seconds INTEGER DEFAULT 0,
    requires_approval BOOLEAN DEFAULT FALSE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create update_history table
CREATE TABLE IF NOT EXISTS update_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    container_id UUID REFERENCES containers(id) ON DELETE CASCADE,
    update_policy_id UUID REFERENCES update_policies(id) ON DELETE SET NULL,
    from_image VARCHAR(255) NOT NULL,
    to_image VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled', 'rolled_back')),
    update_type VARCHAR(50) DEFAULT 'manual' CHECK (update_type IN ('manual', 'auto', 'scheduled')),
    strategy VARCHAR(50) DEFAULT 'recreate',
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    error_message TEXT,
    logs TEXT,
    rollback_available BOOLEAN DEFAULT TRUE,
    rollback_reason TEXT,
    triggered_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('info', 'warning', 'error', 'success')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    category VARCHAR(50) DEFAULT 'general' CHECK (category IN ('general', 'update', 'security', 'system')),
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'critical')),
    read BOOLEAN DEFAULT FALSE,
    action_url TEXT,
    action_label VARCHAR(100),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create update_schedules table
CREATE TABLE IF NOT EXISTS update_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    container_id UUID REFERENCES containers(id) ON DELETE CASCADE,
    policy_id UUID REFERENCES update_policies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    enabled BOOLEAN DEFAULT TRUE,
    next_run TIMESTAMP WITH TIME ZONE,
    last_run TIMESTAMP WITH TIME ZONE,
    last_run_status VARCHAR(50),
    run_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    max_failures INTEGER DEFAULT 3,
    retry_interval INTEGER DEFAULT 300,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create system_settings table
CREATE TABLE IF NOT EXISTS system_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(255) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    description TEXT,
    category VARCHAR(100) DEFAULT 'general',
    data_type VARCHAR(50) DEFAULT 'string' CHECK (data_type IN ('string', 'number', 'boolean', 'object', 'array')),
    validation_rule TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    user_configurable BOOLEAN DEFAULT TRUE,
    requires_restart BOOLEAN DEFAULT FALSE,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(20) DEFAULT 'success' CHECK (status IN ('success', 'failure', 'error')),
    error_message TEXT,
    session_id VARCHAR(255),
    request_id VARCHAR(255),
    execution_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

CREATE INDEX IF NOT EXISTS idx_containers_container_id ON containers(container_id);
CREATE INDEX IF NOT EXISTS idx_containers_name ON containers(name);
CREATE INDEX IF NOT EXISTS idx_containers_image ON containers(image);
CREATE INDEX IF NOT EXISTS idx_containers_status ON containers(status);
CREATE INDEX IF NOT EXISTS idx_containers_state ON containers(state);
CREATE INDEX IF NOT EXISTS idx_containers_update_available ON containers(update_available);
CREATE INDEX IF NOT EXISTS idx_containers_auto_update_enabled ON containers(auto_update_enabled);
CREATE INDEX IF NOT EXISTS idx_containers_user_id ON containers(user_id);
CREATE INDEX IF NOT EXISTS idx_containers_created_at ON containers(created_at);

CREATE INDEX IF NOT EXISTS idx_images_image_id ON images(image_id);
CREATE INDEX IF NOT EXISTS idx_images_repository ON images(repository);
CREATE INDEX IF NOT EXISTS idx_images_repository_tag ON images(repository, tag);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at);

CREATE INDEX IF NOT EXISTS idx_update_history_container_id ON update_history(container_id);
CREATE INDEX IF NOT EXISTS idx_update_history_status ON update_history(status);
CREATE INDEX IF NOT EXISTS idx_update_history_start_time ON update_history(start_time);
CREATE INDEX IF NOT EXISTS idx_update_history_triggered_by ON update_history(triggered_by);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
CREATE INDEX IF NOT EXISTS idx_notifications_category ON notifications(category);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);

CREATE INDEX IF NOT EXISTS idx_update_schedules_container_id ON update_schedules(container_id);
CREATE INDEX IF NOT EXISTS idx_update_schedules_enabled ON update_schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_update_schedules_next_run ON update_schedules(next_run);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to tables with updated_at column
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_containers_updated_at BEFORE UPDATE ON containers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_images_updated_at BEFORE UPDATE ON images FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_update_policies_updated_at BEFORE UPDATE ON update_policies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_update_history_updated_at BEFORE UPDATE ON update_history FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON notifications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_update_schedules_updated_at BEFORE UPDATE ON update_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_system_settings_updated_at BEFORE UPDATE ON system_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default admin user (password: admin123)
INSERT INTO users (username, email, password_hash, first_name, last_name, role, status)
VALUES (
    'admin',
    'admin@docker-auto.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LiKLYMSA.JzIauKjb', -- bcrypt hash of 'admin123'
    'System',
    'Administrator',
    'admin',
    'active'
) ON CONFLICT (username) DO NOTHING;

-- Insert default update policy
INSERT INTO update_policies (name, description, update_strategy, schedule_type, auto_rollback, health_check_enabled)
VALUES (
    'Default Policy',
    'Default update policy for new containers',
    'recreate',
    'manual',
    TRUE,
    TRUE
) ON CONFLICT (name) DO NOTHING;

-- Insert default system settings
INSERT INTO system_settings (key, value, description, category, data_type) VALUES
('app.name', '"Docker Auto Update System"', 'Application name', 'general', 'string'),
('app.version', '"1.0.0"', 'Application version', 'general', 'string'),
('app.timezone', '"Asia/Shanghai"', 'Default application timezone', 'general', 'string'),
('docker.default_check_interval', '3600', 'Default container check interval in seconds', 'docker', 'number'),
('docker.max_concurrent_updates', '5', 'Maximum concurrent container updates', 'docker', 'number'),
('notification.email_enabled', 'false', 'Enable email notifications', 'notification', 'boolean'),
('notification.webhook_enabled', 'false', 'Enable webhook notifications', 'notification', 'boolean'),
('security.session_timeout', '86400', 'Session timeout in seconds', 'security', 'number'),
('security.max_login_attempts', '5', 'Maximum failed login attempts before lockout', 'security', 'number'),
('security.lockout_duration', '1800', 'Account lockout duration in seconds', 'security', 'number'),
('monitoring.prometheus_enabled', 'true', 'Enable Prometheus metrics', 'monitoring', 'boolean'),
('monitoring.health_check_interval', '30', 'Health check interval in seconds', 'monitoring', 'number')
ON CONFLICT (key) DO NOTHING;

-- Create views for commonly used queries
CREATE OR REPLACE VIEW container_summary AS
SELECT
    c.id,
    c.container_id,
    c.name,
    c.image,
    c.tag,
    c.status,
    c.state,
    c.update_available,
    c.auto_update_enabled,
    c.last_checked,
    c.last_updated,
    u.username as owner_username,
    (SELECT COUNT(*) FROM update_history uh WHERE uh.container_id = c.id) as update_count,
    (SELECT MAX(uh.start_time) FROM update_history uh WHERE uh.container_id = c.id) as last_update_time
FROM containers c
LEFT JOIN users u ON c.user_id = u.id;

CREATE OR REPLACE VIEW recent_updates AS
SELECT
    uh.id,
    uh.container_id,
    c.name as container_name,
    uh.from_image,
    uh.to_image,
    uh.status,
    uh.update_type,
    uh.strategy,
    uh.start_time,
    uh.end_time,
    uh.duration_seconds,
    u.username as triggered_by_username
FROM update_history uh
LEFT JOIN containers c ON uh.container_id = c.id
LEFT JOIN users u ON uh.triggered_by = u.id
ORDER BY uh.start_time DESC;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE dockerauto TO postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Log initialization completion
DO $$
BEGIN
    RAISE NOTICE 'Docker Auto Update System database initialization completed successfully at %', CURRENT_TIMESTAMP;
END $$;