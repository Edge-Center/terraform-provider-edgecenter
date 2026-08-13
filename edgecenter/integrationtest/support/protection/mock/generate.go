package protectionmock

//go:generate go run github.com/vektra/mockery/v2 --name=ResourcesService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=AliasesService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=OriginsService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=HeadersService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=BlacklistsService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=WhitelistsService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ServicesService --srcpkg=github.com/Edge-Center/edgecenterprotection-go --output=. --outpkg=protectionmock --testonly=false --with-expecter=false --log-level=error
