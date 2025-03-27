## grafanactl config

View or manipulate configuration settings

### Synopsis

View or manipulate configuration settings.

The configuration file to load is chosen as follows:

1. If the --config flag is set, then that file will be loaded. No other location will be considered.
2. If the $XDG_CONFIG_HOME environment variable is set, then it will be used: $XDG_CONFIG_HOME/grafanactl/config.yaml
   Example: /home/user/.config/grafanactl/config.yaml
3. If the $HOME environment variable is set, then it will be used: $HOME/.config/grafanactl/config.yaml
   Example: /home/user/.config/grafanactl/config.yaml
4. If the $XDG_CONFIG_DIRS environment variable is set, then it will be used: $XDG_CONFIG_DIRS/grafanactl/config.yaml
   Example: /etc/xdg/grafanactl/config.yaml


### Options

```
      --config string    Path to the configuration file to use
      --context string   Name of the context to use
  -h, --help             help for config
```

### Options inherited from parent commands

```
      --no-color   Disable color output
```

### SEE ALSO

* [grafanactl](grafanactl.md)	 - 
* [grafanactl config current-context](grafanactl_config_current-context.md)	 - Display the current context name
* [grafanactl config set](grafanactl_config_set.md)	 - Set an single value in a configuration file
* [grafanactl config unset](grafanactl_config_unset.md)	 - Unset an single value in a configuration file
* [grafanactl config use-context](grafanactl_config_use-context.md)	 - Set the current context
* [grafanactl config view](grafanactl_config_view.md)	 - Display the current configuration

