## Monitor your Docker instance using Prometheus

Your Docker instance metrics are queried using PromQL, and the results are stored in a SQLite database. This database can later be used for further analysis.

### Prerequisites

1) Configure the Docker daemon as a Prometheus target. For this, you need to specify the `metrics-address` in the `daemon.json` configuration file. On Linux, add the following configuration to `/etc/docker/daemon.json` and **restart Docker**. 

    ```json
    {
    "metrics-addr": "127.0.0.1:9323"
    }
    ```

2) Copy the following stock Prometheus configuration file to a location of your choice (`/tmp/prometheus.yml`, for example). 

    ```yaml
    # my global config
    global:
    scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
    evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
    # scrape_timeout is set to the global default (10s).

    # Attach these labels to any time series or alerts when communicating with
    # external systems (federation, remote storage, Alertmanager).
    external_labels:
        monitor: "codelab-monitor"

    # Load rules once and periodically evaluate them according to the global 'evaluation_interval'.
    rule_files:
    # - "first.rules"
    # - "second.rules"

    # A scrape configuration containing exactly one endpoint to scrape:
    # Here it's Prometheus itself.
    scrape_configs:
    # The job name is added as a label `job=<job_name>` to any timeseries scraped from this config.
    - job_name: prometheus

        # metrics_path defaults to '/metrics'
        # scheme defaults to 'http'.

        static_configs:
        - targets: ["localhost:9090"]

    - job_name: docker
        # metrics_path defaults to '/metrics'
        # scheme defaults to 'http'.

        static_configs:
        - targets: ["host.docker.internal:9323"]
    ```

3) Run Prometheus in a container.

    ```bash
    docker run --name my-prometheus \  
        --mount type=bind,source=/tmp/prometheus.yml,destination=/etc/prometheus/prometheus.yml \
        -p 9090:9090 \
        --add-host host.docker.internal=host-gateway \
        prom/prometheus
    ```

4) Open the Prometheus Dashboard

    Verify that the Docker target is listed at http://localhost:9090/targets/.

### Usage 

```bash
go run .
```