package cloudmock

//go:generate go run github.com/vektra/mockery/v2 --name=NetworksService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=TasksService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ProjectsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=RegionsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=VolumesService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=KeyPairsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ServerGroupsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=LifeCyclePoliciesService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=SubnetworksService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=RoutersService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=FloatingIPsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=SecurityGroupsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=SnapshotsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=ReservedFixedIPsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=L7PoliciesService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=L7RulesService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=PortsService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=LoadbalancersService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
//go:generate go run github.com/vektra/mockery/v2 --name=InstancesService --srcpkg=github.com/Edge-Center/edgecentercloud-go/v2 --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
