docker build -t log-transporter:latest .
docker run --env "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}" \
       --env "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}" \
       --env "AWS_SESSION_TOKEN=${AWS_SESSION_TOKEN}" \
       -ti --rm log-transporter:latest /bin/bash
