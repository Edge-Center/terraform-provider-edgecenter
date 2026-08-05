package resellermock

//go:generate go run github.com/vektra/mockery/v2 --name=ResellerImageV2Service --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=resellermock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ResellerNetworksService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=resellermock --testonly=false --with-expecter=false --log-level=error
