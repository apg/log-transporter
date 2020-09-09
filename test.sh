#/bin/bash

cat > /etc/aws.credentials <<EOF
[default]
aws_access_key_id=${AWS_ACCESS_KEY_ID}
aws_secret_access_key=${AWS_SECRET_ACCESS_KEY}
aws_session_token=${AWS_SESSION_TOKEN}
EOF

echo "Starting syslogd"
/usr/sbin/rsyslogd

for n in `seq 1 100`; do
    for m in `seq 1 100`; do
        echo "Audit Message. This is tick $n:$m" >> /var/log/test_audit.log
    done
    sleep 1
done
