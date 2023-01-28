package edgecenter

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storage"
)

const (
	StorageS3BucketSchemaName      = "name"
	StorageS3BucketSchemaStorageID = "storage_id"
)

func resourceStorageS3Bucket() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			StorageS3BucketSchemaStorageID: {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "An id of existing storage resource.",
			},
			StorageS3BucketSchemaName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					storageName := i.(string)
					if !regexp.MustCompile(`^[\w\-]+$`).MatchString(storageName) ||
						len(storageName) > 63 ||
						len(storageName) < 3 {
						return diag.Errorf("bucket name can't be empty and can have only letters & numbers. it also should be less than 63 symbols")
					}
					return nil
				},
				Description: "A name of new storage bucket resource.",
			},
		},
		CreateContext: resourceStorageS3BucketCreate,
		ReadContext:   resourceStorageS3BucketRead,
		DeleteContext: resourceStorageS3BucketDelete,
		Description:   "Represent s3 storage bucket resource. https://storage.edgecenter.ru/storage/list",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceStorageS3BucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Get(StorageSchemaID).(int)
	log.Println("[DEBUG] Start S3 Storage Bucket Resource creating")
	defer log.Printf("[DEBUG] Finish S3 Storage Bucket Resource creating (id=%d)\n", id)
	config := m.(*Config)
	client := config.StorageClient

	opts := []func(opt *storage.StorageBucketCreateHTTPParams){
		func(opt *storage.StorageBucketCreateHTTPParams) {
			opt.Context = ctx
			opt.ID = int64(id)
		},
	}
	name := strings.TrimSpace(d.Get(StorageS3BucketSchemaName).(string))
	if name != "" {
		opts = append(opts, func(opt *storage.StorageBucketCreateHTTPParams) { opt.Name = name })
	}

	err := client.CreateBucket(opts...)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create storage bucket: %w", err))
	}
	d.SetId(fmt.Sprintf("%d:%s", id, name))

	return resourceStorageS3BucketRead(ctx, d, m)
}

func resourceStorageS3BucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	storageID, bucketName := storageBucketResourceID(d)
	log.Printf("[DEBUG] Start S3 Storage Bucket Resource reading (id=%d, name=%s)\n", storageID, bucketName)
	defer log.Println("[DEBUG] Finish S3 Storage Bucket Resource reading")

	config := m.(*Config)
	client := config.StorageClient

	opts := []func(opt *storage.StorageListBucketsHTTPParams){
		func(opt *storage.StorageListBucketsHTTPParams) { opt.Context = ctx },
		func(opt *storage.StorageListBucketsHTTPParams) { opt.ID = int64(storageID) },
	}

	result, err := client.BucketsList(opts...)
	if err != nil {
		return diag.FromErr(fmt.Errorf("storage buckets list: %w", err))
	}
	if len(result) == 0 {
		return diag.Errorf("get buckets: wrong length of search result (%d), want more", len(result))
	}
	for _, bucket := range result {
		if bucket.Name == bucketName {
			d.SetId(fmt.Sprintf("%d:%s", storageID, bucketName))
			_ = d.Set(StorageS3BucketSchemaStorageID, storageID)
			_ = d.Set(StorageS3BucketSchemaName, bucketName)
			return nil
		}
	}

	return diag.FromErr(fmt.Errorf("storage buckets list has not this bucket"))
}

func resourceStorageS3BucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	storageID, bucketName := storageBucketResourceID(d)
	log.Printf("[DEBUG] Start S3 Storage Bucket Resource deleting (id=%d,name=%s)\n", storageID, bucketName)
	defer log.Println("[DEBUG] Finish S3 Storage Bucket Resource deleting")
	if bucketName == "" {
		return diag.Errorf("empty bucket")
	}

	config := m.(*Config)
	client := config.StorageClient

	opts := []func(opt *storage.StorageBucketRemoveHTTPParams){
		func(opt *storage.StorageBucketRemoveHTTPParams) { opt.Context = ctx },
		func(opt *storage.StorageBucketRemoveHTTPParams) { opt.ID = int64(storageID) },
		func(opt *storage.StorageBucketRemoveHTTPParams) { opt.Name = bucketName },
	}
	err := client.DeleteBucket(opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func storageBucketResourceID(d *schema.ResourceData) (int, string) {
	var storageID int
	var bucketName string
	resourceID := d.Id()
	if resourceID == "" {
		storageID = d.Get(StorageS3BucketSchemaStorageID).(int)
		bucketName = strings.TrimSpace(d.Get(StorageS3BucketSchemaName).(string))
		return storageID, bucketName
	}
	parts := strings.Split(resourceID, ":")
	if len(parts) != 2 {
		return storageID, bucketName
	}
	id, _ := strconv.ParseInt(parts[0], 10, 64)
	storageID = int(id)
	bucketName = parts[1]

	return storageID, bucketName
}
