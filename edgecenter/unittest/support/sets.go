package support

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func StringSet(values ...string) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}

	return schema.NewSet(schema.HashString, items)
}

func IntSet(values ...int) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}

	return schema.NewSet(schema.HashInt, items)
}

func MapSet(schemaSetFunc schema.SchemaSetFunc, values ...map[string]interface{}) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}

	return schema.NewSet(schemaSetFunc, items)
}

func List(values ...interface{}) []interface{} {
	return values
}


