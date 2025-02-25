package resources

import (
	"errors"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/snowflake"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var validTaskPrivileges = NewPrivilegeSet(
	privilegeMonitor,
	privilegeOperate,
	privilegeOwnership,
)

var taskGrantSchema = map[string]*schema.Schema{
	"database_name": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the database containing the current or future tasks on which to grant privileges.",
		ForceNew:    true,
	},
	"enable_multiple_grants": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When this is set to true, multiple grants of the same type can be created. This will cause Terraform to not revoke grants applied to roles and objects outside Terraform.",
		Default:     false,
		ForceNew:    true,
	},
	"on_future": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When this is set to true and a schema_name is provided, apply this grant on all future tasks in the given schema. When this is true and no schema_name is provided apply this grant on all future tasks in the given database. The task_name field must be unset in order to use on_future.",
		Default:     false,
		ForceNew:    true,
	},
	"privilege": {
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The privilege to grant on the current or future task.",
		Default:      "USAGE",
		ValidateFunc: validation.StringInSlice(validTaskPrivileges.ToList(), true),
		ForceNew:     true,
	},
	"roles": {
		Type:        schema.TypeSet,
		Required:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: "Grants privilege to these roles.",
	},
	"schema_name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of the schema containing the current or future tasks on which to grant privileges.",
		ForceNew:    true,
	},
	"task_name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of the task on which to grant privileges immediately (only valid if on_future is false).",
		ForceNew:    true,
	},
	"with_grant_option": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When this is set to true, allows the recipient role to grant the privileges to other roles.",
		Default:     false,
		ForceNew:    true,
	},
}

// TaskGrant returns a pointer to the resource representing a task grant.
func TaskGrant() *TerraformGrantResource {
	return &TerraformGrantResource{
		Resource: &schema.Resource{
			Create: CreateTaskGrant,
			Read:   ReadTaskGrant,
			Delete: DeleteTaskGrant,
			Update: UpdateTaskGrant,

			Schema: taskGrantSchema,
			Importer: &schema.ResourceImporter{
				StateContext: schema.ImportStatePassthroughContext,
			},
		},
		ValidPrivs: validTaskPrivileges,
	}
}

// CreateTaskGrant implements schema.CreateFunc.
func CreateTaskGrant(d *schema.ResourceData, meta interface{}) error {
	var taskName string
	if name, ok := d.GetOk("task_name"); ok {
		taskName = name.(string)
	}
	dbName := d.Get("database_name").(string)
	schemaName := d.Get("schema_name").(string)
	priv := d.Get("privilege").(string)
	onFuture := d.Get("on_future").(bool)
	grantOption := d.Get("with_grant_option").(bool)
	roles := expandStringList(d.Get("roles").(*schema.Set).List())

	if (taskName == "") && !onFuture {
		return errors.New("task_name must be set unless on_future is true")
	}
	if (taskName != "") && onFuture {
		return errors.New("task_name must be empty if on_future is true")
	}
	if (schemaName == "") && !onFuture {
		return errors.New("schema_name must be set unless on_future is true")
	}

	var builder snowflake.GrantBuilder
	if onFuture {
		builder = snowflake.FutureTaskGrant(dbName, schemaName)
	} else {
		builder = snowflake.TaskGrant(dbName, schemaName, taskName)
	}

	if err := createGenericGrant(d, meta, builder); err != nil {
		return err
	}

	grant := &grantID{
		ResourceName: dbName,
		SchemaName:   schemaName,
		ObjectName:   taskName,
		Privilege:    priv,
		GrantOption:  grantOption,
		Roles:        roles,
	}
	dataIDInput, err := grant.String()
	if err != nil {
		return err
	}
	d.SetId(dataIDInput)

	return ReadTaskGrant(d, meta)
}

// ReadTaskGrant implements schema.ReadFunc.
func ReadTaskGrant(d *schema.ResourceData, meta interface{}) error {
	grantID, err := grantIDFromString(d.Id())
	if err != nil {
		return err
	}
	dbName := grantID.ResourceName
	schemaName := grantID.SchemaName
	taskName := grantID.ObjectName
	priv := grantID.Privilege

	if err := d.Set("database_name", dbName); err != nil {
		return err
	}

	if err := d.Set("schema_name", schemaName); err != nil {
		return err
	}
	onFuture := false
	if taskName == "" {
		onFuture = true
	}

	if err := d.Set("task_name", taskName); err != nil {
		return err
	}

	if err := d.Set("on_future", onFuture); err != nil {
		return err
	}

	if err := d.Set("privilege", priv); err != nil {
		return err
	}

	if err := d.Set("with_grant_option", grantID.GrantOption); err != nil {
		return err
	}

	var builder snowflake.GrantBuilder
	if onFuture {
		builder = snowflake.FutureTaskGrant(dbName, schemaName)
	} else {
		builder = snowflake.TaskGrant(dbName, schemaName, taskName)
	}

	return readGenericGrant(d, meta, taskGrantSchema, builder, onFuture, validTaskPrivileges)
}

// DeleteTaskGrant implements schema.DeleteFunc.
func DeleteTaskGrant(d *schema.ResourceData, meta interface{}) error {
	grantID, err := grantIDFromString(d.Id())
	if err != nil {
		return err
	}
	dbName := grantID.ResourceName
	schemaName := grantID.SchemaName
	taskName := grantID.ObjectName

	onFuture := (taskName == "")

	var builder snowflake.GrantBuilder
	if onFuture {
		builder = snowflake.FutureTaskGrant(dbName, schemaName)
	} else {
		builder = snowflake.TaskGrant(dbName, schemaName, taskName)
	}
	return deleteGenericGrant(d, meta, builder)
}

// UpdateTaskGrant implements schema.UpdateFunc.
func UpdateTaskGrant(d *schema.ResourceData, meta interface{}) error {
	// for now the only thing we can update are roles or shares
	// if nothing changed, nothing to update and we're done
	if !d.HasChanges("roles") {
		return nil
	}

	rolesToAdd := []string{}
	rolesToRevoke := []string{}

	if d.HasChange("roles") {
		rolesToAdd, rolesToRevoke = changeDiff(d, "roles")
	}

	grantID, err := grantIDFromString(d.Id())
	if err != nil {
		return err
	}

	dbName := grantID.ResourceName
	schemaName := grantID.SchemaName
	taskName := grantID.ObjectName
	onFuture := (taskName == "")

	var builder snowflake.GrantBuilder
	if onFuture {
		builder = snowflake.FutureTaskGrant(dbName, schemaName)
	} else {
		builder = snowflake.TaskGrant(dbName, schemaName, taskName)
	}

	// first revoke
	if err := deleteGenericGrantRolesAndShares(
		meta, builder, grantID.Privilege, rolesToRevoke, []string{},
	); err != nil {
		return err
	}
	// then add
	if err := createGenericGrantRolesAndShares(
		meta, builder, grantID.Privilege, grantID.GrantOption, rolesToAdd, []string{},
	); err != nil {
		return err
	}

	// Done, refresh state
	return ReadTaskGrant(d, meta)
}
