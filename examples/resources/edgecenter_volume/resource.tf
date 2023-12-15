# Example 1
resource "edgecenter_volume" "volume1" {
  region_id                = var.region_id
  project_id               = var.project_id
  name                     = "test-volume"
  size                     = 20
  source                   = "new-volume"
  volume_type              = "ssd_hiiops"
  instance_id_to_attach_to = "00000000-0000-0000-0000-000000000000"
  attachment_tag           = "test-tag"
  metadata = {
    "key1" : "value1",
    "key2" : "value2",
  }
}

# Example 2
resource "edgecenter_volume" "volume_image" {
  region_id  = var.region_id
  project_id = var.project_id
  name       = "test-volume-image"
  size       = 20
  source     = "image"
  image_id   = "00000000-0000-0000-0000-000000000000"
}

# Example 3
resource "edgecenter_volume" "volume_snapshot" {
  region_id   = var.region_id
  project_id  = var.project_id
  name        = "test-volume-snapshot"
  size        = 20
  source      = "snapshot"
  snapshot_id = "00000000-0000-0000-0000-000000000000"
}
