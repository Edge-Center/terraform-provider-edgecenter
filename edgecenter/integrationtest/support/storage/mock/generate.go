package storagemock

//go:generate go run github.com/vektra/mockery/v2 --name=StorageLocationService --srcpkg=github.com/Edge-Center/terraform-provider-edgecenter/edgecenter --output=. --outpkg=storagemock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=StorageS3Service --srcpkg=github.com/Edge-Center/terraform-provider-edgecenter/edgecenter --output=. --outpkg=storagemock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=StorageBucketService --srcpkg=github.com/Edge-Center/terraform-provider-edgecenter/edgecenter --output=. --outpkg=storagemock --testonly=false --with-expecter=false --log-level=error
