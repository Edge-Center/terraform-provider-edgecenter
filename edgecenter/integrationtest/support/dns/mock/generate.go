package dnsmock

//go:generate go run github.com/vektra/mockery/v2 --name=DNSClientService --srcpkg=github.com/Edge-Center/terraform-provider-edgecenter/edgecenter --output=. --outpkg=dnsmock --testonly=false --with-expecter=false --log-level=error
