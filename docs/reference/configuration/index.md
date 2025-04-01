# Configuration reference

```yaml
# Config holds the information needed to connect to remote Grafana instances.
# Contexts is a map of context configurations, indexed by name.
contexts: 
  ${string}:
    # Context holds the information required to connect to a remote Grafana instance.
    grafana: 
      # Server is the address of the Grafana server (https://hostname:port/path).
      server: string
      user: string
      token: string
      # InsecureSkipTLSVerify disables the validation of the server's SSL certificate.
      # Enabling this will make your HTTPS connections insecure.
      insecure-skip-tls-verify: bool
# CurrentContext is the name of the context currently in use.
current-context: string
```
