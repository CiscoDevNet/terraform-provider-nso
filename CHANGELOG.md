## 0.3.0 (unreleased)

- BREAKING CHANGE: Communicate with NSO via NETCONF instead of RESTCONF; RESTCONF is no longer supported
- BREAKING CHANGE: Replace provider attribute `url` (`NSO_URL`) with `host` (`NSO_HOST`), which defaults to port `2022`; the same applies to `instances[].url`
- BREAKING CHANGE: Rename `nso_restconf` resource and data source to `nso_netconf`; `path` is now an absolute XPath, e.g. `/tailf-ncs:devices/device-group[name='GROUP1']`
- BREAKING CHANGE: `nso_device_config` `path` is now an XPath relative to the device config, e.g. `tailf-ned-cisco-ios:interface/Loopback[name='0']`
- BREAKING CHANGE: Resource IDs are now XPaths and import identifiers changed; existing resources must be removed from state and re-imported
- Add provider attributes `attempt_timeout`, `total_timeout`, `lock_release_timeout`, `reuse_connection` and `auto_commit`
- Discover NED module XML namespaces from NETCONF capabilities

## 0.2.1

- Fix issue where nested list paths were not translated correctly in the `nso_restconf` resource
- Add `generic_ned_id` attribute to `nso_device` resource and data source

## 0.2.0

- Migrate to `CiscoDevNet` registry namespace
- BREAKING CHANGE: Remove `attributes` map of list items in `nso_restconf` resource
- BREAKING CHANGE: Remove `attributes` map of list items in `nso_device_config` resource

## 0.1.1

- Add option to specify nested attributes (YANG containers) to `nso_restconf` and `nso_device_config` resources

## 0.1.0

- Initial release
