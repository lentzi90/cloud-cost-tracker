# Start
To start the database and grafana run the command `sudo docker-compose up`

# Setup prometheus
add `http://<address to influx host>:8086/api/v1/prom/write?u=prom&p=prom&db=prometheus` to `remoteWriteURL` in `/agents/values.yaml`
