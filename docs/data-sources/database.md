---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "snowflake_database Data Source - terraform-provider-snowflake"
subcategory: ""
description: |-
  
---

# snowflake_database (Data Source)



## Example Usage

```terraform
data "snowflake_database" "this" {
  name = "DEMO_DB"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The database from which to return its metadata.

### Read-Only

- `comment` (String)
- `created_on` (String)
- `id` (String) The ID of this resource.
- `is_current` (Boolean)
- `is_default` (Boolean)
- `options` (String)
- `origin` (String)
- `owner` (String)
- `retention_time` (Number)


