FROM envoyproxy/envoy:v1.31-latest

# Note: envoy.yaml is mounted via docker-compose volume
# This allows for configuration changes without rebuilding the image
RUN chmod go+r /etc/envoy/envoy.yaml || true
