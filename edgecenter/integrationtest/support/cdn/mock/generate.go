package cdnmock

//go:generate go run github.com/vektra/mockery/v2 --name=ResourceService --srcpkg=github.com/Edge-Center/edgecentercdn-go/resources --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=RulesService --srcpkg=github.com/Edge-Center/edgecentercdn-go/rules --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=OriginGroupService --srcpkg=github.com/Edge-Center/edgecentercdn-go/origingroups --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=LECertService --srcpkg=github.com/Edge-Center/edgecentercdn-go/lecerts --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ShieldingService --srcpkg=github.com/Edge-Center/edgecentercdn-go/shielding --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=SSLCertService --srcpkg=github.com/Edge-Center/edgecentercdn-go/sslcerts --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ResourceStatisticsService --srcpkg=github.com/Edge-Center/edgecentercdn-go/statistics --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ResourceToolsService --srcpkg=github.com/Edge-Center/edgecentercdn-go/tools --output=. --outpkg=cdnmock --testonly=false --with-expecter=false --log-level=error
