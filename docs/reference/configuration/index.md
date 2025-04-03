# Configuration reference

```yaml
# Config holds the information needed to connect to remote Grafana instances.
# Contexts is a map of context configurations, indexed by name.
contexts: 
  ${string}:
    # Context holds the information required to connect to a remote Grafana instance.
    grafana: 
      # Server is the address of the Grafana server (https://hostname:port/path).
      # Required.
      server: string
      # User to authenticate as with basic authentication.
      # Optional.
      user: string
      # Password to use when using with basic authentication.
      # Optional.
      password: string
      # APIToken is a service account token.
      # See https://grafana.com/docs/grafana/latest/administration/service-accounts/#add-a-token-to-a-service-account-in-grafana
      # Note: if defined, the API Token takes precedence over basic auth credentials.
      # Optional.
      token: string
      # OrgID specifies the organization targeted by this config.
      # Note: required when targeting an on-prem Grafana instance.
      # See StackID for Grafana Cloud instances.
      org-id: int
      # StackID specifies the Grafana Cloud stack targeted by this config.
      # Note: required when targeting a Grafana Cloud instance.
      # See OrgID for on-prem Grafana instances.
      stack-id: int
      # TLS contains TLS-related configuration settings.
      tls: 
        # TLS contains settings to enable transport layer security.
        # InsecureSkipTLSVerify disables the validation of the server's SSL certificate.
        # Enabling this will make your HTTPS connections insecure.
        insecure-skip-verify: bool
        # ServerName is passed to the server for SNI and is used in the client to check server
        # certificates against. If ServerName is empty, the hostname used to contact the
        # server is used.
        server-name: string
        # CertData holds PEM-encoded bytes (typically read from a client certificate file).
        # CertData takes precedence over CertFile
        # Note: this value is base64-encoded in the config file and will be
        # automatically decoded.
        cert-data: 
          - int
          - ...
          
        # KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
        # KeyData takes precedence over KeyFile
        # Note: this value is base64-encoded in the config file and will be
        # automatically decoded.
        key-data: 
          - int
          - ...
          
        # CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
        # CAData takes precedence over CAFile
        # Note: this value is base64-encoded in the config file and will be
        # automatically decoded.
        ca-data: 
          - int
          - ...
          
        # NextProtos is a list of supported application level protocols, in order of preference.
        # Used to populate tls.Config.NextProtos.
        # To indicate to the server http/1.1 is preferred over http/2, set to ["http/1.1", "h2"] (though the server is free to ignore that preference).
        # To use only http/1.1, set to ["http/1.1"].
        next-protos: 
          - string
          - ...
          
# CurrentContext is the name of the context currently in use.
current-context: string
```
