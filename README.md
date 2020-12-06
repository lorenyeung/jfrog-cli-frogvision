# frogvision

## About this plugin
This plugin uses the open metrics API of Artifactory to visually display information in graphical format
This README shows the expected structure of your plugin's README.

## Installation with JFrog CLI
Installing the latest version:

`$ jfrog plugin install frogvision`

Installing a specific version:

`$ jfrog plugin install frogvision@version`

Uninstalling a plugin

`$ jfrog plugin uninstall frogvision`

## Usage
### Commands
* graph
    - Arguments:
        - none
    - Flags:
        - none
    - Example:
    ```
   $ jfrog frogvision graph
    
    [display gif here]
    ```
* metrics
    - Arguments:
        - list: list metrics
    - Flags:
        - raw: Output straight from Artifactory **[Default: false]**
        - min: Get minimum JSON from Artifactory (no whitespace) **[Default: false]**
    - Example:
    ```
  $ jfrog frogvision metrics --raw

  # HELP jfrt_artifacts_gc_current_size_bytes Total space occupied by binaries after Garbage Collection
  # UPDATED jfrt_artifacts_gc_current_size_bytes 1607284811440
  # TYPE jfrt_artifacts_gc_current_size_bytes gauge
  jfrt_artifacts_gc_current_size_bytes{end="1607284801199",start="1607284800142",status="COMPLETED",type="FULL"} 3.823509e+10 1607287853275  
  ```

### Environment variables
* HELLO_FROG_GREET_PREFIX - Adds a prefix to every greet **[Default: New greeting: ]**

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).
