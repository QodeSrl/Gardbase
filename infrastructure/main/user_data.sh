#!/bin/bash
#
# User data script for EC2 instance with Nitro Enclave
# This script sets up Docker, pulls the enclave image, and starts the enclave
#

set -e
set -o pipefail

# Logging
exec > >(tee /var/log/user_data.log)
exec 2>&1

echo "Starting user data script..."

# Variables from Terraform
REGION="${region}"
PROJECT_NAME="${project_name}"
ENVIRONMENT="${environment}"
ECR_REPOSITORY_URL="${ecr_repository_url}"
S3_BUCKET="${s3_bucket}"
DYNAMO_OBJECTS_TABLE="${dynamo_objects_table}"
DYNAMO_INDEXES_TABLE="${dynamo_indexes_table}"
KMS_KEY_ID="${kms_key_id}"
ENCLAVE_CPUS="${enclave_cpus}"
ENCLAVE_MEMORY_MIB="${enclave_memory_mib}"
ENABLE_DEBUG_MODE="${enable_debug_mode}"
MAX_ATTESTATION_AGE_MINUTES="${max_attestation_age_minutes}"

echo "Configuration:"
echo "  Region: $REGION"
echo "  Environment: $ENVIRONMENT"
echo "  ECR URL: $ECR_REPOSITORY_URL"
echo "  Enclave CPUs: $ENCLAVE_CPUS"
echo "  Enclave Memory: $ENCLAVE_MEMORY_MIB MiB"
echo "  Debug Mode: $ENABLE_DEBUG_MODE"

# Update system
echo "Updating system packages..."
dnf update -y

# Install required packages
echo "Installing required packages..."
dnf install -y \
    docker \
    aws-nitro-enclaves-cli \
    aws-nitro-enclaves-cli-devel \
    jq \
    git

# Start Docker service
echo "Starting Docker service..."
systemctl enable docker
systemctl start docker

# Update enclave allocator configuration
cat > /etc/nitro_enclaves/allocator.yaml <<EOF
---
# Enclave resource allocations
memory_mib: $ENCLAVE_MEMORY_MIB
cpu_count: $ENCLAVE_CPUS
EOF

# Start and enable Nitro Enclaves allocator
echo "Starting Nitro Enclaves allocator..."
systemctl enable nitro-enclaves-allocator.service
systemctl start nitro-enclaves-allocator.service

# Verify allocator status
echo "Nitro Enclaves allocator status:"
systemctl status nitro-enclaves-allocator.service --no-pager

# Add docker user to ne group (Nitro Enclaves group)
usermod -aG ne ec2-user
usermod -aG docker ec2-user

# Login to ECR
echo "Logging into ECR..."
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$ECR_REPOSITORY_URL"

# Pull the latest images
echo "Pulling enclave image from ECR..."
docker pull "$ECR_REPOSITORY_URL:latest-parent" || \
    echo "Error: Could not pull parent image from ECR"

docker pull "$ECR_REPOSITORY_URL:latest-enclave" || \
    echo "Warning: Enclave image not found, will be built on first run"

# Create directories for application
echo "Creating application directories..."
mkdir -p /opt/gardbase/{logs,config,data}
chown -R ec2-user:ec2-user /opt/gardbase

# Create systemd service for the parent application
echo "Creating systemd service for Gardbase parent application..."
cat > /etc/systemd/system/gardbase-parent.service <<EOF
[Unit]
Description=Gardbase Parent Application
After=docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
RestartSec=10
User=root
ExecStartPre=-/usr/bin/docker stop gardbase-parent
ExecStartPre=-/usr/bin/docker rm gardbase-parent
ExecStartPre=/usr/bin/docker pull $ECR_REPOSITORY_URL:latest-parent
ExecStart=/usr/bin/docker run --rm --name gardbase-parent \\
    -p 80:80 \\
    -p 443:443 \\
    -v /opt/gardbase/logs:/app/logs \\
    -e S3_BUCKET="$S3_BUCKET" \\
    -e DYNAMO_OBJECTS_TABLE="$DYNAMO_OBJECTS_TABLE" \\
    -e DYNAMO_INDEXES_TABLE="$DYNAMO_INDEXES_TABLE" \\
    -e AWS_REGION="$REGION" \\
    -e KMS_KEY_ID="$KMS_KEY_ID" \\
    -e AWS_MAX_RETRIES="3" \\
    -e AWS_REQUEST_TIMEOUT="10" \\
    -e USE_LOCALSTACK="false" \\
    -e LOCALSTACK_URL="" \\
    -e ENVIRONMENT="$ENVIRONMENT" \\
    -e PORT="80" \\
    -e ENCLAVE_PORT="5000" \\
    -e ENCLAVE_CID="16" \\
    -e ENABLE_DEBUG_MODE="$ENABLE_DEBUG_MODE" \\
    -e MAX_ATTESTATION_AGE_MINUTES="$MAX_ATTESTATION_AGE_MINUTES" \\
    $ECR_REPOSITORY_URL:latest-parent
ExecStop=/usr/bin/docker stop gardbase-parent

[Install]
WantedBy=multi-user.target
EOF

# Create run enclave script
echo "Creating run enclave script..."
cat > /opt/gardbase/run-enclave.sh <<EOF
#!/bin/bash
set -e

LOG_FILE="/opt/gardbase/enclave-run.log"

CMD="/usr/bin/nitro-cli run-enclave \
  --eif-path /opt/gardbase/enclave.eif \
  --cpu-count $ENCLAVE_CPUS \
  --memory $ENCLAVE_MEMORY_MIB \
  --enclave-cid 16"

if [ "$ENABLE_DEBUG_MODE" = "true" ]; then
  CMD="\$CMD --debug-mode"
fi

echo "Starting enclave..." > "\$LOG_FILE"

# Run the command
eval "\$CMD" >> "\$LOG_FILE" 2>&1
EOF

chmod +x /opt/gardbase/run-enclave.sh

# Create capture console script
echo "Creating capture console script..."
cat > /opt/gardbase/capture-console.sh <<EOF
#!/bin/bash
set -e
LOG_FILE="/opt/gardbase/enclave-console.log"
if [ "$ENABLE_DEBUG_MODE" = "true" ]; then
    echo "Capturing enclave console output..." > "\$LOG_FILE"
    while ! nitro-cli describe-enclaves | jq -e '.[0].State == "RUNNING"' > /dev/null 2>&1; do
        sleep 2
    done
    nitro-cli console --enclave-id \$(nitro-cli describe-enclaves | jq -r '.[0].EnclaveID') >> "\$LOG_FILE" 2>&1 &
fi
EOF

chmod +x /opt/gardbase/capture-console.sh

# Create systemd service for the Nitro Enclave
echo "Creating systemd service for Gardbase Nitro Enclave..."
cat > /etc/systemd/system/gardbase-enclave.service <<EOF
[Unit]
Description=Gardbase Nitro Enclave
After=nitro-enclaves-allocator.service docker.service
Requires=nitro-enclaves-allocator.service docker.service
Before=gardbase-parent.service

[Service]
Type=oneshot
Restart=no
RemainAfterExit=yes

Environment="NITRO_CLI_BLOBS=/usr/share/nitro_enclaves/blobs"
Environment="NITRO_CLI_ARTIFACTS=/opt/nitro_enclaves"
Environment="PATH=/usr/local/bin:/usr/bin:/bin"

# Sleep to allow allocator to be ready
ExecStartPre=/bin/sleep 5

# Build the enclave image file (EIF) from Docker image
ExecStartPre=/bin/bash -c 'if [ ! -f /opt/gardbase/enclave.eif ]; then \\
    echo "Building enclave image..."; \\
    nitro-cli build-enclave \\
        --docker-uri $ECR_REPOSITORY_URL:latest-enclave \\
        --output-file /opt/gardbase/enclave.eif > /opt/gardbase/enclave-build.log 2>&1; \\
    echo "Enclave build complete. PCR values:"; \\
    cat /opt/gardbase/enclave-build.log | grep -A 10 "Measurements"; \\
fi'

# Stop any existing enclave
ExecStartPre=-/usr/bin/nitro-cli terminate-enclave --all

# Start the enclave
ExecStart=/opt/gardbase/run-enclave.sh

# Stop enclave on service stop
ExecStop=/usr/bin/nitro-cli terminate-enclave --all

# Capture enclave console output
ExecStartPost=/opt/gardbase/capture-console.sh

[Install]
WantedBy=multi-user.target
EOF

# Create script to extract and save PCR values
echo "Creating PCR extraction script..."
cat > /opt/gardbase/extract-pcrs.sh <<'EOF'
#!/bin/bash
# Extract PCR values from the enclave build log
BUILD_LOG="/opt/gardbase/enclave-build.log"
OUTPUT_FILE="/opt/gardbase/pcr-values.json"

if [ ! -f "$BUILD_LOG" ]; then
    echo "Build log not found: $BUILD_LOG"
    exit 1
fi

# Extract PCR values
PCR0=$(grep "PCR0:" "$BUILD_LOG" | awk '{print $2}')
PCR1=$(grep "PCR1:" "$BUILD_LOG" | awk '{print $2}')
PCR2=$(grep "PCR2:" "$BUILD_LOG" | awk '{print $2}')

# Create JSON output
cat > "$OUTPUT_FILE" <<EOFPCR
{
  "pcr_values": {
    "PCR0": "$PCR0",
    "PCR1": "$PCR1",
    "PCR2": "$PCR2"
  },
  "extracted_at": "$(date -Iseconds)",
  "environment": "$ENVIRONMENT",
  "debug_mode": $ENABLE_DEBUG_MODE
}
EOFPCR

echo "PCR values saved to $OUTPUT_FILE"
cat "$OUTPUT_FILE"

# Also save to Parameter Store for easy client access
aws ssm put-parameter \
    --region "$REGION" \
    --name "/$PROJECT_NAME/$ENVIRONMENT/enclave/pcr-values" \
    --type "String" \
    --value "$(cat $OUTPUT_FILE)" \
    --overwrite || echo "Warning: Could not save to Parameter Store"
EOF

chmod +x /opt/gardbase/extract-pcrs.sh

# Create health check script
echo "Creating health check script..."
cat > /opt/gardbase/health-check.sh <<'EOF'
#!/bin/bash
# Check if parent application is responding
if ! curl -f http://localhost/health > /dev/null 2>&1; then
    echo "Parent application health check failed"
    exit 1
fi

# Check if enclave is running
if ! nitro-cli describe-enclaves | jq -e '.[0].State == "RUNNING"' > /dev/null 2>&1; then
    echo "Enclave is not running"
    exit 1
fi

echo "Health check passed"
exit 0
EOF

chmod +x /opt/gardbase/health-check.sh

# Install CloudWatch agent
echo "Installing CloudWatch agent..."
curl -fsSL https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm -o /tmp/amazon-cloudwatch-agent.rpm
rpm -U /tmp/amazon-cloudwatch-agent.rpm

# Ensure CloudWatch agent directories exist
mkdir -p /opt/aws/amazon-cloudwatch-agent/etc

# Create CloudWatch Logs configuration
echo "Configuring CloudWatch Logs..."
cat > /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json <<EOF
{
  "logs": {
    "logs_collected": {
      "files": {
        "collect_list": [
          {
            "file_path": "/var/log/user-data.log",
            "log_group_name": "/aws/ec2/$PROJECT_NAME-api-$ENVIRONMENT",
            "log_stream_name": "{instance_id}/user-data"
          },
          {
            "file_path": "/opt/gardbase/logs/enclave-console.log",
            "log_group_name": "/aws/ec2/$PROJECT_NAME-enclave-$ENVIRONMENT",
            "log_stream_name": "{instance_id}/console"
          }
        ]
      }
    }
  }
}
EOF

# Install and start CloudWatch agent (if available)
if command -v amazon-cloudwatch-agent-ctl &> /dev/null; then
    echo "Starting CloudWatch agent..."
    /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl \
        -a fetch-config \
        -m ec2 \
        -s \
        -c file:/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json
fi

# Reload systemd
echo "Reloading systemd..."
systemctl daemon-reload

# Enable enclave service first (so EIF is built before parent starts)
echo "Enabling Gardbase enclave service..."
systemctl enable gardbase-enclave.service
systemctl start gardbase-enclave.service

# Wait for enclave to start
echo "Waiting for enclave to start..."
sleep 10

echo "Enclave status:"
nitro-cli describe-enclaves | jq '.'

# Extract PCR values
echo "Extracting PCR values..."
/opt/gardbase/extract-pcrs.sh

# Start parent application service
echo "Enabling Gardbase parent application service..."
systemctl enable gardbase-parent.service
systemctl start gardbase-parent.service

# Wait for parent to start
sleep 5

# Check service status
echo "=========================================="
echo "Service Status:"
echo "=========================================="
systemctl status gardbase-enclave.service --no-pager
systemctl status gardbase-parent.service --no-pager

# Run health check
echo "=========================================="
echo "Running health check..."
echo "=========================================="
/opt/gardbase/health-check.sh

echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo "API Endpoint: http://\$(ec2-metadata --public-ipv4 | cut -d' ' -f2)"
echo "Logs:"
echo "  - User data: /var/log/user-data.log"
echo "  - Enclave build: /opt/gardbase/enclave-build.log"
echo "  - Enclave console: /opt/gardbase/logs/enclave-console.log"
echo "  - Parent service: journalctl -u gardbase-parent.service -f"
echo "  - Enclave service: journalctl -u gardbase-enclave.service -f"
echo ""
echo "PCR Values: /opt/gardbase/pcr-values.json"
echo "SSH: ssh ec2-user@\$(ec2-metadata --public-ipv4 | cut -d' ' -f2)"
echo "=========================================="

# Create MOTD with instructions
cat > /etc/motd <<'EOFMOTD'
╔═══════════════════════════════════════════════════════════════╗
║            Welcome to Gardbase Nitro Enclave Server          ║
╚═══════════════════════════════════════════════════════════════╝

Useful Commands:
  • View enclave status:     nitro-cli describe-enclaves
  • View enclave console:    nitro-cli console --enclave-id $(nitro-cli describe-enclaves | jq -r '.[0].EnclaveID')
  • View PCR values:         cat /opt/gardbase/pcr-values.json
  • Parent service logs:     journalctl -u gardbase-parent.service -f
  • Enclave service logs:    journalctl -u gardbase-enclave.service -f
  • Health check:            /opt/gardbase/health-check.sh
  • Restart enclave:         sudo systemctl restart gardbase-enclave.service
  • Restart parent:          sudo systemctl restart gardbase-parent.service

Directories:
  • Application:             /opt/gardbase/
  • Logs:                    /opt/gardbase/logs/
  • Enclave EIF:             /opt/gardbase/enclave.eif

Security Note:
  This instance is running a Nitro Enclave for secure key management.
  PCR values must be verified by clients for trusted execution.

EOFMOTD

echo "User data script completed successfully!"
