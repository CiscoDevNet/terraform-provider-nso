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
